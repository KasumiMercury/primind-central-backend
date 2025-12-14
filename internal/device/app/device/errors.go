package device

import (
	"errors"

	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
)

var (
	ErrUnauthorized                  = authclient.ErrUnauthorized
	ErrRegisterDeviceRequestRequired = errors.New("register device request is required")
	ErrDeviceAlreadyOwned            = domaindevice.ErrDeviceAlreadyOwned
)
