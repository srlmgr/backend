package config

import "time"

var (
	EnableTelemetry bool
	DBURI           string
	LogConfig       string
	LogLevel        string
	OtelOutput      string // output for otel-logger (stdout, grpc)
	ServerAddress   string

	AuthnEnabled               bool
	AuthnJWTEnabled            bool
	AuthnJWTIssuer             string
	AuthnJWTAudience           string
	AuthnJWTJWKSURL            string
	AuthnJWTClockSkew          time.Duration
	AuthnJWTRefreshInterval    time.Duration
	AuthnAPITokenFilePath      string
	AuthnAPITokenRefreshWindow time.Duration

	AuthzEnabled          bool
	AuthzPolicyPath       string
	AuthzDecisionCacheTTL time.Duration
)
