package auth

import (
	"context"
	"errors"
	"testing"

	connect "connectrpc.com/connect"
	applogout "github.com/KasumiMercury/primind-central-backend/internal/auth/app/logout"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	appsession "github.com/KasumiMercury/primind-central-backend/internal/auth/app/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	"go.uber.org/mock/gomock"
)

func TestServiceOIDCParamsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerator := NewMockOIDCParamsGenerator(ctrl)
	mockGenerator.EXPECT().
		Generate(gomock.Any(), domainoidc.ProviderGoogle).
		Return(&appoidc.ParamsResult{
			AuthorizationURL: "https://example.com/auth",
			State:            "abc",
		}, nil)

	svc := NewService(mockGenerator, nil, nil, nil)

	resp, err := svc.OIDCParams(context.Background(), &authv1.OIDCParamsRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetAuthorizationUrl() == "" || resp.GetState() == "" {
		t.Fatalf("expected response fields to be populated")
	}
}

func TestServiceOIDCParamsError(t *testing.T) {
	tests := []struct {
		name         string
		service      func(ctrl *gomock.Controller) *Service
		req          *authv1.OIDCParamsRequest
		expectedCode connect.Code
	}{
		{
			name:         "generator missing",
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil) },
			req:          &authv1.OIDCParamsRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "invalid provider",
			service: func(ctrl *gomock.Controller) *Service {
				return NewService(NewMockOIDCParamsGenerator(ctrl), nil, nil, nil)
			},
			req:          &authv1.OIDCParamsRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_UNSPECIFIED},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "oidc not configured",
			service: func(ctrl *gomock.Controller) *Service {
				mockGenerator := NewMockOIDCParamsGenerator(ctrl)
				mockGenerator.EXPECT().
					Generate(gomock.Any(), domainoidc.ProviderGoogle).
					Return(nil, appoidc.ErrOIDCNotConfigured)

				return NewService(mockGenerator, nil, nil, nil)
			},
			req:          &authv1.OIDCParamsRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "unsupported provider",
			service: func(ctrl *gomock.Controller) *Service {
				mockGenerator := NewMockOIDCParamsGenerator(ctrl)
				mockGenerator.EXPECT().
					Generate(gomock.Any(), domainoidc.ProviderGoogle).
					Return(nil, appoidc.ErrProviderUnsupported)

				return NewService(mockGenerator, nil, nil, nil)
			},
			req:          &authv1.OIDCParamsRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "internal error",
			service: func(ctrl *gomock.Controller) *Service {
				mockGenerator := NewMockOIDCParamsGenerator(ctrl)
				mockGenerator.EXPECT().
					Generate(gomock.Any(), domainoidc.ProviderGoogle).
					Return(nil, errors.New("boom"))

				return NewService(mockGenerator, nil, nil, nil)
			},
			req:          &authv1.OIDCParamsRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).OIDCParams(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func TestServiceOIDCLoginSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogin := NewMockOIDCLoginUseCase(ctrl)
	mockLogin.EXPECT().
		Login(gomock.Any(), &appoidc.LoginRequest{
			Provider: domainoidc.ProviderGoogle,
			Code:     "code",
			State:    "state",
		}).
		Return(&appoidc.LoginResult{SessionToken: "token"}, nil)

	svc := NewService(nil, mockLogin, nil, nil)

	resp, err := svc.OIDCLogin(context.Background(), &authv1.OIDCLoginRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
		Code:     "code",
		State:    "state",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetSessionToken() != "token" {
		t.Fatalf("expected session token, got %s", resp.GetSessionToken())
	}
}

func TestServiceOIDCLoginError(t *testing.T) {
	tests := []struct {
		name         string
		service      func(ctrl *gomock.Controller) *Service
		req          *authv1.OIDCLoginRequest
		expectedCode connect.Code
	}{
		{
			name:         "handler missing",
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil) },
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "invalid provider",
			service: func(ctrl *gomock.Controller) *Service {
				return NewService(nil, NewMockOIDCLoginUseCase(ctrl), nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_UNSPECIFIED},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "oidc not configured",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, appoidc.ErrOIDCNotConfigured)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "unsupported provider",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, appoidc.ErrProviderUnsupported)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid code",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, appoidc.ErrInvalidCode)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid state",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, appoidc.ErrInvalidState)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "params expired",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, domainoidc.ErrParamsExpired)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "nonce mismatch",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, appoidc.ErrInvalidNonce)
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "unexpected error",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogin := NewMockOIDCLoginUseCase(ctrl)
				mockLogin.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))
				return NewService(nil, mockLogin, nil, nil)
			},
			req:          &authv1.OIDCLoginRequest{Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).OIDCLogin(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func TestServiceLogoutSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogout := NewMockLogoutUseCase(ctrl)
	mockLogout.EXPECT().
		Logout(gomock.Any(), &applogout.LogoutRequest{SessionToken: "token"}).
		Return(&applogout.LogoutResponse{Success: true}, nil)

	svc := NewService(nil, nil, nil, mockLogout)

	resp, err := svc.Logout(context.Background(), &authv1.LogoutRequest{SessionToken: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.GetSuccess() {
		t.Fatalf("expected success true")
	}
}

func TestServiceLogoutError(t *testing.T) {
	tests := []struct {
		name         string
		service      func(ctrl *gomock.Controller) *Service
		req          *authv1.LogoutRequest
		expectedCode connect.Code
	}{
		{
			name:         "handler missing",
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil) },
			req:          &authv1.LogoutRequest{SessionToken: "token"},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "token required",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogout := NewMockLogoutUseCase(ctrl)
				mockLogout.EXPECT().
					Logout(gomock.Any(), gomock.Any()).
					Return(nil, applogout.ErrTokenRequired)
				return NewService(nil, nil, nil, mockLogout)
			},
			req:          &authv1.LogoutRequest{},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid token",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogout := NewMockLogoutUseCase(ctrl)
				mockLogout.EXPECT().
					Logout(gomock.Any(), gomock.Any()).
					Return(nil, applogout.ErrInvalidToken)
				return NewService(nil, nil, nil, mockLogout)
			},
			req:          &authv1.LogoutRequest{SessionToken: "bad"},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "unexpected error",
			service: func(ctrl *gomock.Controller) *Service {
				mockLogout := NewMockLogoutUseCase(ctrl)
				mockLogout.EXPECT().
					Logout(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))
				return NewService(nil, nil, nil, mockLogout)
			},
			req:          &authv1.LogoutRequest{SessionToken: "token"},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).Logout(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func TestServiceValidateSessionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID, _ := user.NewID()
	mockValidate := NewMockValidateSessionUseCase(ctrl)
	mockValidate.EXPECT().
		Validate(gomock.Any(), &appsession.ValidateSessionRequest{SessionToken: "token"}).
		Return(&appsession.ValidateSessionResult{UserID: userID}, nil)

	svc := NewService(nil, nil, mockValidate, nil)

	resp, err := svc.ValidateSession(context.Background(), &authv1.ValidateSessionRequest{SessionToken: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetUserId() != userID.String() {
		t.Fatalf("expected user id %s, got %s", userID.String(), resp.GetUserId())
	}
}

func TestServiceValidateSessionError(t *testing.T) {
	tests := []struct {
		name         string
		service      func(ctrl *gomock.Controller) *Service
		req          *authv1.ValidateSessionRequest
		expectedCode connect.Code
	}{
		{
			name:         "handler missing",
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil) },
			req:          &authv1.ValidateSessionRequest{SessionToken: "token"},
			expectedCode: connect.CodeFailedPrecondition,
		},
		{
			name: "token required",
			service: func(ctrl *gomock.Controller) *Service {
				mockValidate := NewMockValidateSessionUseCase(ctrl)
				mockValidate.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil, appsession.ErrTokenRequired)
				return NewService(nil, nil, mockValidate, nil)
			},
			req:          &authv1.ValidateSessionRequest{SessionToken: ""},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "invalid token",
			service: func(ctrl *gomock.Controller) *Service {
				mockValidate := NewMockValidateSessionUseCase(ctrl)
				mockValidate.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil, appsession.ErrInvalidToken)
				return NewService(nil, nil, mockValidate, nil)
			},
			req:          &authv1.ValidateSessionRequest{SessionToken: "bad"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "session missing",
			service: func(ctrl *gomock.Controller) *Service {
				mockValidate := NewMockValidateSessionUseCase(ctrl)
				mockValidate.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil, appsession.ErrSessionNotFound)
				return NewService(nil, nil, mockValidate, nil)
			},
			req:          &authv1.ValidateSessionRequest{SessionToken: "token"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "session expired",
			service: func(ctrl *gomock.Controller) *Service {
				mockValidate := NewMockValidateSessionUseCase(ctrl)
				mockValidate.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil, appsession.ErrSessionExpired)
				return NewService(nil, nil, mockValidate, nil)
			},
			req:          &authv1.ValidateSessionRequest{SessionToken: "token"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "unexpected error",
			service: func(ctrl *gomock.Controller) *Service {
				mockValidate := NewMockValidateSessionUseCase(ctrl)
				mockValidate.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))
				return NewService(nil, nil, mockValidate, nil)
			},
			req:          &authv1.ValidateSessionRequest{SessionToken: "token"},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).ValidateSession(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}
