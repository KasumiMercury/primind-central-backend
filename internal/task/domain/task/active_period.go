package task

import "time"

type ActivePeriod time.Duration

const (
	ActivePeriodUrgent ActivePeriod = ActivePeriod(15 * time.Minute)
	ActivePeriodNormal ActivePeriod = ActivePeriod(1 * time.Hour)
	ActivePeriodLow    ActivePeriod = ActivePeriod(6 * time.Hour)
)

var DefaultActivePeriods = map[Type]ActivePeriod{
	TypeUrgent: ActivePeriodUrgent,
	TypeNormal: ActivePeriodNormal,
	TypeLow:    ActivePeriodLow,
	// TypeScheduled uses scheduled_at directly, not an active period
}

func GetActivePeriodForType(taskType Type) ActivePeriod {
	if period, ok := DefaultActivePeriods[taskType]; ok {
		return period
	}

	return 0
}
