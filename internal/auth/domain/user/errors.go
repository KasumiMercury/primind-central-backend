package user

import "errors"

var (
	ErrIDGeneration    = errors.New("failed to generate user ID")
	ErrIDInvalidFormat = errors.New("user ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("user ID must be a UUIDv7")

	ErrColorEmpty         = errors.New("color must be specified")
	ErrColorInvalidFormat = errors.New("color must be in #RRGGBB hex format")
	ErrPaletteEmpty       = errors.New("color palette is empty")
	ErrPaletteChoice      = errors.New("failed to choose color from palette")
	ErrPaletteInvalid     = errors.New("palette contains invalid color")

	ErrUserNotFound = errors.New("user not found")
)
