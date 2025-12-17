package task

import "time"

type ReminderInterval time.Duration

var (
	DefaultReminderIntervalsUrgent = []ReminderInterval{
		ReminderInterval(3 * time.Minute),
		ReminderInterval(5 * time.Minute),
	}

	DefaultReminderIntervalsNormal = []ReminderInterval{
		ReminderInterval(33 * time.Minute),
		ReminderInterval(53 * time.Minute),
	}

	DefaultReminderIntervalsLow = []ReminderInterval{
		ReminderInterval(126 * time.Minute),
		ReminderInterval(232 * time.Minute),
		ReminderInterval(315 * time.Minute),
	}
)

var DefaultReminderIntervals = map[Type][]ReminderInterval{
	TypeUrgent:    DefaultReminderIntervalsUrgent,
	TypeNormal:    DefaultReminderIntervalsNormal,
	TypeLow:       DefaultReminderIntervalsLow,
	TypeScheduled: DefaultReminderIntervalsLow, // TODO: Customize later
}

func GetReminderIntervalsForType(taskType Type) []ReminderInterval {
	if intervals, ok := DefaultReminderIntervals[taskType]; ok {
		return intervals
	}

	return nil
}
