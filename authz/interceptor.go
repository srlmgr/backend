package authz

import (
	"context"
	"errors"
	"fmt"

	connect "connectrpc.com/connect/v2"
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
	Procedure      string           `json:"procedure"`
	Capability     string           `json:"capability"`
	AllowAnonymous bool             `json:"allowAnonymous"`
	Principal      *authn.Principal `json:"principal"`
	Resource       ResourceScope    `json:"resource"`
}

// NewInterceptor creates the authorization interceptor.
//
//nolint:whitespace // multiline signature for line-length compliance
func NewInterceptor(
	ctx context.Context,
	cfg Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
) (connect.ServerInterceptor, error) {
	a := &authorizer{
		cfg:      cfg,
		logger:   logger.Named("authz"),
		policies: defaultProcedurePolicies(),
		resolver: newScopeResolver(pool),
	}

	if !cfg.Enabled {
		return func(next connect.ServerFunc) connect.ServerFunc {
			return next
		}, nil
	}

	evaluator, err := newOPAEvaluator(ctx, cfg)
	if err != nil {
		return nil, err
	}
	a.eval = evaluator

	return func(next connect.ServerFunc) connect.ServerFunc {
		//nolint:whitespace // multiline callback signature for line-length compliance
		return func(
			ctx context.Context,
			spec connect.Spec,
			stream connect.ServerStream,
		) error {
			if spec.StreamType != connect.StreamTypeUnary {
				return next(ctx, spec, stream)
			}
			return next(ctx, spec, &authorizingStream{
				ServerStream: stream,
				ctx:          ctx,
				procedure:    spec.Procedure,
				authorizer:   a,
			})
		}
	}, nil
}

// authorizingStream authorizes the call once the request message has been
// received, because the resource scope is derived from the message content.
type authorizingStream struct {
	connect.ServerStream
	ctx        context.Context
	procedure  string
	authorizer *authorizer
}

func (s *authorizingStream) Receive(msg any) error {
	if err := s.ServerStream.Receive(msg); err != nil {
		return err
	}
	if err := s.authorizer.authorize(s.ctx, s.procedure, msg); err != nil {
		code := connect.CodePermissionDenied
		if errors.Is(err, errMissingPrincipal) {
			code = connect.CodeUnauthenticated
		}
		return connect.NewError(code, err.Error()).WithCause(err)
	}
	return nil
}

var errMissingPrincipal = fmt.Errorf("missing authenticated principal")

func (a *authorizer) authorize(ctx context.Context, procedure string, msg any) error {
	policy, ok := a.policies[procedure]
	if !ok {
		return fmt.Errorf("no authorization policy for procedure %s", procedure)
	}

	principal, principalFound := authn.PrincipalFromContext(ctx)
	if !principalFound {
		if policy.AllowAnonymous || authn.IsAnonymousProcedure(procedure) {
			return nil
		}
		return errMissingPrincipal
	}

	resourceScope, err := a.resolver.Resolve(ctx, requestHeader(ctx), msg, policy)
	if err != nil {
		return fmt.Errorf("resolve authorization scope: %w", err)
	}

	input := policyInput{
		Procedure:      procedure,
		Capability:     policy.Capability,
		AllowAnonymous: policy.AllowAnonymous,
		Principal:      &principal,
		Resource:       resourceScope,
	}

	allowed, err := a.eval.Evaluate(ctx, input)
	if err != nil {
		return err
	}
	if !allowed {
		a.logger.Warn("authorization denied",
			log.String("procedure", procedure),
			log.String("capability", policy.Capability),
			log.String("principal_id", principal.ID),
		)
		return fmt.Errorf("access denied")
	}

	return nil
}

func requestHeader(ctx context.Context) *connect.Header {
	if callInfo, ok := connect.CallInfoForServerContext(ctx); ok {
		return callInfo.RequestHeader()
	}
	return &connect.Header{}
}
