package authz

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/authn"
)

// CapabilityAuthorizer evaluates capability checks for HTTP endpoints that do
// not run through Connect interceptors.
type CapabilityAuthorizer struct {
	enabled bool
	eval    *opaEvaluator
}

// NewCapabilityAuthorizer creates a capability-based authorizer that uses the
// same OPA policy as the Connect interceptor.
func NewCapabilityAuthorizer(
	ctx context.Context,
	cfg Config,
) (*CapabilityAuthorizer, error) {
	a := &CapabilityAuthorizer{}
	if !cfg.Enabled {
		return a, nil
	}

	evaluator, err := newOPAEvaluator(ctx, cfg)
	if err != nil {
		return nil, err
	}

	a.enabled = true
	a.eval = evaluator

	return a, nil
}

// Authorize evaluates whether the given principal can use capability within
// the provided resource scope.
func (a *CapabilityAuthorizer) Authorize(
	ctx context.Context,
	principal authn.Principal,
	capability string,
	scope ResourceScope,
) error {
	if !a.enabled {
		return nil
	}

	if principal.ID == "" {
		return errMissingPrincipal
	}

	allowed, err := a.eval.Evaluate(ctx, policyInput{
		Capability: capability,
		Principal:  principal,
		Resource:   scope,
	})
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("access denied")
	}

	return nil
}
