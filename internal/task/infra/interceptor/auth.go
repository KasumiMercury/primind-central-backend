package interceptor

import (
	"context"
	"strings"

	connect "connectrpc.com/connect"
)

type contextKey string

const (
	tokenHeader  = "Authorization"
	bearerPrefix = "Bearer "
)
const sessionTokenKey contextKey = "session_token"

func AuthInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				return next(ctx, req)
			}

			rawToken := strings.TrimSpace(req.Header().Get(tokenHeader))
			if rawToken != "" {
				token := rawToken
				if len(rawToken) >= len(bearerPrefix) && strings.EqualFold(rawToken[:len(bearerPrefix)], bearerPrefix) {
					token = strings.TrimSpace(rawToken[len(bearerPrefix):])
				}

				ctx = context.WithValue(ctx, sessionTokenKey, token)
			}

			return next(ctx, req)
		}
	}
}

func ExtractSessionToken(ctx context.Context) string {
	token, ok := ctx.Value(sessionTokenKey).(string)
	if !ok {
		return ""
	}

	return token
}
