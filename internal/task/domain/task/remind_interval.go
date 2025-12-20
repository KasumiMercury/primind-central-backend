package task

import "time"

type ReminderInterval time.Duration

var (
	DefaultReminderIntervalsShort = []ReminderInterval{
		ReminderInterval(3 * time.Minute),
		ReminderInterval(5 * time.Minute),
	}

	DefaultReminderIntervalsNear = []ReminderInterval{
		ReminderInterval(33 * time.Minute),
		ReminderInterval(53 * time.Minute),
	}

	DefaultReminderIntervalsRelaxed = []ReminderInterval{
		ReminderInterval(126 * time.Minute),
		ReminderInterval(232 * time.Minute),
		ReminderInterval(315 * time.Minute),
	}
)

var DefaultReminderIntervals = map[Type][]ReminderInterval{
	TypeShort:     DefaultReminderIntervalsShort,
	TypeNear:      DefaultReminderIntervalsNear,
	TypeRelaxed:   DefaultReminderIntervalsRelaxed,
	TypeScheduled: DefaultReminderIntervalsRelaxed, // TODO: Customize later
}

func GetReminderIntervalsForType(taskType Type) []ReminderInterval {
	if intervals, ok := DefaultReminderIntervals[taskType]; ok {
		return intervals
	}

	return nil
}
