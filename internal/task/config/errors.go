package config

import "errors"

var (
	ErrAuthServiceURLInvalid   = errors.New("auth service URL is invalid")
	ErrDeviceServiceURLInvalid = errors.New("device service URL is invalid")
	ErrPrimindTasksURLInvalid  = errors.New("primind tasks URL is invalid")
)
