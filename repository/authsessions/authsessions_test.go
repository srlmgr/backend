//nolint:lll,govet // integration cases share a database and exercise transaction errors
package authsessions_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/authsessions"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/testsupport/testdb"
)

func sessionInput(id string, expiresAt time.Time) *models.AuthSessionSetter {
	hash := sha256.Sum256([]byte(id))
	setter := &models.AuthSessionSetter{
		SessionIDHash: omit.From(hash[:]),
		Principal: omit.From(types.NewJSON(
			json.RawMessage(`{"id":"user"}`),
		)),
		AccessToken:  omit.From("access"),
		RefreshToken: omit.From("refresh"),
	}
	if !expiresAt.IsZero() {
		setter.ExpiresAt = omitnull.From(expiresAt)
	}
	return setter
}

//nolint:gocyclo,funlen // CRUD and transaction checks share one isolated database.
func TestRepository(t *testing.T) {
	pool, cleanup, err := testdb.InitTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	ctx := context.Background()
	repo := authsessions.New(pool)
	tx := repository.NewBobTransactionFromPool(pool)
	if err := repo.Check(ctx); err != nil {
		t.Fatal(err)
	}
	input := sessionInput("valid", time.Now().Add(time.Hour))
	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	hash := input.SessionIDHash.MustGet()
	if _, err := repo.Create(ctx, input); err == nil {
		t.Fatal("duplicate session was accepted")
	}
	loaded, err := repo.LoadByHash(ctx, hash, false)
	if err != nil || loaded.ID != created.ID {
		t.Fatalf("load = %#v, %v", loaded, err)
	}
	if _, err := repo.LoadByHash(ctx, hash, true); err == nil {
		t.Fatal("lock without transaction succeeded")
	}

	expected := errors.New("rollback")
	err = tx.RunInTx(ctx, func(txCtx context.Context) error {
		locked, err := repo.LoadByHash(txCtx, hash, true)
		if err != nil {
			return err
		}
		_, err = repo.Update(txCtx, locked.ID, &models.AuthSessionSetter{
			AccessToken: omit.From("rolled-back"),
		})
		if err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("transaction error = %v", err)
	}
	loaded, err = repo.LoadByHash(ctx, hash, false)
	if err != nil || loaded.AccessToken != "access" {
		t.Fatalf("rollback load = %#v, %v", loaded, err)
	}
	updated, err := repo.Update(ctx, created.ID, &models.AuthSessionSetter{
		AccessToken: omit.From("updated"),
	})
	if err != nil || updated.AccessToken != "updated" {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	if _, err := repo.Update(ctx, -1, &models.AuthSessionSetter{
		AccessToken: omit.From("missing"),
	}); !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
	if err := repo.DeleteByHash(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.LoadByHash(ctx, hash, false); !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("deleted session error = %v", err)
	}
	if err := repo.DeleteByHash(ctx, hash); err != nil {
		t.Fatal(err)
	}
}

//nolint:gocyclo,funlen // cleanup scenarios share one isolated database
func TestRepositoryCleanupExpired(t *testing.T) {
	pool, cleanup, err := testdb.InitTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	ctx := context.Background()
	repo := authsessions.New(pool)
	expired := []string{"expired-a", "expired-b"}
	for _, id := range expired {
		if _, err := repo.Create(ctx, sessionInput(id, time.Now().Add(-time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	lockedHash := sessionInput(expired[0], time.Time{}).SessionIDHash.MustGet()
	if _, err := tx.Exec(ctx,
		"SELECT id FROM auth_sessions WHERE session_id_hash = $1 FOR UPDATE",
		lockedHash); err != nil {
		t.Fatal(err)
	}
	count, err := repo.CleanupExpired(ctx, 10)
	if err != nil || count != 1 {
		t.Fatalf("cleanup with locked row = %d, %v", count, err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	active := sessionInput("active", time.Now().Add(time.Hour))
	if _, err := repo.Create(ctx, active); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CleanupExpired(ctx, 0); err == nil {
		t.Fatal("zero cleanup limit was accepted")
	}
	count, err = repo.CleanupExpired(ctx, 1)
	if err != nil || count != 1 {
		t.Fatalf("cleanup = %d, %v", count, err)
	}
	if count, err := repo.CleanupExpired(ctx, 1); err != nil || count != 0 {
		t.Fatalf("final cleanup = %d, %v", count, err)
	}
	if _, err := repo.LoadByHash(ctx, active.SessionIDHash.MustGet(), false); err != nil {
		t.Fatalf("active session removed: %v", err)
	}
}
