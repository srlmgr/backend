package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	connect "connectrpc.com/connect/v2"

	"github.com/srlmgr/backend/log"
)

type failingSessionStore struct{ SessionStore }

func (s failingSessionStore) Delete(context.Context, string) error {
	return errors.New("storage unavailable")
}

func (s failingSessionStore) Get(context.Context, string) (Session, error) {
	return Session{}, errors.New("storage unavailable")
}

func TestSessionStoreErrorsAreInternal(t *testing.T) {
	m := &manager{
		cfg:      &Config{},
		logger:   log.New(),
		sessions: failingSessionStore{newInMemorySessionStore()},
	}
	_, _, err := m.sessionFromID(context.Background(), "id")
	if err == nil {
		t.Fatal("expected storage error")
	}
	var connectErr *connect.Error
	if !errors.As(m.toConnectError(err, nil), &connectErr) {
		t.Fatal("expected a connect error")
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Fatalf("unexpected connect error: %v", err)
	}
}

func TestLogoutReportsStoreFailure(t *testing.T) {
	m := &manager{
		cfg:        &Config{},
		logger:     log.New(),
		cookieName: "backend_session",
		sessions:   failingSessionStore{newInMemorySessionStore()},
	}
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/logout", http.NoBody,
	)
	req.AddCookie(&http.Cookie{
		Name: "backend_session", Value: "session", Secure: true,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	recorder := httptest.NewRecorder()
	m.handleLogout(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("cookie was cleared despite failed session invalidation")
	}
}

func TestInMemorySessionUpdate(t *testing.T) {
	ctx := context.Background()
	store := newInMemorySessionStore()
	session := Session{ID: "id", ExpiresAt: time.Now().Add(time.Hour)}
	if err := store.Put(ctx, session); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, session); err == nil {
		t.Fatal("duplicate session ID was accepted")
	}
	updated, err := store.Update(ctx, session.ID, func(s Session) (Session, bool, error) {
		s.AccessToken = "updated"
		return s, true, nil
	})
	if err != nil || updated.AccessToken != "updated" {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	_, err = store.Update(ctx, session.ID, func(s Session) (Session, bool, error) {
		return s, false, nil
	})
	if !errors.Is(err, errSessionNotFound) {
		t.Fatalf("delete update error = %v", err)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("deleted session error = %v", err)
	}
}
