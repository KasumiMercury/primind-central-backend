package clock

import "time"

// Clock provides the current time.
// This interface allows for deterministic testing of time-dependent logic.
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the system time.
type RealClock struct{}

// Now returns the current system time in UTC.
func (c *RealClock) Now() time.Time {
	return time.Now().UTC()
}

// FixedClock implements Clock returning a predetermined time.
// Used for testing time-dependent behavior.
type FixedClock struct {
	fixedTime time.Time
}

// NewFixedClock creates a FixedClock that always returns the given time.
func NewFixedClock(t time.Time) *FixedClock {
	return &FixedClock{fixedTime: t}
}

// Now returns the fixed time.
func (c *FixedClock) Now() time.Time {
	return c.fixedTime
}
