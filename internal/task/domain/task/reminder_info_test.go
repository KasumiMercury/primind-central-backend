package task

import (
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

func TestCalculateReminderTimes(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	t.Run("TypeShort returns 3 reminder times (2 percentages + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 30 minutes (ActivePeriodShort)
		targetAt := baseTime.Add(30 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Short Task",
			TypeShort,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 0.70*30min=21min, 0.96*30min=28min48s, 30min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(21 * time.Minute),
			baseTime.Add(28*time.Minute + 48*time.Second),
			baseTime.Add(30 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeNear returns 3 reminder times (2 percentages + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 3 hours (ActivePeriodNear)
		targetAt := baseTime.Add(3 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Near Task",
			TypeNear,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 0.56*180min=100min48s, 0.89*180min=160min12s, 180min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(100*time.Minute + 48*time.Second),
			baseTime.Add(160*time.Minute + 12*time.Second),
			baseTime.Add(180 * time.Minute),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeRelaxed returns 4 reminder times (3 percentages + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// targetAt = createdAt + 24 hours (ActivePeriodRelaxed)
		targetAt := baseTime.Add(24 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Relaxed Task",
			TypeRelaxed,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Expected: 0.35*1440min=504min, 0.65*1440min=936min, 0.87*1440min=1252min48s, 1440min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(504 * time.Minute),
			baseTime.Add(936 * time.Minute),
			baseTime.Add(1252*time.Minute + 48*time.Second),
			baseTime.Add(24 * time.Hour),
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("TypeScheduled returns 4 reminder times (using Relaxed percentages + targetAt)", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// For scheduled tasks, targetAt = scheduledAt
		scheduledAt := baseTime.Add(24 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Scheduled Task",
			TypeScheduled,
			StatusActive,
			"",
			&scheduledAt,
			baseTime,
			scheduledAt, // targetAt equals scheduledAt for scheduled tasks
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// TypeScheduled uses Relaxed percentages (3) + targetAt = 4 reminder times
		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Expected: 0.35*24h=504min, 0.65*24h=936min, 0.87*24h=1252min48s, then scheduledAt
		expectedTimes := []time.Time{
			baseTime.Add(504 * time.Minute),
			baseTime.Add(936 * time.Minute),
			baseTime.Add(1252*time.Minute + 48*time.Second),
			scheduledAt,
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v",
					i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("nil task returns nil", func(t *testing.T) {
		info := CalculateReminderTimes(nil, "", nil)
		if info != nil {
			t.Error("expected nil for nil task")
		}
	})

	t.Run("short duration scheduled task uses short percentages", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Scheduled task with 25 minutes duration uses short percentages (0.70, 0.96)
		scheduledAt := baseTime.Add(25 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Short Scheduled Task",
			TypeScheduled,
			StatusActive,
			"",
			&scheduledAt,
			baseTime,
			scheduledAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// 2 short percentages + targetAt = 3 reminder times
		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 0.70*25min=17.5min, 0.96*25min=24min, 25min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(17*time.Minute + 30*time.Second),
			baseTime.Add(24 * time.Minute),
			scheduledAt,
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v", i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("near duration scheduled task uses near percentages", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Scheduled task with 2 hours duration uses near percentages (0.56, 0.89)
		scheduledAt := baseTime.Add(2 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Near Scheduled Task",
			TypeScheduled,
			StatusActive,
			"",
			&scheduledAt,
			baseTime,
			scheduledAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// 2 near percentages + targetAt = 3 reminder times
		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Expected: 0.56*120min=67.2min, 0.89*120min=106.8min, 120min (targetAt)
		expectedTimes := []time.Time{
			baseTime.Add(67*time.Minute + 12*time.Second),
			baseTime.Add(106*time.Minute + 48*time.Second),
			scheduledAt,
		}

		for i, expected := range expectedTimes {
			if !info.ReminderTimes[i].Equal(expected) {
				t.Errorf("ReminderTimes[%d] = %v, want %v", i, info.ReminderTimes[i], expected)
			}
		}
	})

	t.Run("last reminder time always equals targetAt", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Verify that targetAt is always the last reminder time
		targetAt := baseTime.Add(24 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test TargetAt Last",
			TypeRelaxed,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// 3 percentages + targetAt = 4 reminder times
		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Last reminder should equal targetAt
		lastReminder := info.ReminderTimes[len(info.ReminderTimes)-1]
		if !lastReminder.Equal(targetAt) {
			t.Errorf("last ReminderTime = %v, want %v (targetAt)", lastReminder, targetAt)
		}
	})

	t.Run("reminder times are in ascending order", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		targetAt := baseTime.Add(24 * time.Hour)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Ascending Order",
			TypeRelaxed,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// Verify all reminder times are in ascending order
		for i := 1; i < len(info.ReminderTimes); i++ {
			if !info.ReminderTimes[i].After(info.ReminderTimes[i-1]) {
				t.Errorf("ReminderTimes[%d] = %v is not after ReminderTimes[%d] = %v",
					i, info.ReminderTimes[i], i-1, info.ReminderTimes[i-1])
			}
		}

		// All reminder times should be between createdAt and targetAt (inclusive)
		for i, rt := range info.ReminderTimes {
			if rt.Before(baseTime) {
				t.Errorf("ReminderTimes[%d] = %v is before createdAt %v", i, rt, baseTime)
			}

			if rt.After(targetAt) {
				t.Errorf("ReminderTimes[%d] = %v is after targetAt %v", i, rt, targetAt)
			}
		}
	})

	t.Run("ReminderInfo contains correct task ID and type", func(t *testing.T) {
		taskID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		targetAt := baseTime.Add(30 * time.Minute)

		task, err := NewTask(
			taskID,
			validUserID,
			"Test Task",
			TypeShort,
			StatusActive,
			"",
			nil,
			baseTime,
			targetAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("NewTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info.TaskID != task.ID() {
			t.Errorf("TaskID = %v, want %v", info.TaskID, task.ID())
		}

		if info.TaskType != task.TaskType() {
			t.Errorf("TaskType = %v, want %v", info.TaskType, task.TaskType())
		}
	})
}

func TestCalculateReminderTimesWithCreateTask(t *testing.T) {
	t.Parallel()

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validColor := MustColor("#FF6B6B")

	t.Run("works correctly with CreateTask for urgent type", func(t *testing.T) {
		task, err := CreateTask(
			nil,
			validUserID,
			"Urgent Task",
			TypeShort,
			"",
			nil,
			validColor,
		)
		if err != nil {
			t.Fatalf("CreateTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// urgent: 2 intervals + targetAt = 3 reminder times
		if len(info.ReminderTimes) != 3 {
			t.Errorf("got %d reminder times, want 3", len(info.ReminderTimes))
		}

		// Verify the last reminder time equals targetAt
		lastReminderTime := info.ReminderTimes[len(info.ReminderTimes)-1]
		if !lastReminderTime.Equal(task.TargetAt()) {
			t.Errorf("last reminder time = %v, want %v (targetAt)", lastReminderTime, task.TargetAt())
		}

		// Verify all reminder times are before or equal to targetAt
		for i, reminderTime := range info.ReminderTimes {
			if reminderTime.After(task.TargetAt()) {
				t.Errorf("ReminderTimes[%d] = %v is after targetAt %v", i, reminderTime, task.TargetAt())
			}
		}

		// Verify reminder times are in ascending order
		for i := 1; i < len(info.ReminderTimes); i++ {
			if !info.ReminderTimes[i].After(info.ReminderTimes[i-1]) {
				t.Errorf("ReminderTimes[%d] = %v is not after ReminderTimes[%d] = %v",
					i, info.ReminderTimes[i], i-1, info.ReminderTimes[i-1])
			}
		}
	})

	t.Run("works correctly with CreateTask for scheduled type", func(t *testing.T) {
		scheduledAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

		task, err := CreateTask(
			nil,
			validUserID,
			"Scheduled Task",
			TypeScheduled,
			"",
			&scheduledAt,
			validColor,
		)
		if err != nil {
			t.Fatalf("CreateTask() unexpected error: %v", err)
		}

		info := CalculateReminderTimes(task, "test-user-id", nil)

		if info == nil {
			t.Fatal("CalculateReminderTimes returned nil")
		}

		// scheduled uses relaxed intervals (3) + targetAt = 4 reminder times
		if len(info.ReminderTimes) != 4 {
			t.Errorf("got %d reminder times, want 4", len(info.ReminderTimes))
		}

		// Verify the last reminder time equals targetAt (which equals scheduledAt)
		lastReminderTime := info.ReminderTimes[len(info.ReminderTimes)-1]
		if !lastReminderTime.Equal(task.TargetAt()) {
			t.Errorf("last reminder time = %v, want %v (targetAt)", lastReminderTime, task.TargetAt())
		}
	})
}
