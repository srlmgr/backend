package config

var (
	EnableTelemetry bool
	DBURI           string
	LogConfig       string
	LogLevel        string
	OtelOutput      string // output for otel-logger (stdout, grpc)
	ServerAddress   string
)
