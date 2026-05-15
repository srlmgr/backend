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
