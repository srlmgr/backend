package authn

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
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

	principal, err := mapClaimsToPrincipal(claims)
	if err != nil {
		return Principal{}, "", err
	}
	principal.Source = "session"

	return principal, stringFromAny(claims["nonce"]), nil
}
