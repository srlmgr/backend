package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

const shutdownTimeout = 10 * time.Second

const traceIDHeader = "x-trace-id"

// Config configures the HTTP server.
type Config struct {
	Address string
}

// Run starts the HTTP server and blocks until shutdown completes.
func Run(ctx context.Context, pool *pgxpool.Pool, cfg *Config) error {
	logger := getLogger(ctx)

	httpServer, err := newHTTPServer(ctx, cfg, pool, logger)
	if err != nil {
		return err
	}

	return serveHTTPServer(ctx, httpServer, cfg.Address, logger)
}

func getLogger(ctx context.Context) *log.Logger {
	logger := log.GetFromContext(ctx)
	if logger != nil {
		return logger
	}

	return log.Default()
}

//nolint:whitespace,unparam //editor/linter issue
func newHTTPServer(
	ctx context.Context,
	cfg *Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
) (*http.Server, error) {
	_ = pool

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := http.Handler(mux)
	handler = newTraceIDHeaderMiddleware()(handler)
	handler = newRequestDebugLoggingMiddleware(logger.Named("http"))(handler)
	handler = otelhttp.NewHandler(handler, "backend-http-server")

	return &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}, nil
}

type middleware func(http.Handler) http.Handler

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func newTraceIDHeaderMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := traceIDFromContext(r.Context())
			if traceID == "" {
				traceID = generateTraceID()
			}
			w.Header().Set(traceIDHeader, traceID)

			next.ServeHTTP(w, r)
		})
	}
}

func newRequestDebugLoggingMiddleware(logger *log.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			capturingWriter := &statusCapturingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(capturingWriter, r)

			traceID := capturingWriter.Header().Get(traceIDHeader)
			if traceID == "" {
				traceID = traceIDFromContext(r.Context())
			}

			logger.Debug("http request",
				log.String("method", r.Method),
				log.String("path", r.URL.Path),
				log.Int("status", capturingWriter.statusCode),
				log.Duration("duration", time.Since(started)),
				log.String("trace_id", traceID),
			)
		})
	}
}

func generateTraceID() string {
	var traceIDBytes [16]byte
	if _, err := rand.Read(traceIDBytes[:]); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}

	return hex.EncodeToString(traceIDBytes[:])
}

func traceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}

	return spanContext.TraceID().String()
}

//nolint:whitespace //editor/linter issue
func serveHTTPServer(
	ctx context.Context,
	httpServer *http.Server,
	address string,
	logger *log.Logger,
) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}

	logger.Info("starting http server", log.String("address", listener.Addr().String()))

	serveErrCh := make(chan error, 1)
	go func() {
		defer close(serveErrCh)
		if serveErr := httpServer.Serve(listener); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {

			serveErrCh <- serveErr
		}
	}()

	select {
	case serveErr := <-serveErrCh:
		if serveErr != nil {
			return fmt.Errorf("serve http server: %w", serveErr)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down http server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	if serveErr := <-serveErrCh; serveErr != nil {
		return fmt.Errorf("serve http server: %w", serveErr)
	}

	return nil
}
