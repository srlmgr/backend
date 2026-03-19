package authz

import "time"

// Config configures authorization behavior.
type Config struct {
	Enabled          bool
	PolicyPath       string
	DecisionCacheTTL time.Duration
}
