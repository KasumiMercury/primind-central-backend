package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	authServiceURLEnv       = "AUTH_SERVICE_URL"
	deviceServiceURLEnv     = "DEVICE_SERVICE_URL"
	defaultAuthServiceURL   = "http://localhost:8080"
	defaultDeviceServiceURL = "http://localhost:8080"

	primindTasksURLEnv            = "PRIMIND_TASKS_URL"
	remindRegisterQueueNameEnv   = "REMIND_REGISTER_QUEUE_NAME"
	remindCancelQueueNameEnv     = "REMIND_CANCEL_QUEUE_NAME"
	taskQueueMaxRetriesEnv       = "TASK_QUEUE_MAX_RETRIES"

	gcloudProjectIDEnv              = "GCLOUD_PROJECT_ID"
	gcloudLocationIDEnv             = "GCLOUD_LOCATION_ID"
	gcloudRemindRegisterQueueIDEnv  = "GCLOUD_REMIND_REGISTER_QUEUE_ID"
	gcloudRemindCancelQueueIDEnv    = "GCLOUD_REMIND_CANCEL_QUEUE_ID"
	gcloudRemindTargetURLEnv        = "GCLOUD_REMIND_TARGET_URL"

	defaultRemindRegisterQueueName = "remind-register"
	defaultRemindCancelQueueName   = "remind-cancel"
	defaultMaxRetries              = 3
)

type Config struct {
	AuthServiceURL   string
	DeviceServiceURL string
	TaskQueue        TaskQueueConfig
}

type TaskQueueConfig struct {
	PrimindTasksURL         string
	RemindRegisterQueueName string
	RemindCancelQueueName   string

	GCloudProjectID             string
	GCloudLocationID            string
	GCloudRemindRegisterQueueID string
	GCloudRemindCancelQueueID   string
	GCloudRemindTargetURL       string

	MaxRetries int
}

func Load() (*Config, error) {
	authServiceURL := getEnv(authServiceURLEnv, defaultAuthServiceURL)
	deviceServiceURL := getEnv(deviceServiceURLEnv, defaultDeviceServiceURL)

	remindRegisterQueueName := getEnv(remindRegisterQueueNameEnv, defaultRemindRegisterQueueName)
	remindCancelQueueName := getEnv(remindCancelQueueNameEnv, defaultRemindCancelQueueName)

	maxRetries := defaultMaxRetries

	if v := os.Getenv(taskQueueMaxRetriesEnv); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxRetries = parsed
		}
	}

	cfg := &Config{
		AuthServiceURL:   authServiceURL,
		DeviceServiceURL: deviceServiceURL,
		TaskQueue: TaskQueueConfig{
			PrimindTasksURL:         os.Getenv(primindTasksURLEnv),
			RemindRegisterQueueName: remindRegisterQueueName,
			RemindCancelQueueName:   remindCancelQueueName,

			GCloudProjectID:             os.Getenv(gcloudProjectIDEnv),
			GCloudLocationID:            os.Getenv(gcloudLocationIDEnv),
			GCloudRemindRegisterQueueID: os.Getenv(gcloudRemindRegisterQueueIDEnv),
			GCloudRemindCancelQueueID:   os.Getenv(gcloudRemindCancelQueueIDEnv),
			GCloudRemindTargetURL:       os.Getenv(gcloudRemindTargetURLEnv),

			MaxRetries: maxRetries,
		},
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrAuthServiceURLInvalid)
	}

	if err := c.validateAuthServiceURL(); err != nil {
		return err
	}

	if err := c.validateDeviceServiceURL(); err != nil {
		return err
	}

	if err := c.TaskQueue.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateAuthServiceURL() error {
	return validateServiceURL(c.AuthServiceURL, ErrAuthServiceURLInvalid)
}

func (c *Config) validateDeviceServiceURL() error {
	return validateServiceURL(c.DeviceServiceURL, ErrDeviceServiceURLInvalid)
}

func validateServiceURL(urlStr string, baseErr error) error {
	if urlStr == "" {
		return fmt.Errorf("%w: service URL is empty", baseErr)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%w: %v", baseErr, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https, got: %s",
			baseErr, parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%w: host is empty", baseErr)
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}

	return defaultVal
}
