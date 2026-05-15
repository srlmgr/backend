package authn

import (
	"errors"
	"sync"
	"time"
)

var errLoginStateNotFound = errors.New("login state not found")

type loginState struct {
	Nonce     string
	ExpiresAt time.Time
}

type loginStateStore struct {
	mu     sync.Mutex
	states map[string]loginState
}

func newLoginStateStore() *loginStateStore {
	return &loginStateStore{states: make(map[string]loginState)}
}

func (s *loginStateStore) Put(state string, value loginState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = value
}

func (s *loginStateStore) Consume(state string) (loginState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.states[state]
	if !ok {
		return loginState{}, errLoginStateNotFound
	}
	delete(s.states, state)

	if !current.ExpiresAt.IsZero() && time.Now().After(current.ExpiresAt) {
		return loginState{}, errLoginStateNotFound
	}

	return current, nil
}
