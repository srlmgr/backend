package authn

import (
	"context"
	"crypto/sha256"
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
	// Put creates a session and must not overwrite an existing session ID.
	Put(ctx context.Context, session Session) error
	Get(ctx context.Context, sessionID string) (Session, error)
	Delete(ctx context.Context, sessionID string) error
	// Update locks and re-reads a session, then atomically saves or deletes it.
	// A false return from fn deletes the session; an error leaves it unchanged.
	Update(
		ctx context.Context,
		sessionID string,
		fn func(Session) (Session, bool, error),
	) (Session, error)
}

type inMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	locks    [64]sync.Mutex
}

func newInMemorySessionStore() SessionStore {
	return &inMemorySessionStore{
		sessions: make(map[string]Session),
	}
}

//nolint:gocritic // session is intentionally copied into store
func (s *inMemorySessionStore) Put(_ context.Context, session Session) error {
	lock := s.sessionLock(session.ID)
	lock.Lock()
	defer lock.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; exists {
		return errors.New("session already exists")
	}
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
	lock := s.sessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *inMemorySessionStore) sessionLock(sessionID string) *sync.Mutex {
	hash := sha256.Sum256([]byte(sessionID))
	return &s.locks[int(hash[0])%len(s.locks)]
}

//nolint:whitespace // editor/linter issue
func (s *inMemorySessionStore) Update(
	ctx context.Context,
	sessionID string,
	fn func(Session) (Session, bool, error),
) (Session, error) {
	lock := s.sessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	updated, keep, err := fn(session)
	if err != nil {
		return Session{}, err
	}
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !keep {
		delete(s.sessions, sessionID)
		return Session{}, errSessionNotFound
	}
	updated.ID = sessionID
	s.sessions[sessionID] = updated
	return updated, nil
}
