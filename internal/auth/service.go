package auth

import (
	"context"
	"errors"

	connect "connectrpc.com/connect"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Service struct {
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

func NewService() *Service {
	return &Service{}
}

func (s *Service) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.Login not implemented"))
}

func (s *Service) HandleCallback(ctx context.Context, req *authv1.HandleCallbackRequest) (*authv1.HandleCallbackResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("auth.HandleCallback not implemented"))
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
