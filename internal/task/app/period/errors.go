package periodsetting

import (
	"errors"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/period"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
)

var (
	ErrUnauthorized                        = authclient.ErrUnauthorized
	ErrAuthServiceUnavailable              = authclient.ErrAuthServiceUnavailable
	ErrGetPeriodSettingsRequestRequired    = errors.New("get period settings request is required")
	ErrUpdatePeriodSettingsRequestRequired = errors.New("update period settings request is required")
	ErrScheduledTypeNotAllowed             = period.ErrScheduledTypeNotAllowed
	ErrInvalidPeriodMinutes                = period.ErrInvalidPeriodMinutes
	ErrInvalidTaskType                     = period.ErrInvalidTaskType
)
