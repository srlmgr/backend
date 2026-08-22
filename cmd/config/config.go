package config

import "time"

//nolint:lll // readability
var (
	TelemetryEnabled  bool
	DBURI             string
	LogConfig         string
	LogLevel          string
	OtelOutput        string // output for otel-logger (stdout, grpc)
	GRPCServerAddress string // serves gRPC API
	HTTPServerAddress string // serves HTTP API for HTML rendering
	GRPCEnabled       bool
	HTMLEnabled       bool
	HTMLExternalURL   string // external URL for HTML server (used for navigation links)
	HTMLContextPart   string // context part for endpoints (used for navigation links)

	AuthnEnabled               bool
	IDPEnabled                 bool
	IDPIssuerURL               string
	IDPClientID                string
	IDPClientSecret            string
	IDPCallbackURL             string
	IDPFrontendURL             string
	IDPRefreshSkew             time.Duration
	AuthnAPITokenFilePath      string
	AuthnAPITokenRefreshWindow time.Duration

	AuthzEnabled          bool
	AuthzPolicyPath       string
	AuthzDecisionCacheTTL time.Duration

	CacheEnabled    bool   // enables in-memory read-through caching for selected repositories
	CacheConfigFile string // path to a cache config YAML file; empty means no per-cache tuning
)
