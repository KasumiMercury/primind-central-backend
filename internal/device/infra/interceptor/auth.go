package interceptor

import (
	"context"
	"strings"

	"connectrpc.com/connect/v2"
)

type contextKey string

const (
	tokenHeader  = "Authorization"
	bearerPrefix = "Bearer "
)

const sessionTokenKey contextKey = "session_token"

func AuthInterceptor() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			info, _ := connect.CallInfoForServerContext(ctx)

			rawToken := strings.TrimSpace(info.RequestHeader().Get(tokenHeader))
			if rawToken == "" {
				return connect.NewError(connect.CodeUnauthenticated, ErrTokenHeaderNotFound.Error()).WithCause(ErrTokenHeaderNotFound)
			}

			token := rawToken
			if len(rawToken) >= len(bearerPrefix) && strings.EqualFold(rawToken[:len(bearerPrefix)], bearerPrefix) {
				token = strings.TrimSpace(rawToken[len(bearerPrefix):])
			}

			ctx = context.WithValue(ctx, sessionTokenKey, token)

			return next(ctx, spec, stream)
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
