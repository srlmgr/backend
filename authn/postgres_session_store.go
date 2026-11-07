//nolint:lll,whitespace // interface signatures and multiline declarations
package authn

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/authsessions"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// PostgresSessionStore persists sessions using the authentication session repository.
type PostgresSessionStore struct {
	sessions authsessions.Repository
	tx       repository.TransactionManager
}

// NewPostgresSessionStore verifies the session repository before accepting requests.
func NewPostgresSessionStore(
	ctx context.Context, sessions authsessions.Repository, tx repository.TransactionManager,
) (*PostgresSessionStore, error) {
	if sessions == nil || tx == nil {
		return nil, errors.New("session repository and transaction manager are required")
	}
	if err := sessions.Check(ctx); err != nil {
		return nil, fmt.Errorf("check session repository: %w", err)
	}
	return &PostgresSessionStore{sessions: sessions, tx: tx}, nil
}

func sessionHash(id string) []byte {
	hash := sha256.Sum256([]byte(id))
	return hash[:]
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func sessionSetter(session *Session) (*models.AuthSessionSetter, error) {
	principal, err := json.Marshal(session.Principal)
	if err != nil {
		return nil, fmt.Errorf("encode session principal: %w", err)
	}
	return &models.AuthSessionSetter{
		SessionIDHash:  omit.From(sessionHash(session.ID)),
		Principal:      omit.From(types.NewJSON(json.RawMessage(principal))),
		AccessToken:    omit.From(session.AccessToken),
		RefreshToken:   omit.From(session.RefreshToken),
		TokenExpiresAt: omitnull.FromPtr(nullableTime(session.Expiry)),
		ExpiresAt:      omitnull.FromPtr(nullableTime(session.ExpiresAt)),
	}, nil
}

func sessionFromModel(id string, model *models.AuthSession) (Session, error) {
	session := Session{
		ID:           id,
		AccessToken:  model.AccessToken,
		RefreshToken: model.RefreshToken,
		Expiry:       model.TokenExpiresAt.GetOrZero(),
		ExpiresAt:    model.ExpiresAt.GetOrZero(),
	}
	if err := json.Unmarshal(model.Principal.Val, &session.Principal); err != nil {
		return Session{}, fmt.Errorf("decode session principal: %w", err)
	}
	return session, nil
}

// Put creates a session. Existing session IDs are never overwritten.
//
//nolint:gocritic // SessionStore accepts sessions by value.
func (s *PostgresSessionStore) Put(ctx context.Context, session Session) error {
	setter, err := sessionSetter(&session)
	if err != nil {
		return err
	}
	if _, err := s.sessions.Create(ctx, setter); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *PostgresSessionStore) get(
	ctx context.Context, sessionID string, forUpdate bool,
) (Session, *models.AuthSession, error) {
	model, err := s.sessions.LoadByHash(ctx, sessionHash(sessionID), forUpdate)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return Session{}, nil, errSessionNotFound
	}
	if err != nil {
		return Session{}, nil, fmt.Errorf("read session: %w", err)
	}
	session, err := sessionFromModel(sessionID, model)
	return session, model, err
}

// Get returns an unexpired session or a session-not-found error.
func (s *PostgresSessionStore) Get(ctx context.Context, sessionID string) (Session, error) {
	session, _, err := s.get(ctx, sessionID, false)
	return session, err
}

// Delete invalidates a session; deleting an absent session succeeds.
func (s *PostgresSessionStore) Delete(ctx context.Context, sessionID string) error {
	if err := s.sessions.DeleteByHash(ctx, sessionHash(sessionID)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Update serializes a read-modify-write with updates and deletes across instances.
func (s *PostgresSessionStore) Update(
	ctx context.Context, sessionID string, fn func(Session) (Session, bool, error),
) (Session, error) {
	var updated Session
	var keep bool
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		session, model, err := s.get(txCtx, sessionID, true)
		if err != nil {
			return err
		}
		updated, keep, err = fn(session)
		if err != nil {
			return err
		}
		if !keep {
			return s.sessions.DeleteByHash(txCtx, sessionHash(sessionID))
		}
		updated.ID = sessionID
		setter, err := sessionSetter(&updated)
		if err != nil {
			return err
		}
		setter.SessionIDHash = omit.Val[[]byte]{}
		setter.UpdatedAt = omit.From(time.Now())
		_, err = s.sessions.Update(txCtx, model.ID, setter)
		return err
	})
	if err != nil {
		return Session{}, fmt.Errorf("update session: %w", err)
	}
	if !keep {
		return Session{}, errSessionNotFound
	}
	return updated, nil
}

// CleanupExpired deletes at most limit expired sessions.
func (s *PostgresSessionStore) CleanupExpired(ctx context.Context, limit int) (int64, error) {
	count, err := s.sessions.CleanupExpired(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return count, nil
}
