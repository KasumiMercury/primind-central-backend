package auth

import (
	"context"
	"errors"
	"fmt"

	connect "connectrpc.com/connect"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Service struct {
	oidcParams appoidc.OIDCParamsGenerator
	oidcLogin  appoidc.OIDCLoginUseCase
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

func NewService(oidcParamsGenerator appoidc.OIDCParamsGenerator, oidcLoginUseCase appoidc.OIDCLoginUseCase) *Service {
	return &Service{
		oidcParams: oidcParamsGenerator,
		oidcLogin:  oidcLoginUseCase,
	}
}

func (s *Service) OIDCParams(ctx context.Context, req *authv1.OIDCParamsRequest) (*authv1.OIDCParamsResponse, error) {
	if s.oidcParams == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, appoidc.ErrOIDCNotConfigured)
	}

	providerID, err := mapProvider(req.GetProvider())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := s.oidcParams.Generate(ctx, providerID)
	if err != nil {
		switch {
		case errors.Is(err, appoidc.ErrOIDCNotConfigured):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		case errors.Is(err, appoidc.ErrProviderUnsupported):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return &authv1.OIDCParamsResponse{
		AuthorizationUrl: result.AuthorizationURL,
		State:            result.State,
	}, nil
}

func (s *Service) OIDCLogin(ctx context.Context, req *authv1.OIDCLoginRequest) (*authv1.OIDCLoginResponse, error) {
	if s.oidcLogin == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, appoidc.ErrOIDCNotConfigured)
	}

	providerID, err := mapProvider(req.GetProvider())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	loginReq := &appoidc.LoginRequest{
		Provider: providerID,
		Code:     req.GetCode(),
		State:    req.GetState(),
	}

	result, err := s.oidcLogin.Login(ctx, loginReq)
	if err != nil {
		switch {
		case errors.Is(err, appoidc.ErrOIDCNotConfigured):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		case errors.Is(err, appoidc.ErrProviderUnsupported):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidCode):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidState):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, domainoidc.ErrParamsExpired):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, appoidc.ErrInvalidNonce):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

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
