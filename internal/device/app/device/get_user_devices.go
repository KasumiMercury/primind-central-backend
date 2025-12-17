package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
)

type GetUserDevicesRequest struct {
	SessionToken string
}

type DeviceInfo struct {
	DeviceID string
	FCMToken *string
}

type GetUserDevicesResult struct {
	Devices []DeviceInfo
}

type GetUserDevicesUseCase interface {
	GetUserDevices(ctx context.Context, req *GetUserDevicesRequest) (*GetUserDevicesResult, error)
}

type getUserDevicesHandler struct {
	authClient authclient.AuthClient
	deviceRepo domaindevice.DeviceRepository
	logger     *slog.Logger
}

func NewGetUserDevicesHandler(
	authClient authclient.AuthClient,
	deviceRepo domaindevice.DeviceRepository,
) GetUserDevicesUseCase {
	return &getUserDevicesHandler{
		authClient: authClient,
		deviceRepo: deviceRepo,
		logger:     slog.Default().WithGroup("device").WithGroup("getuserdevices"),
	}
}

func (h *getUserDevicesHandler) GetUserDevices(ctx context.Context, req *GetUserDevicesRequest) (*GetUserDevicesResult, error) {
	if req == nil {
		return nil, ErrGetUserDevicesRequestRequired
	}

	userIDStr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		if errors.Is(err, authclient.ErrUnauthorized) {
			h.logger.Info("session validation failed", slog.String("error", err.Error()))

			return nil, ErrUnauthorized
		}

		h.logger.Error("session validation failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	userID, err := domainuser.NewIDFromString(userIDStr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return nil, fmt.Errorf("invalid user id from auth service: %w", err)
	}

	devices, err := h.deviceRepo.ListDevicesByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to list devices", slog.String("error", err.Error()))

		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	result := &GetUserDevicesResult{
		Devices: make([]DeviceInfo, 0, len(devices)),
	}

	for _, device := range devices {
		result.Devices = append(result.Devices, DeviceInfo{
			DeviceID: device.ID().String(),
			FCMToken: device.FCMToken(),
		})
	}

	h.logger.Info("user devices retrieved",
		slog.String("user_id", userID.String()),
		slog.Int("device_count", len(result.Devices)),
	)

	return result, nil
}
