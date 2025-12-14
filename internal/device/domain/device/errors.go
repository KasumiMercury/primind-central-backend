package device

import "errors"

var (
	ErrIDGeneration    = errors.New("failed to generate device ID")
	ErrIDInvalidFormat = errors.New("device ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("device ID must be a UUIDv7")

	ErrTimezoneRequired  = errors.New("timezone is required")
	ErrLocaleRequired    = errors.New("locale is required")
	ErrUserAgentRequired = errors.New("user agent is required")
	ErrInvalidPlatform   = errors.New("invalid platform")

	ErrDeviceNotFound     = errors.New("device not found")
	ErrDeviceAlreadyOwned = errors.New("device is already registered to another user")
)
