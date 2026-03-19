package authz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
)

type decisionCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

type opaEvaluator struct {
	query    rego.PreparedEvalQuery
	cacheTTL time.Duration
	cacheMu  sync.Mutex
	cache    map[string]decisionCacheEntry
}

func newOPAEvaluator(ctx context.Context, cfg Config) (*opaEvaluator, error) {
	module := defaultPolicyModule
	if cfg.PolicyPath != "" {
		data, err := os.ReadFile(cfg.PolicyPath)
		if err != nil {
			return nil, fmt.Errorf("read authz policy module: %w", err)
		}
		module = string(data)
	}

	r := rego.New(
		rego.Query("data.backend.authz.allow"),
		rego.Module("authz.rego", module),
	)
	prepared, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("prepare authz policy query: %w", err)
	}

	return &opaEvaluator{
		query:    prepared,
		cacheTTL: cfg.DecisionCacheTTL,
		cache:    map[string]decisionCacheEntry{},
	}, nil
}

func (o *opaEvaluator) Evaluate(ctx context.Context, input any) (bool, error) {
	cacheKey := ""
	if o.cacheTTL > 0 {
		cacheKey = cacheInput(input)
		if allowed, ok := o.lookupCached(cacheKey); ok {
			return allowed, nil
		}
	}

	results, err := o.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("evaluate authz policy: %w", err)
	}

	allowed := false
	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if value, ok := results[0].Expressions[0].Value.(bool); ok {
			allowed = value
		}
	}

	if cacheKey != "" {
		o.storeCached(cacheKey, allowed)
	}
	return allowed, nil
}

func (o *opaEvaluator) lookupCached(key string) (bool, bool) {
	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()

	entry, ok := o.cache[key]
	if !ok {
		return false, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(o.cache, key)
		return false, false
	}
	return entry.allowed, true
}

func (o *opaEvaluator) storeCached(key string, allowed bool) {
	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()
	o.cache[key] = decisionCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(o.cacheTTL),
	}
}

func cacheInput(input any) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v", input)))
	return hex.EncodeToString(sum[:])
}

const defaultPolicyModule = `package backend.authz
import rego.v1

default allow := false

allow if {
	input.principal.id != ""
	is_administrator
}

allow if {
	input.capability == "query.read"
	is_anonymous_or_authenticated
}

allow if {
	has_role("season_operator")
	input.capability == "season.write"
	within_series_scope
}

allow if {
	has_role("season_operator")
	input.capability == "import.write"
	within_series_scope
}

allow if {
	has_role("season_operator")
	input.capability == "admin.write"
	within_series_scope
}

allow if {
	has_role("series_operator")
	input.capability == "series.write"
	within_simulation_scope
}

allow if {
	has_role("series_operator")
	input.capability == "simulation.write"
	within_simulation_scope
}

allow if {
	has_role("master_data_operator")
	input.capability == "master_data.write"
}

allow if {
	has_role("master_data_operator")
	input.capability == "simulation.write"
	within_simulation_scope
}

allow if {
	has_role("series_operator")
	input.capability == "season.write"
	within_series_scope
}

allow if {
	has_role("series_operator")
	input.capability == "import.write"
	within_series_scope
}

allow if {
	has_role("series_operator")
	input.capability == "admin.write"
	within_series_scope
}

has_role(role) if {
	some r in input.principal.roles
	r == role
}

is_administrator if {
	has_role("administrator")
}

is_anonymous_or_authenticated if {
	input.allowAnonymous
}

is_anonymous_or_authenticated if {
	input.principal.id != ""
}

within_series_scope if {
	sid := input.resource.seriesId
	sid != ""
	some allowed in input.principal.seriesIDs
	allowed == sid
}

within_simulation_scope if {
	simid := input.resource.simulationId
	simid != ""
	some allowed in input.principal.simulationIDs
	allowed == simid
}
`
