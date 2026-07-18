package config

import "time"

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
)
