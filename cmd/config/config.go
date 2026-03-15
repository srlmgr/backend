package config

var (
	EnableTelemetry   bool
	TelemetryEndpoint string
	LogConfig         string
	LogLevel          string
	OtelOutput        string // output for otel-logger (stdout, grpc)
)
