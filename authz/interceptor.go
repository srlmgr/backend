package authz

import (
	"context"
	"errors"
	"fmt"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/log"
)

type authorizer struct {
	cfg      Config
	logger   *log.Logger
	policies map[string]ProcedurePolicy
	resolver *scopeResolver
	eval     *opaEvaluator
}

type policyInput struct {
	Procedure      string          `json:"procedure"`
	Capability     string          `json:"capability"`
	AllowAnonymous bool            `json:"allowAnonymous"`
	Principal      authn.Principal `json:"principal"`
	Resource       ResourceScope   `json:"resource"`
}

// NewInterceptor creates the authorization interceptor.
//
//nolint:whitespace // multiline signature for line-length compliance
func NewInterceptor(
	ctx context.Context,
	cfg Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
) (connect.Interceptor, error) {
	a := &authorizer{
		cfg:      cfg,
		logger:   logger.Named("authz"),
		policies: defaultProcedurePolicies(),
		resolver: newScopeResolver(pool),
	}

	if !cfg.Enabled {
		return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return next
		}), nil
	}

	evaluator, err := newOPAEvaluator(ctx, cfg)
	if err != nil {
		return nil, err
	}
	a.eval = evaluator

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		//nolint:whitespace // multiline callback signature for line-length compliance
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if err := a.authorize(ctx, req); err != nil {
				code := connect.CodePermissionDenied
				if errors.Is(err, errMissingPrincipal) {
					code = connect.CodeUnauthenticated
				}
				return nil, connect.NewError(code, err)
			}
			return next(ctx, req)
		}
	}), nil
}

var errMissingPrincipal = fmt.Errorf("missing authenticated principal")

func (a *authorizer) authorize(ctx context.Context, req connect.AnyRequest) error {
	policy, ok := a.policies[req.Spec().Procedure]
	if !ok {
		return fmt.Errorf("no authorization policy for procedure %s", req.Spec().Procedure)
	}

	principal, principalFound := authn.PrincipalFromContext(ctx)
	if !principalFound {
		if policy.AllowAnonymous || authn.IsAnonymousProcedure(req.Spec().Procedure) {
			return nil
		}
		return errMissingPrincipal
	}

	resourceScope, err := a.resolver.Resolve(ctx, req, policy)
	if err != nil {
		return fmt.Errorf("resolve authorization scope: %w", err)
	}

	input := policyInput{
		Procedure:      req.Spec().Procedure,
		Capability:     policy.Capability,
		AllowAnonymous: policy.AllowAnonymous,
		Principal:      principal,
		Resource:       resourceScope,
	}

	allowed, err := a.eval.Evaluate(ctx, input)
	if err != nil {
		return err
	}
	if !allowed {
		a.logger.Warn("authorization denied",
			log.String("procedure", req.Spec().Procedure),
			log.String("capability", policy.Capability),
			log.String("principal_id", principal.ID),
		)
		return fmt.Errorf("access denied")
	}

	return nil
}
