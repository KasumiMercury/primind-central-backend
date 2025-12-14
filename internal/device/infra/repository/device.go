package repository

import (
	"context"
	"errors"
	"time"

	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
	"gorm.io/gorm"
)

type DeviceModel struct {
	ID             string    `gorm:"type:uuid;primaryKey"`
	UserID         string    `gorm:"type:uuid;not null;index:idx_devices_user_id"`
	SessionToken   *string   `gorm:"type:text;index:idx_devices_session_token"`
	Timezone       string    `gorm:"type:varchar(100);not null"`
	Locale         string    `gorm:"type:varchar(20);not null"`
	Platform       string    `gorm:"type:varchar(20);not null"`
	FCMToken       *string   `gorm:"type:text"`
	UserAgent      string    `gorm:"type:text;not null"`
	AcceptLanguage string    `gorm:"type:varchar(500)"`
	CreatedAt      time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"not null;autoUpdateTime"`
}

func (DeviceModel) TableName() string {
	return "devices"
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) domaindevice.DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) SaveDevice(ctx context.Context, device *domaindevice.Device) error {
	if device == nil {
		return ErrDeviceRequired
	}

	record := DeviceModel{
		ID:             device.ID().String(),
		UserID:         device.UserID().String(),
		SessionToken:   device.SessionToken(),
		Timezone:       device.Timezone(),
		Locale:         device.Locale(),
		Platform:       device.Platform().String(),
		FCMToken:       device.FCMToken(),
		UserAgent:      device.UserAgent(),
		AcceptLanguage: device.AcceptLanguage(),
		CreatedAt:      device.CreatedAt(),
		UpdatedAt:      device.UpdatedAt(),
	}

	return r.db.WithContext(ctx).Create(&record).Error
}

func (r *deviceRepository) GetDeviceByID(ctx context.Context, id domaindevice.ID) (*domaindevice.Device, error) {
	var record DeviceModel
	if err := r.db.WithContext(ctx).
		Where("id = ?", id.String()).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaindevice.ErrDeviceNotFound
		}

		return nil, err
	}

	return r.recordToDevice(record)
}

func (r *deviceRepository) GetDeviceByIDAndUserID(ctx context.Context, id domaindevice.ID, userID domainuser.ID) (*domaindevice.Device, error) {
	var record DeviceModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id.String(), userID.String()).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaindevice.ErrDeviceNotFound
		}

		return nil, err
	}

	return r.recordToDevice(record)
}

func (r *deviceRepository) UpdateDevice(ctx context.Context, device *domaindevice.Device) error {
	if device == nil {
		return ErrDeviceRequired
	}

	var record DeviceModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", device.ID().String(), device.UserID().String()).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domaindevice.ErrDeviceNotFound
		}

		return err
	}

	return r.db.WithContext(ctx).
		Model(&record).
		Updates(map[string]any{
			"session_token":   device.SessionToken(),
			"timezone":        device.Timezone(),
			"locale":          device.Locale(),
			"platform":        device.Platform().String(),
			"fcm_token":       device.FCMToken(),
			"user_agent":      device.UserAgent(),
			"accept_language": device.AcceptLanguage(),
			"updated_at":      device.UpdatedAt(),
		}).Error
}

func (r *deviceRepository) ExistsDeviceByID(ctx context.Context, id domaindevice.ID) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&DeviceModel{}).
		Where("id = ?", id.String()).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *deviceRepository) recordToDevice(record DeviceModel) (*domaindevice.Device, error) {
	deviceID, err := domaindevice.NewIDFromString(record.ID)
	if err != nil {
		return nil, err
	}

	userID, err := domainuser.NewIDFromString(record.UserID)
	if err != nil {
		return nil, err
	}

	platform, err := domaindevice.NewPlatform(record.Platform)
	if err != nil {
		return nil, err
	}

	return domaindevice.NewDevice(
		deviceID,
		userID,
		record.SessionToken,
		record.Timezone,
		record.Locale,
		platform,
		record.FCMToken,
		record.UserAgent,
		record.AcceptLanguage,
		record.CreatedAt,
		record.UpdatedAt,
	)
}

func (r *deviceRepository) ListDevicesByUserID(ctx context.Context, userID domainuser.ID) ([]*domaindevice.Device, error) {
	var records []DeviceModel

	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	devices := make([]*domaindevice.Device, 0, len(records))

	for _, record := range records {
		device, err := r.recordToDevice(record)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func (r *deviceRepository) GetDeviceBySessionToken(ctx context.Context, sessionToken string) (*domaindevice.Device, error) {
	if sessionToken == "" {
		return nil, domaindevice.ErrDeviceNotFound
	}

	var record DeviceModel
	if err := r.db.WithContext(ctx).
		Where("session_token = ?", sessionToken).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaindevice.ErrDeviceNotFound
		}

		return nil, err
	}

	return r.recordToDevice(record)
}

func (r *deviceRepository) DeleteDevicesByUserID(ctx context.Context, userID domainuser.ID) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		Delete(&DeviceModel{})

	return result.Error
}
