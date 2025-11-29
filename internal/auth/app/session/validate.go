package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
)

var (
	ErrTokenRequired   = errors.New("session token is required")
	ErrInvalidToken    = errors.New("invalid session token")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

type TokenVerifier interface {
	Verify(token string) error
	ExtractSessionID(token string) (string, error)
}

type ValidateSessionRequest struct {
	SessionToken string
}

type ValidateSessionResult struct {
	UserID user.ID
}

type ValidateSessionUseCase interface {
	Validate(ctx context.Context, req *ValidateSessionRequest) (*ValidateSessionResult, error)
}

type validateSessionHandler struct {
	sessionRepo   domainsession.SessionRepository
	tokenVerifier TokenVerifier
	clock         clock.Clock
	logger        *slog.Logger
}

func NewValidateSessionHandler(
	sessionRepo domainsession.SessionRepository,
	tokenVerifier TokenVerifier,
) ValidateSessionUseCase {
	return newValidateSessionHandler(sessionRepo, tokenVerifier, &clock.RealClock{})
}

func newValidateSessionHandler(
	sessionRepo domainsession.SessionRepository,
	tokenVerifier TokenVerifier,
	clk clock.Clock,
) ValidateSessionUseCase {
	return &validateSessionHandler{
		sessionRepo:   sessionRepo,
		tokenVerifier: tokenVerifier,
		clock:         clk,
		logger:        slog.Default().WithGroup("auth").WithGroup("session").WithGroup("validate"),
	}
}

func (h *validateSessionHandler) Validate(ctx context.Context, req *ValidateSessionRequest) (*ValidateSessionResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidToken)
	}

	if req.SessionToken == "" {
		h.logger.Warn("validate session called with empty token")

		return nil, ErrTokenRequired
	}

	if err := h.tokenVerifier.Verify(req.SessionToken); err != nil {
		h.logger.Info("session token verification failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	rawSessionID, err := h.tokenVerifier.ExtractSessionID(req.SessionToken)
	if err != nil {
		h.logger.Info("session id extraction failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	sessionID, err := domainsession.ParseID(rawSessionID)
	if err != nil {
		h.logger.Info("session id in token is invalid", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	session, err := h.sessionRepo.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.Info("session not found for validated token", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrSessionNotFound, err)
	}

	now := h.clock.Now()
	if !session.ExpiresAt().After(now) {
		h.logger.Info("session has expired")

		return nil, ErrSessionExpired
	}

	return &ValidateSessionResult{
		UserID: session.UserID(),
	}, nil
}
