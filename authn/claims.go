package authn

import (
	"fmt"
	"strings"
)

func mapClaimsToPrincipal(claims map[string]any) (Principal, error) {
	id := stringFromAny(claims["id"])
	if id == "" {
		id = stringFromAny(claims["sub"])
	}
	tenant := stringFromAny(claims["tenant"])
	firstName := stringFromAny(claims["given_name"])
	lastName := stringFromAny(claims["family_name"])
	preferredUserName := stringFromAny(claims["preferred_username"])
	if id == "" {
		return Principal{}, fmt.Errorf("token missing required id/sub claim")
	}

	roles := stringSliceFromAny(claims["roles"])

	scopeSimulationIDs := stringSliceFromAny(claims["simulationIDs"])
	if len(scopeSimulationIDs) == 0 {
		scopeSimulationIDs = stringSliceFromAny(claims["simulationsIDs"])
	}
	scopeSeriesIDs := stringSliceFromAny(claims["seriesIDs"])

	if scopesRaw, ok := claims["scopes"].(map[string]any); ok {
		scopeSimulationIDs = resolveSimulationScope(scopeSimulationIDs, scopesRaw)
		scopeSeriesIDs = resolveSeriesScope(scopeSeriesIDs, scopesRaw)
	}

	return Principal{
		ID:            id,
		Tenant:        tenant,
		Roles:         roles,
		SimulationIDs: scopeSimulationIDs,
		SeriesIDs:     scopeSeriesIDs,
		FirstName:     firstName,
		LastName:      lastName,
		Name:          preferredUserName,
	}, nil
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func stringSliceFromAny(v any) []string {
	switch typed := v.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if val := stringFromAny(item); val != "" {
				out = append(out, val)
			}
		}
		return out
	default:
		return nil
	}
}

func resolveSimulationScope(current []string, scopesRaw map[string]any) []string {
	if len(current) > 0 {
		return current
	}

	simulationIDs := stringSliceFromAny(scopesRaw["simulationIDs"])
	if len(simulationIDs) > 0 {
		return simulationIDs
	}

	return stringSliceFromAny(scopesRaw["simulationsIDs"])
}

func resolveSeriesScope(current []string, scopesRaw map[string]any) []string {
	if len(current) > 0 {
		return current
	}

	return stringSliceFromAny(scopesRaw["seriesIDs"])
}
