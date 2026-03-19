package authn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/srlmgr/backend/log"
)

// apiTokenEntry represents one entry in the api-token file.
type apiTokenEntry struct {
	Token  string         `yaml:"token"`
	ID     string         `yaml:"id"`
	Tenant string         `yaml:"tenant"`
	Active bool           `yaml:"active"`
	Roles  []string       `yaml:"roles"`
	Scopes apiTokenScopes `yaml:"scopes"`
}

type apiTokenScopes struct {
	SimulationIDs []string `yaml:"simulationIDs"`
	SeriesIDs     []string `yaml:"seriesIDs"`
}

// apiTokenFile is the top-level structure of the token YAML file.
type apiTokenFile struct {
	Tokens []apiTokenEntry `yaml:"tokens"`
}

// apiTokenStore validates api-tokens against a filesystem token store.
type apiTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]apiTokenEntry
	cfg    APITokenConfig
	logger *log.Logger
}

// newAPITokenStore loads the token file and optionally starts background reload.
func newAPITokenStore(
	ctx context.Context,
	cfg APITokenConfig,
	logger *log.Logger,
) (*apiTokenStore, error) {
	s := &apiTokenStore{cfg: cfg, logger: logger}
	if err := s.reload(); err != nil {
		return nil, fmt.Errorf("load api-token file: %w", err)
	}
	if cfg.ReloadInterval > 0 {
		go s.backgroundReload(ctx)
	}
	return s, nil
}

// errTokenInactive is returned when a token is found but marked inactive.
var errTokenInactive = errors.New("api-token is inactive")

// Validate looks up the token and returns a Principal if valid.
func (s *apiTokenStore) Validate(token string) (*Principal, error) {
	s.mu.RLock()
	entry, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("api-token not found")
	}
	if !entry.Active {
		return nil, errTokenInactive
	}
	return &Principal{
		ID:     entry.ID,
		Tenant: entry.Tenant,
		Roles:  entry.Roles,
		Scopes: Scopes{
			SimulationIDs: entry.Scopes.SimulationIDs,
			SeriesIDs:     entry.Scopes.SeriesIDs,
		},
	}, nil
}

func (s *apiTokenStore) reload() error {
	data, err := os.ReadFile(s.cfg.FilePath)
	if err != nil {
		return fmt.Errorf("read token file: %w", err)
	}
	var f apiTokenFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse token file: %w", err)
	}
	newTokens := make(map[string]apiTokenEntry, len(f.Tokens))
	for _, t := range f.Tokens {
		if t.Token == "" {
			return fmt.Errorf("token entry missing token value")
		}
		if _, dup := newTokens[t.Token]; dup {
			return fmt.Errorf("duplicate token entry: %s", t.Token)
		}
		newTokens[t.Token] = t
	}
	s.mu.Lock()
	s.tokens = newTokens
	s.mu.Unlock()
	return nil
}

func (s *apiTokenStore) backgroundReload(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.ReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.reload(); err != nil {
				s.logger.Warn("api-token file reload failed", log.ErrorField(err))
			}
		case <-ctx.Done():
			return
		}
	}
}
