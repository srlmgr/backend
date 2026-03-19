package authn

import (
	"context"
	"errors"
	"fmt"

	connect "connectrpc.com/connect"

	"github.com/srlmgr/backend/log"
)

type authenticator struct {
	jwt      *jwtValidator
	apiToken *apiTokenStore
	logger   *log.Logger
}

// NewInterceptor creates the authentication interceptor.
//
//nolint:gocritic,whitespace // startup config copied intentionally
func NewInterceptor(
	ctx context.Context,
	cfg Config,
	l *log.Logger,
) (connect.Interceptor, error) {
	a := &authenticator{logger: l.Named("authn")}
	if !cfg.Enabled {
		return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return next
		}), nil
	}

	jwtValidator, err := newJWTValidator(ctx, cfg.JWT)
	if err != nil {
		return nil, err
	}
	a.jwt = jwtValidator

	store, err := newAPITokenStore(ctx, cfg.APIToken, a.logger)
	if err != nil {
		return nil, err
	}
	a.apiToken = store

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		//nolint:whitespace // multiline callback signature for line-length compliance
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			principalCtx, err := a.authenticate(ctx, req)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			return next(principalCtx, req)
		}
	}), nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (a *authenticator) authenticate(
	ctx context.Context,
	req connect.AnyRequest,
) (context.Context, error) {
	cred, err := selectCredential(req.Header())
	if err != nil {
		a.logger.Warn(
			"invalid authentication metadata",
			log.String("procedure", req.Spec().Procedure),
		)
		return nil, err
	}

	if cred.source == authSourceNone {
		if IsAnonymousProcedure(req.Spec().Procedure) {
			return ctx, nil
		}
		return nil, errors.New("missing authentication credentials")
	}

	principal, err := a.validateCredential(ctx, cred)
	if err != nil {
		a.logger.Warn("authentication failed",
			log.String("procedure", req.Spec().Procedure),
			log.String("source", authSourceName(cred.source)),
		)
		return nil, err
	}

	return AddPrincipal(ctx, &principal), nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (a *authenticator) validateCredential(
	ctx context.Context,
	cred selectedCredential,
) (Principal, error) {
	switch cred.source {
	case authSourceNone:
		return Principal{}, fmt.Errorf("no authentication credentials supplied")
	case authSourceJWT:
		if a.jwt == nil {
			return Principal{}, fmt.Errorf("jwt authentication is disabled")
		}
		return a.jwt.validate(ctx, cred.token)
	case authSourceAPIToken:
		if a.apiToken == nil {
			return Principal{}, fmt.Errorf("api-token authentication is disabled")
		}
		return a.apiToken.validate(cred.token)
	default:
		return Principal{}, fmt.Errorf("no supported authentication source found")
	}
}

func authSourceName(source authSource) string {
	switch source {
	case authSourceNone:
		return "none"
	case authSourceJWT:
		return "jwt"
	case authSourceAPIToken:
		return "api-token"
	default:
		return "none"
	}
}
