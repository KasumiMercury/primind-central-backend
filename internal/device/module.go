package device

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	appdevice "github.com/KasumiMercury/primind-central-backend/internal/device/app/device"
	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/interceptor"
	devicesvc "github.com/KasumiMercury/primind-central-backend/internal/device/infra/service"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1/devicev1connect"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/middleware"
)

const moduleName logging.Module = "device"

type Repositories struct {
	Devices    domaindevice.DeviceRepository
	AuthClient authclient.AuthClient
}

func NewHTTPHandler(
	ctx context.Context,
	deviceRepo domaindevice.DeviceRepository,
	authServiceURL string,
) (string, http.Handler, error) {
	return NewHTTPHandlerWithRepositories(ctx, Repositories{
		Devices:    deviceRepo,
		AuthClient: authclient.NewAuthClient(authServiceURL),
	})
}

func NewHTTPHandlerWithRepositories(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().With(
		slog.String("module", string(moduleName)),
	).WithGroup("device")

	logger.Debug("initializing device module")

	if repos.Devices == nil {
		return "", nil, fmt.Errorf("device repository is not configured")
	}

	if repos.AuthClient == nil {
		return "", nil, fmt.Errorf("auth client is not configured")
	}

	registerDeviceUseCase := appdevice.NewRegisterDeviceHandler(repos.AuthClient, repos.Devices)
	getUserDevicesUseCase := appdevice.NewGetUserDevicesHandler(repos.AuthClient, repos.Devices)

	deviceService := devicesvc.NewService(registerDeviceUseCase, getUserDevicesUseCase)

	// Create OpenTelemetry interceptor for tracing
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		logger.Error("failed to create otelconnect interceptor", slog.String("error", err.Error()))

		return "", nil, fmt.Errorf("failed to create otelconnect interceptor: %w", err)
	}

	devicePath, deviceHandler := devicev1connect.NewDeviceServiceHandler(
		deviceService,
		connect.WithInterceptors(
			otelInterceptor,
			middleware.ConnectLoggingInterceptor(moduleName),
			interceptor.AuthInterceptor(),
		),
	)
	logger.Info("device service handler registered", slog.String("path", devicePath))

	return devicePath, deviceHandler, nil
}
