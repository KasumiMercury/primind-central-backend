package task

import "time"

type ActivePeriod time.Duration

const (
	ActivePeriodShort   ActivePeriod = ActivePeriod(15 * time.Minute)
	ActivePeriodNear    ActivePeriod = ActivePeriod(1 * time.Hour)
	ActivePeriodRelaxed ActivePeriod = ActivePeriod(6 * time.Hour)
)

var DefaultActivePeriods = map[Type]ActivePeriod{
	TypeShort:   ActivePeriodShort,
	TypeNear:    ActivePeriodNear,
	TypeRelaxed: ActivePeriodRelaxed,
	// TypeScheduled uses scheduled_at directly, not an active period
}

func GetActivePeriodForType(taskType Type) ActivePeriod {
	if period, ok := DefaultActivePeriods[taskType]; ok {
		return period
	}

	return 0
}
