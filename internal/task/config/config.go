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

	primindTasksURLEnv     = "PRIMIND_TASKS_URL"
	taskQueueNameEnv       = "TASK_QUEUE_NAME"
	taskQueueMaxRetriesEnv = "TASK_QUEUE_MAX_RETRIES"

	gcloudProjectIDEnv  = "GCLOUD_PROJECT_ID"
	gcloudLocationIDEnv = "GCLOUD_LOCATION_ID"
	gcloudQueueIDEnv    = "GCLOUD_QUEUE_ID"
	gcloudTargetURLEnv  = "GCLOUD_TARGET_URL"

	defaultQueueName  = "default"
	defaultMaxRetries = 3
)

type Config struct {
	AuthServiceURL   string
	DeviceServiceURL string
	TaskQueue        TaskQueueConfig
}

type TaskQueueConfig struct {
	PrimindTasksURL string
	QueueName       string

	GCloudProjectID  string
	GCloudLocationID string
	GCloudQueueID    string
	GCloudTargetURL  string

	MaxRetries int
}

func Load() (*Config, error) {
	authServiceURL := getEnv(authServiceURLEnv, defaultAuthServiceURL)
	deviceServiceURL := getEnv(deviceServiceURLEnv, defaultDeviceServiceURL)

	queueName := getEnv(taskQueueNameEnv, defaultQueueName)

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
			PrimindTasksURL: os.Getenv(primindTasksURLEnv),
			QueueName:       queueName,

			GCloudProjectID:  os.Getenv(gcloudProjectIDEnv),
			GCloudLocationID: os.Getenv(gcloudLocationIDEnv),
			GCloudQueueID:    os.Getenv(gcloudQueueIDEnv),
			GCloudTargetURL:  os.Getenv(gcloudTargetURLEnv),

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
