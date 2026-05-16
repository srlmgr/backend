package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/srlmgr/backend/log"
	"golang.org/x/oauth2"
)

type tokenBundle struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Principal    Principal
	Nonce        string
}

type oidcClient struct {
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

type (
	//nolint:tagliatelle // external API
	KeycloakClaims struct {
		RealmAccess RealmAccess `json:"realm_access"`
	}
	RealmAccess struct {
		Roles []string `json:"roles"`
	}
)

func newOIDCClient(ctx context.Context, cfg *IDPConfig) (*oidcClient, error) {
	if cfg == nil {
		return nil, nil
	}
	if !cfg.Enabled {
		return nil, nil
	}
	missingRequired := strings.TrimSpace(cfg.IssuerURL) == "" ||
		strings.TrimSpace(cfg.ClientID) == "" ||
		strings.TrimSpace(cfg.ClientSecret) == "" ||
		strings.TrimSpace(cfg.CallbackURL) == ""
	if missingRequired {
		return nil, fmt.Errorf(
			"idp requires issuer-url, client-id, client-secret and callback-url",
		)
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	callbackURL, err := url.Parse(cfg.CallbackURL)
	if err != nil {
		return nil, fmt.Errorf("parse callback url: %w", err)
	}
	if callbackURL.Scheme == "" || callbackURL.Host == "" {
		return nil, fmt.Errorf("callback url must be an absolute URL")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	return &oidcClient{oauthConfig: oauthConfig, verifier: verifier}, nil
}

func (c *oidcClient) authCodeURL(state, nonce string) string {
	if c == nil {
		return ""
	}
	return c.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("nonce", nonce),
	)
}

func (c *oidcClient) exchange(ctx context.Context, code string) (tokenBundle, error) {
	token, err := c.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return tokenBundle{}, fmt.Errorf("exchange auth code: %w", err)
	}

	bundle := tokenBundle{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	if bundle.RefreshToken == "" {
		return tokenBundle{}, fmt.Errorf("idp did not return a refresh token")
	}

	principal, nonce, err := c.principalFromIDToken(ctx, token)
	if err != nil {
		return tokenBundle{}, err
	}
	bundle.Principal = principal
	bundle.Nonce = nonce
	return bundle, nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (c *oidcClient) refresh(
	ctx context.Context,
	refreshToken string,
) (tokenBundle, error) {
	token, err := c.oauthConfig.TokenSource(
		ctx,
		&oauth2.Token{RefreshToken: refreshToken},
	).Token()
	if err != nil {
		return tokenBundle{}, fmt.Errorf("refresh token: %w", err)
	}

	bundle := tokenBundle{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshToken,
		Expiry:       token.Expiry,
	}
	if token.RefreshToken != "" {
		bundle.RefreshToken = token.RefreshToken
	}

	principal, _, err := c.principalFromIDToken(ctx, token)
	if err == nil {
		bundle.Principal = principal
	}

	return bundle, nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (c *oidcClient) principalFromIDToken(
	ctx context.Context,
	token *oauth2.Token,
) (Principal, string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return Principal{}, "", fmt.Errorf("idp did not return an id_token")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Principal{}, "", fmt.Errorf("verify id token: %w", err)
	}

	claims := map[string]any{}
	claimsErr := idToken.Claims(&claims)
	if claimsErr != nil {
		return Principal{}, "", fmt.Errorf("decode id token claims: %w", claimsErr)
	}
	if kcClaims, kcErr := c.extractKeycloakClaims(token.AccessToken); kcErr == nil {
		claims["roles"] = kcClaims.RealmAccess.Roles
	}

	principal, err := mapClaimsToPrincipal(claims)
	if err != nil {
		return Principal{}, "", err
	}
	log.Debug("claims", log.Any("claims", claims))
	log.Debug("mapped id token claims to principal", log.Any("principal", principal))
	principal.Source = "session"

	return principal, stringFromAny(claims["nonce"]), nil
}

//nolint:whitespace // editor/linter issue
func (c *oidcClient) extractKeycloakClaims(accessToken string) (
	ret *KeycloakClaims,
	err error,
) {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 3 {
		log.Warn("invalid access token format")
		return nil, fmt.Errorf("invalid access token format")
	}
	var payload []byte
	payload, err = base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	claims := KeycloakClaims{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		log.Warn("failed to unmarshal access token claims", log.ErrorField(err))
		return nil, fmt.Errorf("failed to unmarshal access token claims: %w", err)
	}
	return &claims, nil
}
