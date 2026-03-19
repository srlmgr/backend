package server

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	connect "connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace //editor/linter issue
func newConnectHandlerOptions(
	logger *log.Logger,
	authnr *authn.Authenticator,
) ([]connect.HandlerOption, error) {
	telemetryInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, fmt.Errorf("create connect telemetry interceptor: %w", err)
	}

	interceptors := []connect.Interceptor{
		newConnectLoggingInterceptor(logger.Named("rpc")),
		telemetryInterceptor,
		newConnectTraceIDInterceptor(),
	}
	if authnr != nil {
		interceptors = slices.Insert(interceptors, 0, authnr.NewInterceptor())
	}

	return []connect.HandlerOption{
		connect.WithInterceptors(interceptors...),
	}, nil
}

const traceIDHeader = "x-trace-id"

func newConnectTraceIDInterceptor() connect.Interceptor {
	return &connectTraceIDInterceptor{}
}

type connectTraceIDInterceptor struct{}

//nolint:whitespace //editor/linter issue
func (i *connectTraceIDInterceptor) WrapUnary(
	next connect.UnaryFunc,
) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)

		traceID := traceIDFromContext(ctx)
		if traceID == "" {
			return resp, err
		}
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			connectErr.Meta().Set(traceIDHeader, traceID)
			return resp, connectErr
		}

		if resp != nil && resp.Header() != nil {
			resp.Header().Set(traceIDHeader, traceID)
		}

		return resp, err
	}
}

//nolint:whitespace //editor/linter issue
func (i *connectTraceIDInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if traceID := traceIDFromContext(ctx); traceID != "" {
			if conn.ResponseHeader() != nil {
				conn.ResponseHeader().Set(traceIDHeader, traceID)
			}
		}

		return next(ctx, conn)
	}
}

//nolint:whitespace //editor/linter issue
func (i *connectTraceIDInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return next
}

func traceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}

	return spanContext.TraceID().String()
}

//nolint:whitespace //editor/linter issue
func newConnectLoggingInterceptor(logger *log.Logger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (
			connect.AnyResponse,
			error,
		) {
			start := time.Now()
			resp, err := next(ctx, req)

			fields := []log.Field{
				log.String("procedure", req.Spec().Procedure),
				log.String("protocol", req.Peer().Protocol),
				log.String("peer_address", req.Peer().Addr),
				log.String("http_method", req.HTTPMethod()),
				log.String("code", connect.CodeOf(err).String()),
				log.Duration("duration", time.Since(start)),
			}

			if err != nil {
				logger.Error("rpc failed", append(fields, log.ErrorField(err))...)
				return resp, err
			}

			logger.Info("rpc completed", fields...)
			return resp, nil
		}
	})
}
