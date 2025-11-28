package jwt

import (
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/crypto/sha3"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
)

type SessionClaims struct {
	jwt.Claims
	Color string `json:"color,omitempty"`
}

type SessionJWTGenerator struct {
	sessionCfg *sessionCfg.Config
}

func NewSessionJWTGenerator(cfg *sessionCfg.Config) *SessionJWTGenerator {
	return &SessionJWTGenerator{
		sessionCfg: cfg,
	}
}

func (g *SessionJWTGenerator) Generate(session *domain.Session, u *user.User) (string, error) {
	if u == nil {
		return "", fmt.Errorf("user is required for session token generation")
	}

	userColor := u.Color()
	if err := userColor.Validate(); err != nil {
		return "", fmt.Errorf("invalid user color: %w", err)
	}

	key := deriveHMACKey(g.sessionCfg.Secret)

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key}, nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT signer: %w", err)
	}

	now := session.CreatedAt()
	if now.IsZero() {
		now = time.Now()
	}

	claims := SessionClaims{
		Claims: jwt.Claims{
			ID:       session.ID().String(),
			IssuedAt: jwt.NewNumericDate(now),
			Expiry:   jwt.NewNumericDate(session.ExpiresAt()),
		},
		Color: userColor.String(),
	}

	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (v *SessionJWTValidator) parseClaims(token string) (*SessionClaims, error) {
	key := deriveHMACKey(v.sessionCfg.Secret)

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	//exhaustruct:ignore
	claims := &SessionClaims{}
	if err := parsed.Claims(key, claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func deriveHMACKey(secret string) []byte {
	sum := sha3.Sum256([]byte(secret))

	return sum[:]
}

type SessionJWTValidator struct {
	sessionCfg *sessionCfg.Config
}

func NewSessionJWTValidator(cfg *sessionCfg.Config) *SessionJWTValidator {
	return &SessionJWTValidator{
		sessionCfg: cfg,
	}
}

func (v *SessionJWTValidator) Verify(token string) error {
	claims, err := v.parseClaims(token)
	if err != nil {
		return err
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return err
	}

	return nil
}

func (v *SessionJWTValidator) ExtractSessionID(token string) (string, error) {
	claims, err := v.parseClaims(token)
	if err != nil {
		return "", err
	}

	if claims == nil || claims.ID == "" {
		return "", fmt.Errorf("session id missing in token")
	}

	return claims.ID, nil
}
