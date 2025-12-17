package device

import (
	"errors"

	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
)

var (
	ErrUnauthorized                    = authclient.ErrUnauthorized
	ErrAuthServiceUnavailable          = authclient.ErrAuthServiceUnavailable
	ErrRegisterDeviceRequestRequired   = errors.New("register device request is required")
	ErrGetUserDevicesRequestRequired   = errors.New("get user devices request is required")
	ErrDeviceAlreadyOwned              = domaindevice.ErrDeviceAlreadyOwned
)
