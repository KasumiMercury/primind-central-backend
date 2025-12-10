package task

import (
	"testing"
	"time"
)

func TestDefaultActivePeriods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		taskType Type
		expected ActivePeriod
		exists   bool
	}{
		{
			name:     "TypeUrgent has active period",
			taskType: TypeUrgent,
			expected: ActivePeriodUrgent,
			exists:   true,
		},
		{
			name:     "TypeNormal has active period",
			taskType: TypeNormal,
			expected: ActivePeriodNormal,
			exists:   true,
		},
		{
			name:     "TypeLow has active period",
			taskType: TypeLow,
			expected: ActivePeriodLow,
			exists:   true,
		},
		{
			name:     "TypeScheduled has no active period in map",
			taskType: TypeScheduled,
			expected: 0,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period, ok := DefaultActivePeriods[tt.taskType]
			if ok != tt.exists {
				t.Errorf("DefaultActivePeriods[%v] exists = %v, want %v", tt.taskType, ok, tt.exists)
			}

			if ok && period != tt.expected {
				t.Errorf("DefaultActivePeriods[%v] = %v, want %v", tt.taskType, period, tt.expected)
			}
		})
	}
}

func TestGetActivePeriodForType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		taskType Type
		expected ActivePeriod
	}{
		{
			name:     "TypeUrgent returns 15 minutes",
			taskType: TypeUrgent,
			expected: ActivePeriodUrgent,
		},
		{
			name:     "TypeNormal returns 1 hour",
			taskType: TypeNormal,
			expected: ActivePeriodNormal,
		},
		{
			name:     "TypeLow returns 6 hours",
			taskType: TypeLow,
			expected: ActivePeriodLow,
		},
		{
			name:     "TypeScheduled returns 0",
			taskType: TypeScheduled,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetActivePeriodForType(tt.taskType)
			if result != tt.expected {
				t.Errorf("GetActivePeriodForType(%v) = %v, want %v", tt.taskType, result, tt.expected)
			}
		})
	}
}

func TestActivePeriodTypeConversion(t *testing.T) {
	t.Parallel()

	t.Run("ActivePeriod can be converted to time.Duration", func(t *testing.T) {
		period := ActivePeriodUrgent
		duration := time.Duration(period)

		if duration != 15*time.Minute {
			t.Errorf("time.Duration(ActivePeriodUrgent) = %v, want %v", duration, 15*time.Minute)
		}
	})

	t.Run("ActivePeriod can be used with time.Add", func(t *testing.T) {
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		period := ActivePeriodNormal

		result := baseTime.Add(time.Duration(period))
		expected := baseTime.Add(1 * time.Hour)

		if !result.Equal(expected) {
			t.Errorf("baseTime.Add(ActivePeriodNormal) = %v, want %v", result, expected)
		}
	})
}
