package task

import "time"

type ReminderInterval float64

var (
	DefaultReminderIntervalsShort = []ReminderInterval{
		ReminderInterval(0.70),
		ReminderInterval(0.96),
	}

	DefaultReminderIntervalsNear = []ReminderInterval{
		ReminderInterval(0.56),
		ReminderInterval(0.89),
	}

	DefaultReminderIntervalsRelaxed = []ReminderInterval{
		ReminderInterval(0.35),
		ReminderInterval(0.65),
		ReminderInterval(0.87),
	}
)

var DefaultReminderIntervals = map[Type][]ReminderInterval{
	TypeShort:   DefaultReminderIntervalsShort,
	TypeNear:    DefaultReminderIntervalsNear,
	TypeRelaxed: DefaultReminderIntervalsRelaxed,
}

func GetReminderIntervalsForType(taskType Type) []ReminderInterval {
	if intervals, ok := DefaultReminderIntervals[taskType]; ok {
		return intervals
	}

	return nil
}

func GetReminderIntervalsForDuration(duration time.Duration) []ReminderInterval {
	switch {
	case duration <= time.Duration(ActivePeriodShort):
		return DefaultReminderIntervalsShort
	case duration <= time.Duration(ActivePeriodNear):
		return DefaultReminderIntervalsNear
	default:
		return DefaultReminderIntervalsRelaxed
	}
}
