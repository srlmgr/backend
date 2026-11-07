//nolint:lll // repository setup is clearer on one line
package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/authsessions"
	"github.com/srlmgr/backend/testsupport/testdb"
)

//nolint:gocyclo,funlen,whitespace // integration scenarios share one isolated database
func TestPostgresSessionStore(t *testing.T) {
	pool, cleanup, err := testdb.InitTestDB()
	if err != nil {
		t.Fatalf("initialize test database: %v", err)
	}
	defer cleanup()
	ctx := context.Background()
	store, err := NewPostgresSessionStore(
		ctx, authsessions.New(pool), repository.NewBobTransactionFromPool(pool),
	)
	if err != nil {
		t.Fatal(err)
	}
	restartedPool, err := pgxpool.NewWithConfig(ctx, pool.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer restartedPool.Close()
	other, err := NewPostgresSessionStore(
		ctx, authsessions.New(restartedPool), repository.NewBobTransactionFromPool(restartedPool),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("startup fails without session table", func(t *testing.T) {
		cfg := pool.Config().Copy()
		cfg.ConnConfig.RuntimeParams["search_path"] = "pg_catalog"
		missingPool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer missingPool.Close()
		if _, err := NewPostgresSessionStore(
			ctx, authsessions.New(missingPool), repository.NewBobTransactionFromPool(missingPool),
		); err == nil {
			t.Fatal("expected a missing table error")
		}
	})

	t.Run("persistence and expiry", func(t *testing.T) {
		session := Session{
			ID:          "persistent-session",
			Principal:   Principal{ID: "alice", Roles: []string{"admin"}},
			AccessToken: "access", RefreshToken: "refresh",
			Expiry:    time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond),
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour).Truncate(time.Microsecond),
		}
		if err := store.Put(ctx, session); err != nil {
			t.Fatal(err)
		}
		if err := store.Put(ctx, session); err == nil {
			t.Fatal("duplicate session ID was accepted")
		}
		got, err := other.Get(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Expiry.Equal(session.Expiry) || !got.ExpiresAt.Equal(session.ExpiresAt) {
			t.Fatalf("session expiries = %v, %v", got.Expiry, got.ExpiresAt)
		}
		got.Expiry = session.Expiry
		got.ExpiresAt = session.ExpiresAt
		if !reflect.DeepEqual(got, session) {
			t.Fatalf("session = %#v, want %#v", got, session)
		}
		var rawHash []byte
		if err := pool.QueryRow(ctx,
			"SELECT session_id_hash FROM auth_sessions WHERE session_id_hash = $1",
			sessionHash(session.ID)).Scan(&rawHash); err != nil {
			t.Fatal(err)
		}
		if string(rawHash) == session.ID ||
			!reflect.DeepEqual(rawHash, sessionHash(session.ID)) {
			t.Fatal("session ID was not hashed")
		}
		expired := Session{ID: "expired-session", ExpiresAt: time.Now().Add(-time.Hour)}
		if err := store.Put(ctx, expired); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Get(ctx, expired.ID); !errors.Is(err, errSessionNotFound) {
			t.Fatalf("expired session error = %v", err)
		}
		if count, err := store.CleanupExpired(ctx, 1); err != nil || count != 1 {
			t.Fatalf("cleanup = %d, %v", count, err)
		}
		if err := other.Delete(ctx, session.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Get(ctx, session.ID); !errors.Is(err, errSessionNotFound) {
			t.Fatalf("deleted session error = %v", err)
		}
	})

	t.Run("zero expiry and failed update", func(t *testing.T) {
		session := Session{ID: "no-expiry", Principal: Principal{ID: "bob"}}
		if err := store.Put(ctx, session); err != nil {
			t.Fatal(err)
		}
		got, err := other.Get(ctx, session.ID)
		if err != nil || !got.Expiry.IsZero() || !got.ExpiresAt.IsZero() {
			t.Fatalf("zero expiry session = %#v, %v", got, err)
		}
		expected := errors.New("update failed")
		_, err = store.Update(ctx, session.ID, func(s Session) (Session, bool, error) {
			s.AccessToken = "should-not-save"
			return s, true, expected
		})
		if !errors.Is(err, expected) {
			t.Fatalf("update error = %v", err)
		}
		got, err = store.Get(ctx, session.ID)
		if err != nil || got.AccessToken != "" {
			t.Fatalf("failed update was persisted: %#v, %v", got, err)
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := store.Get(
			canceled,
			session.ID,
		); err == nil ||
			errors.Is(err, errSessionNotFound) {
			t.Fatalf("database error not propagated: %v", err)
		}
	})

	t.Run("refresh is shared across instances", func(t *testing.T) {
		var calls atomic.Int32
		provider := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh",`+
					`"token_type":"Bearer","expires_in":3600}`)
			}),
		)
		defer provider.Close()
		session := Session{
			ID: "refresh-session", Principal: Principal{ID: "carol"},
			RefreshToken: "old-refresh", Expiry: time.Now().Add(-time.Minute),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := store.Put(ctx, session); err != nil {
			t.Fatal(err)
		}
		client := &oidcClient{oauthConfig: &oauth2.Config{
			ClientID: "client", ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: provider.URL, AuthStyle: oauth2.AuthStyleInParams,
			},
		}}
		cfg := &Config{
			IDP: IDPConfig{Enabled: true, RefreshSkew: time.Second, SessionTTL: time.Hour},
		}
		managers := []*manager{
			{cfg: cfg, sessions: store, idpClient: client, logger: log.New()},
			{cfg: cfg, sessions: other, idpClient: client, logger: log.New()},
		}
		results := make(chan error, 2)
		for _, m := range managers {
			go func() {
				refreshed, found, err := m.refreshOnce(ctx, session.ID)
				if err == nil && (!found || refreshed.RefreshToken != "new-refresh") {
					err = fmt.Errorf("unexpected refresh result: found=%v", found)
				}
				results <- err
			}()
		}
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		if calls.Load() != 1 {
			t.Fatalf("provider calls = %d, want 1", calls.Load())
		}
	})

	t.Run("logout cannot be undone by a refresh", func(t *testing.T) {
		session := Session{ID: "logout-race", ExpiresAt: time.Now().Add(time.Hour)}
		if err := store.Put(ctx, session); err != nil {
			t.Fatal(err)
		}
		entered, release := make(chan struct{}), make(chan struct{})
		updateDone, deleteDone := make(chan error, 1), make(chan error, 1)
		go func() {
			_, err := store.Update(ctx, session.ID, func(s Session) (Session, bool, error) {
				close(entered)
				<-release
				s.RefreshToken = "rotated"
				return s, true, nil
			})
			updateDone <- err
		}()
		<-entered
		go func() { deleteDone <- other.Delete(ctx, session.ID) }()
		close(release)
		if err := <-updateDone; err != nil {
			t.Fatal(err)
		}
		if err := <-deleteDone; err != nil {
			t.Fatal(err)
		}
		if _, err := store.Get(ctx, session.ID); !errors.Is(err, errSessionNotFound) {
			t.Fatalf("session exists after logout: %v", err)
		}
		if _, err := store.Update(ctx, session.ID, func(s Session) (Session, bool, error) {
			return s, true, nil
		}); !errors.Is(err, errSessionNotFound) {
			t.Fatalf("update after logout error = %v", err)
		}
	})
}
