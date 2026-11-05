package server

import (
	"context"
	"fmt"
	"time"

	connect "connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/otelconnect"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/authz"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // multiline signature for line-length compliance
func newConnectInterceptors(
	ctx context.Context,
	cfg *Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
	authnInterceptor connect.ServerInterceptor,
) ([]connect.ServerInterceptor, error) {
	authzInterceptor, err := authz.NewInterceptor(ctx, cfg.Authz, pool, logger)
	if err != nil {
		return nil, fmt.Errorf("create authorization interceptor: %w", err)
	}

	telemetryInterceptor, err := otelconnect.NewServerInterceptor()
	if err != nil {
		return nil, fmt.Errorf("create connect telemetry interceptor: %w", err)
	}

	return []connect.ServerInterceptor{
		authnInterceptor,
		authzInterceptor,
		newConnectLoggingInterceptor(logger.Named("rpc")),
		telemetryInterceptor,
		newConnectTraceIDInterceptor(),
	}, nil
}

const traceIDHeader = "x-trace-id"

// newConnectTraceIDInterceptor adds the trace id as response header.
// The header is set before the handler runs so it is sent with both
// successful and error responses.
func newConnectTraceIDInterceptor() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		//nolint:whitespace // multiline callback signature for line-length compliance
		return func(
			ctx context.Context,
			spec connect.Spec,
			stream connect.ServerStream,
		) error {
			if traceID := traceIDFromContext(ctx); traceID != "" {
				if callInfo, ok := connect.CallInfoForServerContext(ctx); ok {
					callInfo.ResponseHeader().Set(traceIDHeader, traceID)
				}
			}
			return next(ctx, spec, stream)
		}
	}
}

func traceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}

	return spanContext.TraceID().String()
}

func newConnectLoggingInterceptor(logger *log.Logger) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		//nolint:whitespace // multiline callback signature for line-length compliance
		return func(
			ctx context.Context,
			spec connect.Spec,
			stream connect.ServerStream,
		) error {
			if spec.StreamType != connect.StreamTypeUnary {
				return next(ctx, spec, stream)
			}

			start := time.Now()
			err := next(ctx, spec, stream)
			l := logger.WithCtx(ctx)

			fields := []log.Field{
				log.String("procedure", spec.Procedure),
			}
			if callInfo, ok := connect.CallInfoForServerContext(ctx); ok {
				fields = append(fields,
					log.String("protocol", callInfo.Protocol),
					log.String("peer_address", callInfo.PeerAddr),
				)
			}
			if httpInfo, ok := connecthttp.ServerInfoForContext(ctx); ok {
				fields = append(fields, log.String("http_method", httpInfo.HTTPMethod()))
			}
			fields = append(fields, log.Duration("duration", time.Since(start)))

			if err != nil {
				fields = append(fields, log.String("code", connect.CodeOf(err).String()))
				l.Error("rpc failed", append(fields, log.ErrorField(err))...)
				return err
			}

			l.Info("rpc completed", fields...)
			return nil
		}
	}
}
