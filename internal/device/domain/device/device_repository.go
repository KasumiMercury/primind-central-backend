package device

import (
	"context"

	"github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
)

type DeviceRepository interface {
	SaveDevice(ctx context.Context, device *Device) error
	GetDeviceByID(ctx context.Context, id ID) (*Device, error)
	GetDeviceByIDAndUserID(ctx context.Context, id ID, userID user.ID) (*Device, error)
	UpdateDevice(ctx context.Context, device *Device) error
	ExistsDeviceByID(ctx context.Context, id ID) (bool, error)
	ListDevicesByUserID(ctx context.Context, userID user.ID) ([]*Device, error)
	GetDeviceBySessionToken(ctx context.Context, sessionToken string) (*Device, error)
	DeleteDevicesByUserID(ctx context.Context, userID user.ID) error
}
