package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	connect "connectrpc.com/connect"

	"github.com/srlmgr/backend/log"
)

const (
	authorizationHeader = "Authorization"
	apiTokenHeader      = "api-token"
	bearerPrefix        = "Bearer "
)

// Authenticator validates request credentials and produces a Principal.
type Authenticator struct {
	jwt    *jwtValidator
	tokens *apiTokenStore
	logger *log.Logger
}

// NewAuthenticator creates an Authenticator from Config.
// ctx is used for JWKS initialization and background refresh.
//nolint:whitespace //editor/linter issue
func NewAuthenticator(
	ctx context.Context,
	cfg Config,
	logger *log.Logger,
) (*Authenticator, error) {
	a := &Authenticator{logger: logger}
	if cfg.JWT.JWKSUrl != "" {
		v, err := newJWTValidator(ctx, cfg.JWT, logger.Named("jwks"))
		if err != nil {
			return nil, fmt.Errorf("create JWT validator: %w", err)
		}
		a.jwt = v
	}
	if cfg.APIToken.FilePath != "" {
		s, err := newAPITokenStore(ctx, cfg.APIToken, logger.Named("apitoken"))
		if err != nil {
			return nil, fmt.Errorf("create api-token store: %w", err)
		}
		a.tokens = s
	}
	return a, nil
}

// NewInterceptor returns a Connect-RPC unary interceptor that enforces
// authentication.
func (a *Authenticator) NewInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			principal, err := a.AuthenticateHeaders(ctx, req.Spec().Procedure, req.Header())
			if err != nil {
				a.logger.Warn("authentication failed",
					log.String("procedure", req.Spec().Procedure),
					log.ErrorField(err),
				)
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("unauthenticated"),
				)
			}
			if principal != nil {
				ctx = WithPrincipal(ctx, principal)
			}
			return next(ctx, req)
		}
	})
}

// AuthenticateHeaders validates the credentials in the given HTTP headers for
// the named procedure and returns a Principal on success.
// It returns (nil, nil) for anonymous procedures with no credentials.
func (a *Authenticator) AuthenticateHeaders(
	ctx context.Context,
	procedure string,
	header http.Header,
) (*Principal, error) {
	authHeader := header.Get(authorizationHeader)
	apiToken := header.Get(apiTokenHeader)

	hasAuth := authHeader != ""
	hasAPIToken := apiToken != ""

	if hasAuth && hasAPIToken {
		return nil, errors.New(
			"ambiguous authentication: both Authorization and api-token present",
		)
	}

	if !hasAuth && !hasAPIToken {
		if IsAnonymousProcedure(procedure) {
			return nil, nil
		}
		return nil, errors.New("missing credentials")
	}

	if hasAuth {
		return a.validateJWT(ctx, authHeader)
	}
	return a.validateAPIToken(apiToken)
}

//nolint:whitespace //editor/linter issue
func (a *Authenticator) validateJWT(
	_ context.Context,
	authHeader string,
) (*Principal, error) {
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, errors.New("Authorization header must use Bearer scheme")
	}
	tokenStr := strings.TrimPrefix(authHeader, bearerPrefix)
	if strings.TrimSpace(tokenStr) == "" {
		return nil, errors.New("empty bearer token")
	}
	if a.jwt == nil {
		return nil, errors.New("JWT authentication not configured")
	}
	p, err := a.jwt.Validate(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("JWT validation: %w", err)
	}
	return p, nil
}

func (a *Authenticator) validateAPIToken(token string) (*Principal, error) {
	if a.tokens == nil {
		return nil, errors.New("api-token authentication not configured")
	}
	p, err := a.tokens.Validate(token)
	if err != nil {
		return nil, fmt.Errorf("api-token validation: %w", err)
	}
	return p, nil
}
