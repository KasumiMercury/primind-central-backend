package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	connect "connectrpc.com/connect"
	applogout "github.com/KasumiMercury/primind-central-backend/internal/auth/app/logout"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	appsession "github.com/KasumiMercury/primind-central-backend/internal/auth/app/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Service struct {
	oidcParams      appoidc.OIDCParamsGenerator
	oidcLogin       appoidc.OIDCLoginUseCase
	validateSession appsession.ValidateSessionUseCase
	logout          applogout.LogoutUseCase
	logger          *slog.Logger
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

func NewService(
	oidcParamsGenerator appoidc.OIDCParamsGenerator,
	oidcLoginUseCase appoidc.OIDCLoginUseCase,
	validateSessionUseCase appsession.ValidateSessionUseCase,
	logoutUseCase applogout.LogoutUseCase,
) *Service {
	return &Service{
		oidcParams:      oidcParamsGenerator,
		oidcLogin:       oidcLoginUseCase,
		validateSession: validateSessionUseCase,
		logout:          logoutUseCase,
		logger:          slog.Default().WithGroup("auth").WithGroup("service"),
	}
}

func (s *Service) OIDCParams(ctx context.Context, req *authv1.OIDCParamsRequest) (*authv1.OIDCParamsResponse, error) {
	if s.oidcParams == nil {
		s.logger.Warn("oidc params requested but generator is not configured")

		return nil, connect.NewError(connect.CodeFailedPrecondition, appoidc.ErrOIDCNotConfigured)
	}

	providerID, err := mapProvider(req.GetProvider())
	if err != nil {
		s.logger.Warn("invalid provider in params request", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	s.logger.Debug("handling oidc params request", slog.String("provider", string(providerID)))

	result, err := s.oidcParams.Generate(ctx, providerID)
	if err != nil {
		switch {
		case errors.Is(err, appoidc.ErrOIDCNotConfigured):
			s.logger.Warn("oidc not configured during params request")

			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		case errors.Is(err, appoidc.ErrOIDCProviderUnsupported):
			s.logger.Warn("oidc params requested for unsupported provider", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			s.logger.Error("failed to generate oidc params", slog.String("error", err.Error()), slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	s.logger.Debug("oidc params generated", slog.String("provider", string(providerID)))

	return &authv1.OIDCParamsResponse{
		AuthorizationUrl: result.AuthorizationURL,
		State:            result.State,
	}, nil
}

func (s *Service) OIDCLogin(ctx context.Context, req *authv1.OIDCLoginRequest) (*authv1.OIDCLoginResponse, error) {
	if s.oidcLogin == nil {
		s.logger.Warn("oidc login requested but handler is not configured")

		return nil, connect.NewError(connect.CodeFailedPrecondition, appoidc.ErrOIDCNotConfigured)
	}

	providerID, err := mapProvider(req.GetProvider())
	if err != nil {
		s.logger.Warn("invalid provider in login request", slog.String("error", err.Error()))

		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	s.logger.Debug("handling oidc login request", slog.String("provider", string(providerID)))

	loginReq := &appoidc.LoginRequest{
		Provider: providerID,
		Code:     req.GetCode(),
		State:    req.GetState(),
	}

	result, err := s.oidcLogin.Login(ctx, loginReq)
	if err != nil {
		switch {
		case errors.Is(err, appoidc.ErrOIDCNotConfigured):
			s.logger.Warn("oidc not configured during login")

			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		case errors.Is(err, appoidc.ErrOIDCProviderUnsupported):
			s.logger.Warn("oidc login requested for unsupported provider", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrCodeInvalid):
			s.logger.Warn("login failed due to invalid authorization code", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrStateInvalid):
			s.logger.Warn("login failed due to invalid state", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, domainoidc.ErrParamsExpired):
			s.logger.Warn("login failed due to expired params", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrNonceInvalid):
			s.logger.Warn("login failed due to nonce mismatch", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			s.logger.Error("unexpected login failure", slog.String("error", err.Error()), slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	s.logger.Info("oidc login succeeded", slog.String("provider", string(providerID)))

	return &authv1.OIDCLoginResponse{
		SessionToken: result.SessionToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if s.logout == nil {
		s.logger.Warn("logout requested but handler is not configured")

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("logout not configured"))
	}

	sessionToken := req.GetSessionToken()

	result, err := s.logout.Logout(ctx, &applogout.LogoutRequest{
		SessionToken: sessionToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, applogout.ErrSessionTokenRequired),
			errors.Is(err, applogout.ErrSessionTokenInvalid):
			s.logger.Info("logout failed", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			s.logger.Error("unexpected logout error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return &authv1.LogoutResponse{
		Success: result.Success,
	}, nil
}

func (s *Service) ValidateSession(ctx context.Context, req *authv1.ValidateSessionRequest) (*authv1.ValidateSessionResponse, error) {
	if s.validateSession == nil {
		s.logger.Warn("validate session requested but handler is not configured")

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("session validation not configured"))
	}

	result, err := s.validateSession.Validate(ctx, &appsession.ValidateSessionRequest{
		SessionToken: req.GetSessionToken(),
	})
	if err != nil {
		switch {
		case errors.Is(err, appsession.ErrSessionTokenRequired),
			errors.Is(err, appsession.ErrSessionTokenInvalid),
			errors.Is(err, appsession.ErrSessionNotFound),
			errors.Is(err, appsession.ErrSessionExpired):
			s.logger.Info("session validation failed", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		default:
			s.logger.Error("unexpected session validation error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return &authv1.ValidateSessionResponse{
		UserId: result.UserID.String(),
	}, nil
}

func mapProvider(provider authv1.OIDCProvider) (domainoidc.ProviderID, error) {
	switch provider {
	case authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE:
		return domainoidc.ProviderGoogle, nil
	case authv1.OIDCProvider_OIDC_PROVIDER_UNSPECIFIED:
		return "", fmt.Errorf("oidc provider is required")
	default:
		return "", fmt.Errorf("unsupported oidc provider: %s", provider.String())
	}
}
