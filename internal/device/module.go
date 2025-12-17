package device

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	connect "connectrpc.com/connect"
	appdevice "github.com/KasumiMercury/primind-central-backend/internal/device/app/device"
	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/interceptor"
	devicesvc "github.com/KasumiMercury/primind-central-backend/internal/device/infra/service"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1/devicev1connect"
)

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
	logger := slog.Default().WithGroup("device")

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

	devicePath, deviceHandler := devicev1connect.NewDeviceServiceHandler(
		deviceService,
		connect.WithInterceptors(interceptor.AuthInterceptor()),
	)
	logger.Info("device service handler registered", slog.String("path", devicePath))

	return devicePath, deviceHandler, nil
}
