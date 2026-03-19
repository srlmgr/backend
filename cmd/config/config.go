package config

import "time"

var (
	EnableTelemetry bool
	DBURI           string
	LogConfig       string
	LogLevel        string
	OtelOutput      string // output for otel-logger (stdout, grpc)
	ServerAddress   string

	// JWT authentication configuration
	JWTIssuer               string
	JWTAudience             string
	JWKSUrl                 string
	JWKSCacheRefreshInterval time.Duration
	JWTClockSkew            time.Duration

	// API Token authentication configuration
	APITokenFilePath       string
	APITokenReloadInterval time.Duration
)
