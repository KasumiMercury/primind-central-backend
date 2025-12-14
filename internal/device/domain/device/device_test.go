package device

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
	"github.com/google/uuid"
)

func TestNewIDSuccess(t *testing.T) {
	t.Parallel()

	t.Run("generates valid UUIDv7", func(t *testing.T) {
		id, err := NewID()
		if err != nil {
			t.Fatalf("NewID() unexpected error: %v", err)
		}

		if id.String() == "" {
			t.Error("NewID() returned empty ID")
		}

		parsedUUID := uuid.UUID(id)
		if parsedUUID.Version() != 7 {
			t.Errorf("NewID() returned UUIDv%d, want v7", parsedUUID.Version())
		}
	})
}

func TestNewIDFromStringSuccess(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validIDStr := validID.String()

	t.Run("valid UUIDv7 string", func(t *testing.T) {
		id, err := NewIDFromString(validIDStr)
		if err != nil {
			t.Fatalf("NewIDFromString(%q) unexpected error: %v", validIDStr, err)
		}

		if id.String() != validIDStr {
			t.Errorf("NewIDFromString(%q) = %q, want %q", validIDStr, id.String(), validIDStr)
		}

		parsedUUID := uuid.UUID(id)
		if parsedUUID.Version() != 7 {
			t.Errorf("NewIDFromString(%q) returned UUIDv%d, want v7", validIDStr, parsedUUID.Version())
		}
	})
}

func TestNewIDFromStringErrors(t *testing.T) {
	t.Parallel()

	uuidv4 := uuid.New()

	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty string",
			input:       "",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "invalid UUID format",
			input:       "not-a-uuid",
			expectedErr: ErrIDInvalidFormat,
		},
		{
			name:        "UUIDv4 instead of v7",
			input:       uuidv4.String(),
			expectedErr: ErrIDInvalidV7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIDFromString(tt.input)
			if err == nil {
				t.Fatalf("NewIDFromString(%q) expected error, got ID: %s", tt.input, id.String())
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewIDFromString(%q) error = %v, want %v", tt.input, err, tt.expectedErr)
			}
		})
	}
}

func TestNewPlatformSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Platform
	}{
		{name: "web", input: "web", expected: PlatformWeb},
		{name: "android", input: "android", expected: PlatformAndroid},
		{name: "ios", input: "ios", expected: PlatformIOS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, err := NewPlatform(tt.input)
			if err != nil {
				t.Fatalf("NewPlatform(%q) unexpected error: %v", tt.input, err)
			}

			if platform != tt.expected {
				t.Errorf("NewPlatform(%q) = %v, want %v", tt.input, platform, tt.expected)
			}
		})
	}
}

func TestNewPlatformErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "invalid platform", input: "windows"},
		{name: "uppercase", input: "WEB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlatform(tt.input)
			if err == nil {
				t.Fatalf("NewPlatform(%q) expected error", tt.input)
			}

			if !errors.Is(err, ErrInvalidPlatform) {
				t.Errorf("NewPlatform(%q) error = %v, want %v", tt.input, err, ErrInvalidPlatform)
			}
		})
	}
}

func TestNewDeviceSuccess(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	now := time.Now().UTC()
	fcmToken := "test-fcm-token"
	sessionToken := "test-session-token"

	t.Run("creates device with all fields", func(t *testing.T) {
		device, err := NewDevice(
			validID,
			validUserID,
			&sessionToken,
			"America/New_York",
			"en-US",
			PlatformAndroid,
			&fcmToken,
			"Mozilla/5.0",
			"en-US,en;q=0.9",
			now,
			now,
		)
		if err != nil {
			t.Fatalf("NewDevice() unexpected error: %v", err)
		}

		if device.ID() != validID {
			t.Errorf("Device.ID() = %v, want %v", device.ID(), validID)
		}

		if device.UserID() != validUserID {
			t.Errorf("Device.UserID() = %v, want %v", device.UserID(), validUserID)
		}

		if device.SessionToken() == nil || *device.SessionToken() != sessionToken {
			t.Errorf("Device.SessionToken() = %v, want %v", device.SessionToken(), &sessionToken)
		}

		if device.Timezone() != "America/New_York" {
			t.Errorf("Device.Timezone() = %v, want %v", device.Timezone(), "America/New_York")
		}

		if device.Locale() != "en-US" {
			t.Errorf("Device.Locale() = %v, want %v", device.Locale(), "en-US")
		}

		if device.Platform() != PlatformAndroid {
			t.Errorf("Device.Platform() = %v, want %v", device.Platform(), PlatformAndroid)
		}

		if device.FCMToken() == nil || *device.FCMToken() != fcmToken {
			t.Errorf("Device.FCMToken() = %v, want %v", device.FCMToken(), &fcmToken)
		}

		if device.UserAgent() != "Mozilla/5.0" {
			t.Errorf("Device.UserAgent() = %v, want %v", device.UserAgent(), "Mozilla/5.0")
		}

		if device.AcceptLanguage() != "en-US,en;q=0.9" {
			t.Errorf("Device.AcceptLanguage() = %v, want %v", device.AcceptLanguage(), "en-US,en;q=0.9")
		}
	})

	t.Run("creates device without optional FCM token", func(t *testing.T) {
		device, err := NewDevice(
			validID,
			validUserID,
			&sessionToken,
			"Asia/Tokyo",
			"ja-JP",
			PlatformIOS,
			nil,
			"Mozilla/5.0",
			"ja-JP,ja;q=0.9",
			now,
			now,
		)
		if err != nil {
			t.Fatalf("NewDevice() unexpected error: %v", err)
		}

		if device.FCMToken() != nil {
			t.Errorf("Device.FCMToken() = %v, want nil", device.FCMToken())
		}
	})

	t.Run("creates device with empty accept language", func(t *testing.T) {
		device, err := NewDevice(
			validID,
			validUserID,
			nil,
			"UTC",
			"en",
			PlatformWeb,
			nil,
			"Mozilla/5.0",
			"",
			now,
			now,
		)
		if err != nil {
			t.Fatalf("NewDevice() unexpected error: %v", err)
		}

		if device.AcceptLanguage() != "" {
			t.Errorf("Device.AcceptLanguage() = %v, want empty", device.AcceptLanguage())
		}

		if device.SessionToken() != nil {
			t.Errorf("Device.SessionToken() = %v, want nil", device.SessionToken())
		}
	})
}

func TestNewDeviceErrors(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	now := time.Now().UTC()

	tests := []struct {
		name        string
		timezone    string
		locale      string
		userAgent   string
		expectedErr error
	}{
		{
			name:        "empty timezone",
			timezone:    "",
			locale:      "en-US",
			userAgent:   "Mozilla/5.0",
			expectedErr: ErrTimezoneRequired,
		},
		{
			name:        "empty locale",
			timezone:    "UTC",
			locale:      "",
			userAgent:   "Mozilla/5.0",
			expectedErr: ErrLocaleRequired,
		},
		{
			name:        "empty user agent",
			timezone:    "UTC",
			locale:      "en-US",
			userAgent:   "",
			expectedErr: ErrUserAgentRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDevice(
				validID,
				validUserID,
				nil,
				tt.timezone,
				tt.locale,
				PlatformWeb,
				nil,
				tt.userAgent,
				"",
				now,
				now,
			)
			if err == nil {
				t.Fatalf("NewDevice() expected error")
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewDevice() error = %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestCreateDeviceSuccess(t *testing.T) {
	t.Parallel()

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	fcmToken := "test-token"
	sessionToken := "test-session-token"

	t.Run("creates device with auto-generated ID", func(t *testing.T) {
		device, err := CreateDevice(
			nil,
			validUserID,
			&sessionToken,
			"UTC",
			"en-US",
			PlatformWeb,
			&fcmToken,
			"Mozilla/5.0",
			"en-US",
		)
		if err != nil {
			t.Fatalf("CreateDevice() unexpected error: %v", err)
		}

		if device.ID().String() == "" {
			t.Error("CreateDevice() auto-generated ID is empty")
		}

		if device.SessionToken() == nil || *device.SessionToken() != sessionToken {
			t.Errorf("CreateDevice() SessionToken = %v, want %v", device.SessionToken(), &sessionToken)
		}

		parsedUUID := uuid.UUID(device.ID())
		if parsedUUID.Version() != 7 {
			t.Errorf("CreateDevice() auto-generated UUIDv%d, want v7", parsedUUID.Version())
		}
	})

	t.Run("creates device with provided ID", func(t *testing.T) {
		providedID, err := NewID()
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		device, err := CreateDevice(
			&providedID,
			validUserID,
			nil,
			"UTC",
			"en-US",
			PlatformAndroid,
			nil,
			"Mozilla/5.0",
			"en-US",
		)
		if err != nil {
			t.Fatalf("CreateDevice() unexpected error: %v", err)
		}

		if device.ID() != providedID {
			t.Errorf("CreateDevice() ID = %v, want %v", device.ID(), providedID)
		}

		if device.SessionToken() != nil {
			t.Errorf("CreateDevice() SessionToken = %v, want nil", device.SessionToken())
		}
	})
}

func TestDeviceUpdateInfo(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	originalTime := time.Now().UTC().Add(-1 * time.Hour)
	originalFCM := "original-token"
	originalSession := "original-session"
	newFCM := "new-token"
	newSession := "new-session"

	original, err := NewDevice(
		validID,
		validUserID,
		&originalSession,
		"America/New_York",
		"en-US",
		PlatformAndroid,
		&originalFCM,
		"OldUserAgent",
		"en-US",
		originalTime,
		originalTime,
	)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("updates device info preserving ID and userID", func(t *testing.T) {
		updated, err := original.UpdateInfo(
			&newSession,
			"Asia/Tokyo",
			"ja-JP",
			PlatformIOS,
			&newFCM,
			"NewUserAgent",
			"ja-JP",
		)
		if err != nil {
			t.Fatalf("UpdateInfo() unexpected error: %v", err)
		}

		if updated.ID() != original.ID() {
			t.Errorf("UpdateInfo() ID = %v, want %v", updated.ID(), original.ID())
		}

		if updated.UserID() != original.UserID() {
			t.Errorf("UpdateInfo() UserID = %v, want %v", updated.UserID(), original.UserID())
		}

		if !updated.CreatedAt().Equal(original.CreatedAt()) {
			t.Errorf("UpdateInfo() CreatedAt = %v, want %v", updated.CreatedAt(), original.CreatedAt())
		}

		if !updated.UpdatedAt().After(original.UpdatedAt()) {
			t.Errorf("UpdateInfo() UpdatedAt = %v, should be after %v", updated.UpdatedAt(), original.UpdatedAt())
		}

		if updated.SessionToken() == nil || *updated.SessionToken() != newSession {
			t.Errorf("UpdateInfo() SessionToken = %v, want %v", updated.SessionToken(), &newSession)
		}

		if updated.Timezone() != "Asia/Tokyo" {
			t.Errorf("UpdateInfo() Timezone = %v, want %v", updated.Timezone(), "Asia/Tokyo")
		}

		if updated.Locale() != "ja-JP" {
			t.Errorf("UpdateInfo() Locale = %v, want %v", updated.Locale(), "ja-JP")
		}

		if updated.Platform() != PlatformIOS {
			t.Errorf("UpdateInfo() Platform = %v, want %v", updated.Platform(), PlatformIOS)
		}

		if updated.FCMToken() == nil || *updated.FCMToken() != newFCM {
			t.Errorf("UpdateInfo() FCMToken = %v, want %v", updated.FCMToken(), &newFCM)
		}

		if updated.UserAgent() != "NewUserAgent" {
			t.Errorf("UpdateInfo() UserAgent = %v, want %v", updated.UserAgent(), "NewUserAgent")
		}
	})
}

func TestIDString(t *testing.T) {
	t.Parallel()

	id, err := NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	idStr := id.String()
	if idStr == "" {
		t.Error("ID.String() returned empty string")
	}

	_, err = uuid.Parse(idStr)
	if err != nil {
		t.Errorf("ID.String() = %q is not valid UUID format: %v", idStr, err)
	}

	if !strings.Contains(idStr, "-") {
		t.Errorf("ID.String() = %q does not appear to be UUID format (missing dashes)", idStr)
	}
}

func TestPlatformString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		platform Platform
		expected string
	}{
		{PlatformWeb, "web"},
		{PlatformAndroid, "android"},
		{PlatformIOS, "ios"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.platform.String() != tt.expected {
				t.Errorf("Platform.String() = %v, want %v", tt.platform.String(), tt.expected)
			}
		})
	}
}
