package middleware

import (
	"context"
	"log/slog"

	"connectrpc.com/connect/v2"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func ConnectLoggingInterceptor(module logging.Module) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			info, _ := connect.CallInfoForServerContext(ctx)

			requestID := logging.ValidateAndExtractRequestID(info.RequestHeader().Get("x-request-id"))

			ctx = logging.WithRequestID(ctx, requestID)
			if module != "" {
				ctx = logging.WithModule(ctx, module)
			}

			info.RequestHeader().Set("x-request-id", requestID)

			procedure := spec.Procedure

			err := next(ctx, spec, stream)
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

			return err
		}
	}
}

func ConnectClientInterceptor() connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info, _ := connect.CallInfoForClientContext(ctx)

			requestID := logging.ValidateAndExtractRequestID(logging.RequestIDFromContext(ctx))
			ctx = logging.WithRequestID(ctx, requestID)
			info.RequestHeader().Set("x-request-id", requestID)

			carrier := propagation.MapCarrier{}
			otel.GetTextMapPropagator().Inject(ctx, carrier)

			for key, value := range carrier {
				info.RequestHeader().Set(key, value)
			}

			return next(ctx, spec)
		}
	}
}
