package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	connect "connectrpc.com/connect"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Service struct {
	oidcParams appoidc.OIDCParamsGenerator
	oidcLogin  appoidc.OIDCLoginUseCase
	logger     *slog.Logger
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

func NewService(oidcParamsGenerator appoidc.OIDCParamsGenerator, oidcLoginUseCase appoidc.OIDCLoginUseCase) *Service {
	return &Service{
		oidcParams: oidcParamsGenerator,
		oidcLogin:  oidcLoginUseCase,
		logger:     slog.Default().WithGroup("auth").WithGroup("service"),
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
		case errors.Is(err, appoidc.ErrProviderUnsupported):
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
		case errors.Is(err, appoidc.ErrProviderUnsupported):
			s.logger.Warn("oidc login requested for unsupported provider", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidCode):
			s.logger.Warn("login failed due to invalid authorization code", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidState):
			s.logger.Warn("login failed due to invalid state", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, domainoidc.ErrParamsExpired):
			s.logger.Warn("login failed due to expired params", slog.String("provider", string(providerID)))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidNonce):
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
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.Logout not implemented"))
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
