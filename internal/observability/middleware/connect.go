package middleware

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func ConnectLoggingInterceptor(module logging.Module) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				return next(ctx, req)
			}

			requestID := logging.ValidateAndExtractRequestID(req.Header().Get("x-request-id"))

			ctx = logging.WithRequestID(ctx, requestID)
			if module != "" {
				ctx = logging.WithModule(ctx, module)
			}

			req.Header().Set("x-request-id", requestID)

			procedure := req.Spec().Procedure

			resp, err := next(ctx, req)
			if err != nil {
				code := connect.CodeOf(err).String()

				slog.ErrorContext(ctx, "rpc failed",
					slog.String("event", "rpc.request.fail"),
					slog.String("procedure", procedure),
					slog.String("error", err.Error()),
					slog.String("code", code),
				)
			} else {
				slog.InfoContext(ctx, "rpc completed",
					slog.String("event", "rpc.request.finish"),
					slog.String("procedure", procedure),
				)
			}

			return resp, err
		}
	}
}

func ConnectClientInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !req.Spec().IsClient {
				return next(ctx, req)
			}

			requestID := logging.ValidateAndExtractRequestID(logging.RequestIDFromContext(ctx))
			ctx = logging.WithRequestID(ctx, requestID)
			req.Header().Set("x-request-id", requestID)
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header()))

			return next(ctx, req)
		}
	}
}
