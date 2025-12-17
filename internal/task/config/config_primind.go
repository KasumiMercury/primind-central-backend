//go:build !gcloud

package config

import "fmt"

func (c *TaskQueueConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: task queue config is nil", ErrPrimindTasksURLInvalid)
	}

	if c.PrimindTasksURL == "" {
		return nil
	}

	if err := validateServiceURL(c.PrimindTasksURL, ErrPrimindTasksURLInvalid); err != nil {
		return err
	}

	return nil
}
