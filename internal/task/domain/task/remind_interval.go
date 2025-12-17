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

type ReminderInfo struct {
	TaskID        ID
	TaskType      Type
	ReminderTimes []time.Time
}

func GetReminderIntervalsForType(taskType Type) []ReminderInterval {
	if intervals, ok := DefaultReminderIntervals[taskType]; ok {
		return intervals
	}

	return nil
}

func CalculateReminderTimes(task *Task) *ReminderInfo {
	if task == nil {
		return nil
	}

	intervals := GetReminderIntervalsForType(task.TaskType())
	createdAt := task.CreatedAt()
	targetAt := task.TargetAt()

	reminderTimes := make([]time.Time, 0, len(intervals)+1)

	for _, interval := range intervals {
		reminderTime := createdAt.Add(time.Duration(interval))
		reminderTimes = append(reminderTimes, reminderTime)
	}

	reminderTimes = append(reminderTimes, targetAt)

	return &ReminderInfo{
		TaskID:        task.ID(),
		TaskType:      task.TaskType(),
		ReminderTimes: reminderTimes,
	}
}
