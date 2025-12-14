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

type RegisterDeviceRequest struct {
	SessionToken   string
	DeviceID       *string
	Timezone       string
	Locale         string
	Platform       domaindevice.Platform
	FCMToken       *string
	UserAgent      string
	AcceptLanguage string
}

type RegisterDeviceResult struct {
	DeviceID string
	IsNew    bool
}

type RegisterDeviceUseCase interface {
	RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*RegisterDeviceResult, error)
}

type registerDeviceHandler struct {
	authClient authclient.AuthClient
	deviceRepo domaindevice.DeviceRepository
	logger     *slog.Logger
}

func NewRegisterDeviceHandler(
	authClient authclient.AuthClient,
	deviceRepo domaindevice.DeviceRepository,
) RegisterDeviceUseCase {
	return &registerDeviceHandler{
		authClient: authClient,
		deviceRepo: deviceRepo,
		logger:     slog.Default().WithGroup("device").WithGroup("registerdevice"),
	}
}

func (h *registerDeviceHandler) RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*RegisterDeviceResult, error) {
	if req == nil {
		return nil, ErrRegisterDeviceRequestRequired
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

	if req.DeviceID != nil && *req.DeviceID != "" {
		return h.handleExistingDeviceID(ctx, req, userID)
	}

	return h.createNewDevice(ctx, req, userID)
}

func (h *registerDeviceHandler) handleExistingDeviceID(
	ctx context.Context,
	req *RegisterDeviceRequest,
	userID domainuser.ID,
) (*RegisterDeviceResult, error) {
	deviceID, err := domaindevice.NewIDFromString(*req.DeviceID)
	if err != nil {
		h.logger.Warn("invalid device ID format", slog.String("error", err.Error()))

		return nil, err
	}

	existingDevice, err := h.deviceRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, domaindevice.ErrDeviceNotFound) {
			return h.createDeviceWithID(ctx, req, userID, &deviceID)
		}

		h.logger.Error("failed to get device", slog.String("error", err.Error()))

		return nil, err
	}

	// Device exists - check ownership
	if existingDevice.UserID().String() != userID.String() {
		h.logger.Warn("device already owned by another user",
			slog.String("device_id", deviceID.String()),
		)

		return nil, ErrDeviceAlreadyOwned
	}

	// Same user - update device info
	return h.updateExistingDevice(ctx, req, existingDevice)
}

func (h *registerDeviceHandler) createNewDevice(
	ctx context.Context,
	req *RegisterDeviceRequest,
	userID domainuser.ID,
) (*RegisterDeviceResult, error) {
	return h.createDeviceWithID(ctx, req, userID, nil)
}

func (h *registerDeviceHandler) createDeviceWithID(
	ctx context.Context,
	req *RegisterDeviceRequest,
	userID domainuser.ID,
	deviceID *domaindevice.ID,
) (*RegisterDeviceResult, error) {
	// Copy session token to heap to avoid storing pointer to request-scoped memory
	sessionToken := req.SessionToken

	device, err := domaindevice.CreateDevice(
		deviceID,
		userID,
		&sessionToken,
		req.Timezone,
		req.Locale,
		req.Platform,
		req.FCMToken,
		req.UserAgent,
		req.AcceptLanguage,
	)
	if err != nil {
		h.logger.Warn("failed to create device entity", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.deviceRepo.SaveDevice(ctx, device); err != nil {
		h.logger.Error("failed to save device", slog.String("error", err.Error()))

		return nil, err
	}

	h.logger.Info("device created", slog.String("device_id", device.ID().String()))

	return &RegisterDeviceResult{
		DeviceID: device.ID().String(),
		IsNew:    true,
	}, nil
}

func (h *registerDeviceHandler) updateExistingDevice(
	ctx context.Context,
	req *RegisterDeviceRequest,
	existingDevice *domaindevice.Device,
) (*RegisterDeviceResult, error) {
	// Copy session token to heap to avoid storing pointer to request-scoped memory
	sessionToken := req.SessionToken

	updatedDevice, err := existingDevice.UpdateInfo(
		&sessionToken,
		req.Timezone,
		req.Locale,
		req.Platform,
		req.FCMToken,
		req.UserAgent,
		req.AcceptLanguage,
	)
	if err != nil {
		h.logger.Warn("failed to update device entity", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.deviceRepo.UpdateDevice(ctx, updatedDevice); err != nil {
		h.logger.Error("failed to update device", slog.String("error", err.Error()))

		return nil, err
	}

	h.logger.Info("device updated", slog.String("device_id", updatedDevice.ID().String()))

	return &RegisterDeviceResult{
		DeviceID: updatedDevice.ID().String(),
		IsNew:    false,
	}, nil
}
