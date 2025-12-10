package task

import (
	"fmt"
	"strings"
)

type Color struct {
	hex string
}

func NewColor(value string) (Color, error) {
	normalized, err := normalizeHex(value)
	if err != nil {
		return Color{}, err
	}

	return Color{hex: normalized}, nil
}

// MustColor is a helper for tests or pre-validated constants.
func MustColor(value string) Color {
	c, err := NewColor(value)
	if err != nil {
		panic(fmt.Sprintf("invalid color %q: %v", value, err))
	}

	return c
}

func (c Color) String() string {
	return c.hex
}

func (c Color) Validate() error {
	if c.hex == "" {
		return ErrColorEmpty
	}

	_, err := normalizeHex(c.hex)

	return err
}

func normalizeHex(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", ErrColorEmpty
	}

	v = strings.TrimPrefix(v, "#")

	if len(v) != 6 {
		return "", ErrColorInvalidFormat
	}

	for _, r := range v {
		if !isHexDigit(r) {
			return "", ErrColorInvalidFormat
		}
	}

	return "#" + strings.ToUpper(v), nil
}

func isHexDigit(r rune) bool {
	switch {
	case r >= '0' && r <= '9':
		return true
	case r >= 'a' && r <= 'f':
		return true
	case r >= 'A' && r <= 'F':
		return true
	default:
		return false
	}
}
