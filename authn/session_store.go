package authn

import (
	"context"
	"errors"
	"sync"
	"time"
)

var errSessionNotFound = errors.New("session not found")

type Session struct {
	ID           string
	Principal    Principal
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	ExpiresAt    time.Time
}

type SessionStore interface {
	Put(ctx context.Context, session Session) error
	Get(ctx context.Context, sessionID string) (Session, error)
	Delete(ctx context.Context, sessionID string) error
}

type inMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func newInMemorySessionStore() SessionStore {
	return &inMemorySessionStore{
		sessions: make(map[string]Session),
	}
}

//nolint:gocritic // session is intentionally copied into store
func (s *inMemorySessionStore) Put(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	return nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (s *inMemorySessionStore) Get(
	_ context.Context,
	sessionID string,
) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return Session{}, errSessionNotFound
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return Session{}, errSessionNotFound
	}

	return session, nil
}

func (s *inMemorySessionStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}
