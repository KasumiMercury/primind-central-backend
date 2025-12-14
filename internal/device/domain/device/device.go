package device

import (
	"fmt"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
	"github.com/google/uuid"
)

type ID uuid.UUID

func NewID() (ID, error) {
	v7, err := uuid.NewV7()
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDGeneration, err)
	}

	return ID(v7), nil
}

func NewIDFromString(idStr string) (ID, error) {
	uuidVal, err := uuid.Parse(idStr)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDInvalidFormat, err)
	}

	if uuidVal.Version() != 7 {
		return ID{}, ErrIDInvalidV7
	}

	return ID(uuidVal), nil
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type Device struct {
	id             ID
	userID         user.ID
	sessionToken   *string
	timezone       string
	locale         string
	platform       Platform
	fcmToken       *string
	userAgent      string
	acceptLanguage string
	createdAt      time.Time
	updatedAt      time.Time
}

func NewDevice(
	id ID,
	userID user.ID,
	sessionToken *string,
	timezone string,
	locale string,
	platform Platform,
	fcmToken *string,
	userAgent string,
	acceptLanguage string,
	createdAt time.Time,
	updatedAt time.Time,
) (*Device, error) {
	if timezone == "" {
		return nil, ErrTimezoneRequired
	}

	if locale == "" {
		return nil, ErrLocaleRequired
	}

	if userAgent == "" {
		return nil, ErrUserAgentRequired
	}

	switch platform {
	case PlatformWeb, PlatformAndroid, PlatformIOS:
		// ok
	default:
		return nil, ErrInvalidPlatform
	}

	return &Device{
		id:             id,
		userID:         userID,
		sessionToken:   sessionToken,
		timezone:       timezone,
		locale:         locale,
		platform:       platform,
		fcmToken:       fcmToken,
		userAgent:      userAgent,
		acceptLanguage: acceptLanguage,
		createdAt:      createdAt.UTC().Truncate(time.Microsecond),
		updatedAt:      updatedAt.UTC().Truncate(time.Microsecond),
	}, nil
}
}

func CreateDevice(
	deviceID *ID,
	userID user.ID,
	sessionToken *string,
	timezone string,
	locale string,
	platform Platform,
	fcmToken *string,
	userAgent string,
	acceptLanguage string,
) (*Device, error) {
	var id ID

	if deviceID != nil {
		id = *deviceID
	} else {
		newID, err := NewID()
		if err != nil {
			return nil, err
		}

		id = newID
	}

	now := time.Now().UTC()

	return NewDevice(id, userID, sessionToken, timezone, locale, platform, fcmToken, userAgent, acceptLanguage, now, now)
}

func (d *Device) UpdateInfo(
	sessionToken *string,
	timezone string,
	locale string,
	platform Platform,
	fcmToken *string,
	userAgent string,
	acceptLanguage string,
) (*Device, error) {
	return NewDevice(
		d.id,
		d.userID,
		sessionToken,
		timezone,
		locale,
		platform,
		fcmToken,
		userAgent,
		acceptLanguage,
		d.createdAt,
		time.Now().UTC(),
	)
}

func (d *Device) ID() ID {
	return d.id
}

func (d *Device) UserID() user.ID {
	return d.userID
}

func (d *Device) SessionToken() *string {
	return d.sessionToken
}

func (d *Device) Timezone() string {
	return d.timezone
}

func (d *Device) Locale() string {
	return d.locale
}

func (d *Device) Platform() Platform {
	return d.platform
}

func (d *Device) FCMToken() *string {
	return d.fcmToken
}

func (d *Device) UserAgent() string {
	return d.userAgent
}

func (d *Device) AcceptLanguage() string {
	return d.acceptLanguage
}

func (d *Device) CreatedAt() time.Time {
	return d.createdAt
}

func (d *Device) UpdatedAt() time.Time {
	return d.updatedAt
}
