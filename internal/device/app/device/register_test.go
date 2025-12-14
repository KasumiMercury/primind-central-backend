package device

import (
	"context"
	"errors"
	"testing"

	domaindevice "github.com/KasumiMercury/primind-central-backend/internal/device/domain/device"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/device/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient"
	"go.uber.org/mock/gomock"
)

func TestRegisterDeviceSuccess(t *testing.T) {
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user ID: %v", err)
	}

	fcmToken := "test-fcm-token"

	tests := []struct {
		name         string
		req          RegisterDeviceRequest
		setupMocks   func(ctrl *gomock.Controller, userID domainuser.ID) (*MockAuthClient, *MockDeviceRepository)
		expectedNew  bool
	}{
		{
			name: "create new device without device_id",
			req: RegisterDeviceRequest{
				SessionToken:   "valid-token",
				DeviceID:       nil,
				Timezone:       "America/New_York",
				Locale:         "en-US",
				Platform:       domaindevice.PlatformAndroid,
				FCMToken:       &fcmToken,
				UserAgent:      "Mozilla/5.0",
				AcceptLanguage: "en-US",
			},
			setupMocks: func(ctrl *gomock.Controller, userID domainuser.ID) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(userID.String(), nil)
				mockRepo.EXPECT().SaveDevice(gomock.Any(), gomock.Any()).Return(nil)

				return mockAuth, mockRepo
			},
			expectedNew: true,
		},
		{
			name: "create new device with provided device_id",
			req: func() RegisterDeviceRequest {
				deviceID, _ := domaindevice.NewID()
				deviceIDStr := deviceID.String()

				return RegisterDeviceRequest{
					SessionToken:   "valid-token",
					DeviceID:       &deviceIDStr,
					Timezone:       "Asia/Tokyo",
					Locale:         "ja-JP",
					Platform:       domaindevice.PlatformIOS,
					FCMToken:       nil,
					UserAgent:      "Mozilla/5.0",
					AcceptLanguage: "ja-JP",
				}
			}(),
			setupMocks: func(ctrl *gomock.Controller, userID domainuser.ID) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(userID.String(), nil)
				mockRepo.EXPECT().GetDeviceByID(gomock.Any(), gomock.Any()).Return(nil, domaindevice.ErrDeviceNotFound)
				mockRepo.EXPECT().SaveDevice(gomock.Any(), gomock.Any()).Return(nil)

				return mockAuth, mockRepo
			},
			expectedNew: true,
		},
		{
			name: "update existing device owned by same user",
			req: func() RegisterDeviceRequest {
				deviceID, _ := domaindevice.NewID()
				deviceIDStr := deviceID.String()

				return RegisterDeviceRequest{
					SessionToken:   "valid-token",
					DeviceID:       &deviceIDStr,
					Timezone:       "Europe/London",
					Locale:         "en-GB",
					Platform:       domaindevice.PlatformWeb,
					FCMToken:       &fcmToken,
					UserAgent:      "NewUserAgent",
					AcceptLanguage: "en-GB",
				}
			}(),
			setupMocks: func(ctrl *gomock.Controller, userID domainuser.ID) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				existingDeviceID, _ := domaindevice.NewID()
				oldSession := "old-session-token"
				existingDevice, _ := domaindevice.CreateDevice(
					&existingDeviceID,
					userID,
					&oldSession,
					"America/New_York",
					"en-US",
					domaindevice.PlatformAndroid,
					nil,
					"OldUserAgent",
					"en-US",
				)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(userID.String(), nil)
				mockRepo.EXPECT().GetDeviceByID(gomock.Any(), gomock.Any()).Return(existingDevice, nil)
				mockRepo.EXPECT().UpdateDevice(gomock.Any(), gomock.Any()).Return(nil)

				return mockAuth, mockRepo
			},
			expectedNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAuth, mockRepo := tt.setupMocks(ctrl, validUserID)
			handler := NewRegisterDeviceHandler(mockAuth, mockRepo)

			result, err := handler.RegisterDevice(ctx, &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if result.DeviceID == "" {
				t.Error("expected device ID, got empty string")
			}

			if result.IsNew != tt.expectedNew {
				t.Errorf("expected IsNew=%v, got %v", tt.expectedNew, result.IsNew)
			}
		})
	}
}

func TestRegisterDeviceErrors(t *testing.T) {
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user ID: %v", err)
	}

	otherUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate other user ID: %v", err)
	}

	tests := []struct {
		name        string
		req         *RegisterDeviceRequest
		setupMocks  func(ctrl *gomock.Controller) (*MockAuthClient, *MockDeviceRepository)
		expectedErr error
	}{
		{
			name: "nil request",
			req:  nil,
			setupMocks: func(ctrl *gomock.Controller) (*MockAuthClient, *MockDeviceRepository) {
				return NewMockAuthClient(ctrl), NewMockDeviceRepository(ctrl)
			},
			expectedErr: ErrRegisterDeviceRequestRequired,
		},
		{
			name: "unauthorized session",
			req: &RegisterDeviceRequest{
				SessionToken:   "invalid-token",
				Timezone:       "UTC",
				Locale:         "en-US",
				Platform:       domaindevice.PlatformWeb,
				UserAgent:      "Mozilla/5.0",
				AcceptLanguage: "en-US",
			},
			setupMocks: func(ctrl *gomock.Controller) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "invalid-token").Return("", authclient.ErrUnauthorized)

				return mockAuth, mockRepo
			},
			expectedErr: ErrUnauthorized,
		},
		{
			name: "device already owned by another user",
			req: func() *RegisterDeviceRequest {
				deviceID, _ := domaindevice.NewID()
				deviceIDStr := deviceID.String()

				return &RegisterDeviceRequest{
					SessionToken:   "valid-token",
					DeviceID:       &deviceIDStr,
					Timezone:       "UTC",
					Locale:         "en-US",
					Platform:       domaindevice.PlatformAndroid,
					UserAgent:      "Mozilla/5.0",
					AcceptLanguage: "en-US",
				}
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				existingDeviceID, _ := domaindevice.NewID()
				otherSession := "other-session-token"
				existingDevice, _ := domaindevice.CreateDevice(
					&existingDeviceID,
					otherUserID, // Different user owns this device
					&otherSession,
					"America/New_York",
					"en-US",
					domaindevice.PlatformAndroid,
					nil,
					"OldUserAgent",
					"en-US",
				)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(validUserID.String(), nil)
				mockRepo.EXPECT().GetDeviceByID(gomock.Any(), gomock.Any()).Return(existingDevice, nil)

				return mockAuth, mockRepo
			},
			expectedErr: ErrDeviceAlreadyOwned,
		},
		{
			name: "invalid device ID format",
			req: func() *RegisterDeviceRequest {
				invalidID := "not-a-uuid"

				return &RegisterDeviceRequest{
					SessionToken:   "valid-token",
					DeviceID:       &invalidID,
					Timezone:       "UTC",
					Locale:         "en-US",
					Platform:       domaindevice.PlatformWeb,
					UserAgent:      "Mozilla/5.0",
					AcceptLanguage: "en-US",
				}
			}(),
			setupMocks: func(ctrl *gomock.Controller) (*MockAuthClient, *MockDeviceRepository) {
				mockAuth := NewMockAuthClient(ctrl)
				mockRepo := NewMockDeviceRepository(ctrl)

				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(validUserID.String(), nil)

				return mockAuth, mockRepo
			},
			expectedErr: domaindevice.ErrIDInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAuth, mockRepo := tt.setupMocks(ctrl)
			handler := NewRegisterDeviceHandler(mockAuth, mockRepo)

			result, err := handler.RegisterDevice(ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error, got result: %+v", result)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}
