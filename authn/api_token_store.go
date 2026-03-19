package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/srlmgr/backend/log"
)

type apiTokenStore struct {
	path   string
	mu     sync.RWMutex
	tokens map[string]Principal
	logger *log.Logger
}

type apiTokenFile struct {
	Tokens []apiTokenRecord `json:"tokens"`
}

type apiTokenRecord struct {
	Token  string         `json:"token"`
	Active bool           `json:"active"`
	ID     string         `json:"id"`
	Tenant string         `json:"tenant"`
	Roles  []string       `json:"roles"`
	Scope  apiTokenScopes `json:"scope"`
}

type apiTokenScopes struct {
	SimulationIDs []string `json:"simulationIDs"`
	SeriesIDs     []string `json:"seriesIDs"`
}

//nolint:whitespace // multiline signature for line-length compliance
func newAPITokenStore(
	ctx context.Context,
	cfg APITokenConfig,
	logger *log.Logger,
) (*apiTokenStore, error) {
	if cfg.FilePath == "" {
		return nil, nil
	}

	store := &apiTokenStore{
		path:   cfg.FilePath,
		tokens: make(map[string]Principal),
		logger: logger,
	}

	if err := store.reload(); err != nil {
		return nil, err
	}

	if cfg.RefreshInterval > 0 {
		go store.watch(ctx, cfg.RefreshInterval)
	}

	return store, nil
}

func (s *apiTokenStore) validate(token string) (Principal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	principal, ok := s.tokens[token]
	if !ok {
		return Principal{}, fmt.Errorf("api-token not found")
	}
	return principal, nil
}

func (s *apiTokenStore) watch(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.reload(); err != nil {
				s.logger.Error(
					"failed to reload api-token file",
					log.ErrorField(err),
					log.String("path", s.path),
				)
			}
		}
	}
}

func (s *apiTokenStore) reload() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read api-token file: %w", err)
	}

	var payload apiTokenFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decode api-token file: %w", err)
	}

	next := make(map[string]Principal, len(payload.Tokens))
	for i := range payload.Tokens {
		record := payload.Tokens[i]
		if !record.Active {
			continue
		}
		if record.Token == "" {
			return fmt.Errorf("api-token entry has empty token")
		}
		if _, exists := next[record.Token]; exists {
			return fmt.Errorf("api-token entry duplicated")
		}
		if record.ID == "" || record.Tenant == "" {
			return fmt.Errorf("api-token entry missing required id or tenant")
		}

		next[record.Token] = Principal{
			ID:            record.ID,
			Tenant:        record.Tenant,
			Roles:         append([]string(nil), record.Roles...),
			SimulationIDs: append([]string(nil), record.Scope.SimulationIDs...),
			SeriesIDs:     append([]string(nil), record.Scope.SeriesIDs...),
			Source:        "api-token",
		}
	}

	s.mu.Lock()
	s.tokens = next
	s.mu.Unlock()

	return nil
}
