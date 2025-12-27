package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

func PanicRecoveryHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func(ctx context.Context) {
			if rec := recover(); rec != nil {
				slog.ErrorContext(ctx, "panic recovered",
					slog.String("event", "app.panic"),
					slog.Any("error", rec),
				)

				w.WriteHeader(http.StatusInternalServerError)

				// Re-panic
				panic(rec)
			}
		}(ctx)

		next.ServeHTTP(w, r)
	})
}
