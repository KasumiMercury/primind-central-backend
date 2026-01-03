package task

import "time"

type DeviceInfo struct {
	DeviceID string
	FCMToken *string
}

type ReminderInfo struct {
	TaskID        ID
	TaskType      Type
	UserID        string
	ReminderTimes []time.Time
	Devices       []DeviceInfo
}

func CalculateReminderTimes(task *Task, userID string, devices []DeviceInfo) *ReminderInfo {
	if task == nil {
		return nil
	}

	createdAt := task.CreatedAt()
	targetAt := task.TargetAt()
	totalDuration := targetAt.Sub(createdAt)

	var percentages []ReminderInterval
	if task.TaskType() == TypeScheduled {
		percentages = GetReminderIntervalsForDuration(totalDuration)
	} else {
		percentages = GetReminderIntervalsForType(task.TaskType())
	}

	reminderTimes := make([]time.Time, 0, len(percentages)+1)

	for _, percentage := range percentages {
		offset := time.Duration(float64(totalDuration) * float64(percentage))

		offset = offset.Round(time.Microsecond)
		reminderTime := createdAt.Add(offset)

		if !reminderTime.After(targetAt) {
			reminderTimes = append(reminderTimes, reminderTime)
		}
	}

	if len(reminderTimes) == 0 || !reminderTimes[len(reminderTimes)-1].Equal(targetAt) {
		reminderTimes = append(reminderTimes, targetAt)
	}

	return &ReminderInfo{
		TaskID:        task.ID(),
		TaskType:      task.TaskType(),
		UserID:        userID,
		ReminderTimes: reminderTimes,
		Devices:       devices,
	}
}
