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
			name:     "TypeShort has active period",
			taskType: TypeShort,
			expected: ActivePeriodShort,
			exists:   true,
		},
		{
			name:     "TypeNear has active period",
			taskType: TypeNear,
			expected: ActivePeriodNear,
			exists:   true,
		},
		{
			name:     "TypeRelaxed has active period",
			taskType: TypeRelaxed,
			expected: ActivePeriodRelaxed,
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
			name:     "TypeShort returns 30 minutes",
			taskType: TypeShort,
			expected: ActivePeriodShort,
		},
		{
			name:     "TypeNear returns 3 hours",
			taskType: TypeNear,
			expected: ActivePeriodNear,
		},
		{
			name:     "TypeRelaxed returns 24 hours",
			taskType: TypeRelaxed,
			expected: ActivePeriodRelaxed,
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
		period := ActivePeriodShort
		duration := time.Duration(period)

		if duration != 30*time.Minute {
			t.Errorf("time.Duration(ActivePeriodShort) = %v, want %v", duration, 30*time.Minute)
		}
	})

	t.Run("ActivePeriod can be used with time.Add", func(t *testing.T) {
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		period := ActivePeriodNear

		result := baseTime.Add(time.Duration(period))
		expected := baseTime.Add(3 * time.Hour)

		if !result.Equal(expected) {
			t.Errorf("baseTime.Add(ActivePeriodNear) = %v, want %v", result, expected)
		}
	})
}
