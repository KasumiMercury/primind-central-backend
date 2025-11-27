package jwt

import (
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/crypto/sha3"

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

	expiresAt := session.ExpiresAt()
	if expiresAt.IsZero() {
		expiresAt = now.Add(g.sessionCfg.Duration)
	}

	claims := jwt.Claims{
		ID:       session.ID().String(),
		Subject:  session.UserID(),
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
	key := deriveHMACKey(g.sessionCfg.Secret)

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	claims := &jwt.Claims{}
	if err := parsed.Claims(key, claims); err != nil {
		return nil, err
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, err
	}

	return claims, nil
}

func deriveHMACKey(secret string) []byte {
	sum := sha3.Sum256([]byte(secret))
	return sum[:]
}
