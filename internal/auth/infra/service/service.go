package auth

import (
	"context"
	"errors"
	"fmt"

	connect "connectrpc.com/connect"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	oidcctrl "github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Service struct {
	config     *config.AuthConfig
	oidcParams oidc.OIDCParamsGenerator
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

func NewService(cfg *config.AuthConfig, oidcParamsGenerator oidc.OIDCParamsGenerator) *Service {
	return &Service{
		config:     cfg,
		oidcParams: oidcParamsGenerator,
	}
}

func (s *Service) OIDCParams(ctx context.Context, req *authv1.OIDCParamsRequest) (*authv1.OIDCParamsResponse, error) {
	providerID, err := mapProvider(req.GetProvider())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := s.oidcParams.Generate(ctx, providerID)
	if err != nil {
		switch {
		case errors.Is(err, oidcctrl.ErrOIDCNotConfigured):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		case errors.Is(err, oidcctrl.ErrProviderUnsupported):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return &authv1.OIDCParamsResponse{
		AuthorizationUrl: result.AuthorizationURL,
		ClientId:         result.ClientID,
		RedirectUri:      result.RedirectURI,
		Scope:            result.Scope,
	}, nil
}

func (s *Service) OIDCLogin(ctx context.Context, req *authv1.OIDCLoginRequest) (*authv1.OIDCLoginResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.OIDCLogin not implemented"))
}

func (s *Service) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.Logout not implemented"))
}

func (s *Service) GetSession(ctx context.Context, req *authv1.GetSessionRequest) (*authv1.GetSessionResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.GetSession not implemented"))
}

func (s *Service) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.GetUser not implemented"))
}

func (s *Service) GetUserByUsername(ctx context.Context, req *authv1.GetUserByUsernameRequest) (*authv1.GetUserByUsernameResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.GetUserByUsername not implemented"))
}

func (s *Service) GetCurrentUser(ctx context.Context, req *authv1.GetCurrentUserRequest) (*authv1.GetCurrentUserResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.GetCurrentUser not implemented"))
}

func mapProvider(provider authv1.OIDCProvider) (oidccfg.ProviderID, error) {
	switch provider {
	case authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE:
		return oidccfg.ProviderGoogle, nil
	case authv1.OIDCProvider_OIDC_PROVIDER_UNSPECIFIED:
		return "", fmt.Errorf("oidc provider is required")
	default:
		return "", fmt.Errorf("unsupported oidc provider: %s", provider.String())
	}
}
