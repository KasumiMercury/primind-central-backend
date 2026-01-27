package period

import (
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

const (
	MinPeriodMinutes = 1
	MaxPeriodMinutes = 10080 // 7 days
)

// UserPeriodSettings holds custom period settings for a user
type UserPeriodSettings struct {
	userID  user.ID
	periods map[task.Type]int // task.Type -> minutes
}

// NewUserPeriodSettings creates a new UserPeriodSettings instance
func NewUserPeriodSettings(userID user.ID, periods map[task.Type]int) (*UserPeriodSettings, error) {
	if periods == nil {
		periods = make(map[task.Type]int)
	}

	// Validate periods
	for taskType, minutes := range periods {
		if taskType == task.TypeScheduled {
			return nil, ErrScheduledTypeNotAllowed
		}

		if !isValidTaskType(taskType) {
			return nil, ErrInvalidTaskType
		}

		if minutes < MinPeriodMinutes || minutes > MaxPeriodMinutes {
			return nil, ErrInvalidPeriodMinutes
		}
	}

	return &UserPeriodSettings{
		userID:  userID,
		periods: periods,
	}, nil
}

// UserID returns the user ID
func (s *UserPeriodSettings) UserID() user.ID {
	return s.userID
}

// Periods returns a copy of the periods map
func (s *UserPeriodSettings) Periods() map[task.Type]int {
	result := make(map[task.Type]int, len(s.periods))
	for k, v := range s.periods {
		result[k] = v
	}

	return result
}

// GetPeriod returns the custom period for a task type if set
func (s *UserPeriodSettings) GetPeriod(taskType task.Type) (time.Duration, bool) {
	if minutes, ok := s.periods[taskType]; ok {
		return time.Duration(minutes) * time.Minute, true
	}

	return 0, false
}

// GetPeriodOrDefault returns the custom period if set, otherwise returns the default
func (s *UserPeriodSettings) GetPeriodOrDefault(taskType task.Type) time.Duration {
	if period, ok := s.GetPeriod(taskType); ok {
		return period
	}

	return time.Duration(task.GetActivePeriodForType(taskType))
}

// HasCustomPeriod checks if the user has a custom period for the given task type
func (s *UserPeriodSettings) HasCustomPeriod(taskType task.Type) bool {
	_, ok := s.periods[taskType]
	return ok
}

// IsEmpty returns true if no custom periods are set
func (s *UserPeriodSettings) IsEmpty() bool {
	return len(s.periods) == 0
}

// DefaultPeriodSettings returns the default period settings
func DefaultPeriodSettings() map[task.Type]int {
	return map[task.Type]int{
		task.TypeShort:   int(task.ActivePeriodShort / task.ActivePeriod(time.Minute)),
		task.TypeNear:    int(task.ActivePeriodNear / task.ActivePeriod(time.Minute)),
		task.TypeRelaxed: int(task.ActivePeriodRelaxed / task.ActivePeriod(time.Minute)),
	}
}

func isValidTaskType(t task.Type) bool {
	switch t {
	case task.TypeShort, task.TypeNear, task.TypeRelaxed:
		return true
	default:
		return false
	}
}
