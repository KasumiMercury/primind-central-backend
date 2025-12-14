package device

import "fmt"

type Platform string

const (
	PlatformWeb     Platform = "web"
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
)

func NewPlatform(p string) (Platform, error) {
	switch p {
	case string(PlatformWeb), string(PlatformAndroid), string(PlatformIOS):
		return Platform(p), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidPlatform, p)
	}
}

func (p Platform) String() string {
	return string(p)
}
