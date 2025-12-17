package deviceclient

import "errors"

var (
	ErrUnauthorized             = errors.New("unauthorized")
	ErrDeviceServiceUnavailable = errors.New("device service unavailable")
	ErrInvalidArgument          = errors.New("invalid argument")
)
