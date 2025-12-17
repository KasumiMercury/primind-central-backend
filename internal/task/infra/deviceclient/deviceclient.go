package deviceclient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	connect "connectrpc.com/connect"
	devicev1 "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1"
	devicev1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1/devicev1connect"
)

type DeviceInfo struct {
	DeviceID string
	FCMToken *string
}

type DeviceClient interface {
	GetUserDevices(ctx context.Context, sessionToken string) ([]DeviceInfo, error)
	GetUserDevicesWithRetry(ctx context.Context, sessionToken string, config RetryConfig) ([]DeviceInfo, error)
}

type deviceClient struct {
	client devicev1connect.DeviceServiceClient
	logger *slog.Logger
}

func NewDeviceClient(baseURL string) DeviceClient {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	return NewDeviceClientWithHTTPClient(baseURL, httpClient)
}

func NewDeviceClientWithHTTPClient(baseURL string, httpClient connect.HTTPClient) DeviceClient {
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if token := sessionTokenFromContext(ctx); token != "" {
				req.Header().Set("Authorization", "Bearer "+token)
			}

			return next(ctx, req)
		}
	})

	client := devicev1connect.NewDeviceServiceClient(
		httpClient,
		baseURL,
		connect.WithInterceptors(authInterceptor),
	)

	return &deviceClient{
		client: client,
		logger: slog.Default().WithGroup("task").WithGroup("deviceclient"),
	}
}

func (c *deviceClient) GetUserDevices(ctx context.Context, sessionToken string) ([]DeviceInfo, error) {
	if sessionToken == "" {
		c.logger.Warn("get user devices called with empty token")

		return nil, ErrUnauthorized
	}

	ctx = contextWithSessionToken(ctx, sessionToken)

	resp, err := c.client.GetUserDevices(ctx, &devicev1.GetUserDevicesRequest{})
	if err != nil {
		c.logger.Debug("get user devices failed", slog.String("error", err.Error()))

		connectErr := new(connect.Error)
		if errors.As(err, &connectErr) {
			switch connectErr.Code() {
			case connect.CodeUnauthenticated:
				return nil, ErrUnauthorized
			case connect.CodeInvalidArgument:
				return nil, ErrInvalidArgument
			case connect.CodeCanceled, connect.CodeUnknown, connect.CodeDeadlineExceeded,
				connect.CodeNotFound, connect.CodeAlreadyExists, connect.CodePermissionDenied,
				connect.CodeResourceExhausted, connect.CodeFailedPrecondition, connect.CodeAborted,
				connect.CodeOutOfRange, connect.CodeUnimplemented, connect.CodeInternal,
				connect.CodeUnavailable, connect.CodeDataLoss:
				return nil, ErrDeviceServiceUnavailable
			default:
				return nil, ErrDeviceServiceUnavailable
			}
		}

		return nil, ErrDeviceServiceUnavailable
	}

	devices := make([]DeviceInfo, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		devices = append(devices, DeviceInfo{
			DeviceID: d.DeviceId,
			FCMToken: d.FcmToken,
		})
	}

	return devices, nil
}

type sessionTokenKey struct{}

func contextWithSessionToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, sessionTokenKey{}, token)
}

func sessionTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(sessionTokenKey{}).(string); ok {
		return token
	}

	return ""
}
