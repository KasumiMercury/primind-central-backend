package service

import (
	"context"
	"errors"
	"log/slog"

	connect "connectrpc.com/connect"
	appdevice "github.com/KasumiMercury/primind-central-backend/internal/device/app/device"
	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/interceptor"
	devicev1 "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1"
	devicev1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1/devicev1connect"
)

type Service struct {
	registerDevice appdevice.RegisterDeviceUseCase
	getUserDevices appdevice.GetUserDevicesUseCase
	logger         *slog.Logger
}

var _ devicev1connect.DeviceServiceHandler = (*Service)(nil)

func NewService(
	registerDeviceUseCase appdevice.RegisterDeviceUseCase,
	getUserDevicesUseCase appdevice.GetUserDevicesUseCase,
) *Service {
	return &Service{
		registerDevice: registerDeviceUseCase,
		getUserDevices: getUserDevicesUseCase,
		logger:         slog.Default().WithGroup("device").WithGroup("service"),
	}
}

func (s *Service) RegisterDevice(
	ctx context.Context,
	req *devicev1.RegisterDeviceRequest,
) (*devicev1.RegisterDeviceResponse, error) {
	if req == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}

	token := extractSessionTokenFromContext(ctx)
	if token == "" {
		s.logger.Warn("register device called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	platform, err := protoPlatformToDomain(req.GetPlatform())
	if err != nil {
		s.logger.Warn("invalid platform", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var deviceID *string
	if req.DeviceId != nil {
		deviceID = req.DeviceId
	}

	var fcmToken *string
	if req.FcmToken != nil {
		fcmToken = req.FcmToken
	}

	result, err := s.registerDevice.RegisterDevice(ctx, &appdevice.RegisterDeviceRequest{
		SessionToken:   token,
		DeviceID:       deviceID,
		Timezone:       req.GetTimezone(),
		Locale:         req.GetLocale(),
		Platform:       platform,
		FCMToken:       fcmToken,
		UserAgent:      req.GetUserAgent(),
		AcceptLanguage: req.GetAcceptLanguage(),
	})
	if err != nil {
		return s.handleRegisterDeviceError(err)
	}

	s.logger.Info("device registered",
		slog.String("device_id", result.DeviceID),
		slog.Bool("is_new", result.IsNew),
	)

	return &devicev1.RegisterDeviceResponse{
		DeviceId: result.DeviceID,
		IsNew:    result.IsNew,
	}, nil
}

func (s *Service) handleRegisterDeviceError(err error) (*devicev1.RegisterDeviceResponse, error) {
	switch {
	case errors.Is(err, appdevice.ErrUnauthorized):
		s.logger.Info("unauthorized register device attempt")

		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, appdevice.ErrAuthServiceUnavailable):
		s.logger.Error("auth service unavailable during register device", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, appdevice.ErrDeviceAlreadyOwned):
		s.logger.Warn("device already owned by another user", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, domaindevice.ErrIDInvalidFormat),
		errors.Is(err, domaindevice.ErrIDInvalidV7),
		errors.Is(err, domaindevice.ErrTimezoneRequired),
		errors.Is(err, domaindevice.ErrLocaleRequired),
		errors.Is(err, domaindevice.ErrUserAgentRequired),
		errors.Is(err, domaindevice.ErrInvalidPlatform):
		s.logger.Warn("invalid register device request", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		s.logger.Error("unexpected register device error", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInternal, errors.New("internal server error"))
	}
}

func extractSessionTokenFromContext(ctx context.Context) string {
	return interceptor.ExtractSessionToken(ctx)
}

func protoPlatformToDomain(platform devicev1.Platform) (domaindevice.Platform, error) {
	switch platform {
	case devicev1.Platform_PLATFORM_WEB:
		return domaindevice.PlatformWeb, nil
	case devicev1.Platform_PLATFORM_ANDROID:
		return domaindevice.PlatformAndroid, nil
	case devicev1.Platform_PLATFORM_IOS:
		return domaindevice.PlatformIOS, nil
	case devicev1.Platform_PLATFORM_UNSPECIFIED:
		return "", errors.New("platform is required")
	default:
		return "", errors.New("unsupported platform")
	}
}

func (s *Service) GetUserDevices(
	ctx context.Context,
	req *devicev1.GetUserDevicesRequest,
) (*devicev1.GetUserDevicesResponse, error) {
	if req == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}

	token := extractSessionTokenFromContext(ctx)
	if token == "" {
		s.logger.Warn("get user devices called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	result, err := s.getUserDevices.GetUserDevices(ctx, &appdevice.GetUserDevicesRequest{
		SessionToken: token,
	})
	if err != nil {
		return s.handleGetUserDevicesError(err)
	}

	devices := make([]*devicev1.DeviceInfo, 0, len(result.Devices))
	for _, d := range result.Devices {
		device := &devicev1.DeviceInfo{
			DeviceId: d.DeviceID,
			FcmToken: nil,
		}
		if d.FCMToken != nil {
			device.FcmToken = d.FCMToken
		}

		devices = append(devices, device)
	}

	s.logger.Info("user devices retrieved", slog.Int("device_count", len(devices)))

	return &devicev1.GetUserDevicesResponse{
		Devices: devices,
	}, nil
}

func (s *Service) handleGetUserDevicesError(err error) (*devicev1.GetUserDevicesResponse, error) {
	switch {
	case errors.Is(err, appdevice.ErrUnauthorized):
		s.logger.Info("unauthorized get user devices attempt")

		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, appdevice.ErrAuthServiceUnavailable):
		s.logger.Error("auth service unavailable during get user devices", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, appdevice.ErrGetUserDevicesRequestRequired):
		s.logger.Warn("get user devices request required", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		s.logger.Error("unexpected get user devices error", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInternal, errors.New("internal server error"))
	}
}
