package authclient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	connect "connectrpc.com/connect"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type AuthClient interface {
	ValidateSession(ctx context.Context, sessionToken string) (userID string, err error)
}

type authClient struct {
	client authv1connect.AuthServiceClient
	logger *slog.Logger
}

func NewAuthClient(baseURL string) AuthClient {
	client := authv1connect.NewAuthServiceClient(
		http.DefaultClient,
		baseURL,
	)

	return &authClient{
		client: client,
		logger: slog.Default().WithGroup("device").WithGroup("authclient"),
	}
}

func (c *authClient) ValidateSession(ctx context.Context, sessionToken string) (string, error) {
	if sessionToken == "" {
		c.logger.Warn("validate session called with empty token")

		return "", ErrUnauthorized
	}

	req := &authv1.ValidateSessionRequest{
		SessionToken: sessionToken,
	}

	resp, err := c.client.ValidateSession(ctx, req)
	if err != nil {
		c.logger.Info("session validation failed", slog.String("error", err.Error()))

		connectErr := new(connect.Error)
		if errors.As(err, &connectErr) {
			switch connectErr.Code() {
			case connect.CodeUnauthenticated, connect.CodeInvalidArgument:
				return "", ErrUnauthorized
			case connect.CodeCanceled, connect.CodeUnknown, connect.CodeDeadlineExceeded,
				connect.CodeNotFound, connect.CodeAlreadyExists, connect.CodePermissionDenied,
				connect.CodeResourceExhausted, connect.CodeFailedPrecondition, connect.CodeAborted,
				connect.CodeOutOfRange, connect.CodeUnimplemented, connect.CodeInternal,
				connect.CodeUnavailable, connect.CodeDataLoss:
				return "", ErrAuthServiceUnavailable
			default:
				return "", ErrAuthServiceUnavailable
			}
		}

		return "", ErrAuthServiceUnavailable
	}

	userID := resp.UserId
	if userID == "" {
		c.logger.Warn("auth service returned empty user ID")

		return "", ErrUnauthorized
	}

	return userID, nil
}
