package authn

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/srlmgr/backend/log"
)

// jwksKey represents a single JWK entry.
type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

// jwksResponse is the JSON structure of a JWKS endpoint response.
type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// jwksCache holds the cached public keys from a JWKS endpoint.
type jwksCache struct {
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	url        string
	refresher  time.Duration
	httpClient *http.Client
	logger     *log.Logger
}

// newJWKSCache creates a new JWKS cache and starts background refresh.
//
//nolint:whitespace //editor/linter issue
func newJWKSCache(
	ctx context.Context,
	url string,
	refresh time.Duration,
	logger *log.Logger,
) (*jwksCache, error) {
	c := &jwksCache{
		keys:       make(map[string]*rsa.PublicKey),
		url:        url,
		refresher:  refresh,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
	if err := c.refresh(ctx); err != nil {
		return nil, fmt.Errorf("initial JWKS fetch: %w", err)
	}
	if refresh > 0 {
		go c.backgroundRefresh(ctx)
	}
	return c, nil
}

// getKey returns the RSA public key for a given key ID.
func (c *jwksCache) getKey(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

func (c *jwksCache) backgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(c.refresher)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.refresh(ctx); err != nil {
				c.logger.Warn("JWKS background refresh failed", log.ErrorField(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *jwksCache) refresh(ctx context.Context) error {
	//nolint:gosec // URL comes from trusted configuration
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read JWKS response: %w", err)
	}
	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("parse JWKS response: %w", err)
	}
	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := parseRSAPublicKey(k)
		if err != nil {
			return fmt.Errorf("parse key %s: %w", k.Kid, err)
		}
		newKeys[k.Kid] = pub
	}
	c.mu.Lock()
	c.keys = newKeys
	c.mu.Unlock()
	return nil
}

func parseRSAPublicKey(k jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}
	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}, nil
}
