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

	intervals := GetReminderIntervalsForType(task.TaskType())
	createdAt := task.CreatedAt()
	targetAt := task.TargetAt()

	reminderTimes := make([]time.Time, 0, len(intervals)+1)

	for _, interval := range intervals {
		reminderTime := createdAt.Add(time.Duration(interval))
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
