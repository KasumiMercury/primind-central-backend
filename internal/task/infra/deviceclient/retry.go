package deviceclient

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     2 * time.Second,
		Multiplier:      2.0,
	}
}

func (c *deviceClient) GetUserDevicesWithRetry(ctx context.Context, sessionToken string, config RetryConfig) ([]DeviceInfo, error) {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}

	var lastErr error

	interval := config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		devices, err := c.GetUserDevices(ctx, sessionToken)
		if err == nil {
			return devices, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			c.logger.Debug("non-retryable error encountered, stopping retry",
				slog.String("error", err.Error()),
				slog.Int("attempt", attempt))

			return nil, err
		}

		if attempt == config.MaxAttempts {
			c.logger.Warn("max retry attempts reached",
				slog.String("error", err.Error()),
				slog.Int("attempts", attempt))

			break
		}

		c.logger.Debug("retrying GetUserDevices",
			slog.Int("attempt", attempt),
			slog.Duration("next_interval", interval),
			slog.String("error", err.Error()))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		interval = time.Duration(float64(interval) * config.Multiplier)
		if interval > config.MaxInterval {
			interval = config.MaxInterval
		}
	}

	return nil, lastErr
}

func isRetryableError(err error) bool {
	if errors.Is(err, ErrUnauthorized) {
		return false
	}

	if errors.Is(err, ErrInvalidArgument) {
		return false
	}

	if errors.Is(err, ErrDeviceServiceUnavailable) {
		return true
	}

	return false
}
