package repository

import "errors"

var (
	ErrTaskRequired          = errors.New("task is required")
	ErrPeriodSettingRequired = errors.New("period setting is required")
)
