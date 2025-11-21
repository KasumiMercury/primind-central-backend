package sessionjwt

import (
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
)

type SessionJWTGenerator struct {
	sessionCfg *sessionCfg.Config
}

func NewSessionJWTGenerator(cfg *sessionCfg.Config) *SessionJWTGenerator {
	return &SessionJWTGenerator{
		sessionCfg: cfg,
	}
}

func (g *SessionJWTGenerator) Generate(session *domain.Session) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: g.sessionCfg.Secret},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}

	now := session.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}

	expiresAt := session.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = now.Add(g.sessionCfg.Duration)
	}

	claims := jwt.Claims{
		Subject:  session.UserID,
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(expiresAt),
	}

	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (g *SessionJWTGenerator) Verify(token string) (*jwt.Claims, error) {
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	claims := &jwt.Claims{}
	if err := parsed.Claims(g.sessionCfg.Secret, claims); err != nil {
		return nil, err
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, err
	}

	return claims, nil
}
