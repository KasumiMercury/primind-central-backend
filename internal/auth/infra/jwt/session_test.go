package jwt

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
)

func TestSessionJWTIncludesColorClaim(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret",
	}

	generator := NewSessionJWTGenerator(cfg)

	expectedColor := user.MustColor("#123456")
	userId, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}
	u := user.NewUser(userId, expectedColor)

	now := time.Now().UTC().Truncate(time.Second)
	session, err := domain.NewSession(u.ID(), now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	token, err := generator.Generate(session, u)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	var colorClaims struct {
		Color string `json:"color"`
	}

	if err := parsed.Claims(deriveHMACKey(cfg.Secret), &colorClaims); err != nil {
		t.Fatalf("failed to extract color claim: %v", err)
	}

	if colorClaims.Color != expectedColor.String() {
		t.Fatalf("expected color %s, got %s", expectedColor, colorClaims.Color)
	}
}

func TestSessionJWTGenerateRequiresUser(t *testing.T) {
	t.Parallel()

	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret",
	}

	generator := NewSessionJWTGenerator(cfg)

	uid, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	now := time.Now().UTC()
	session, err := domain.NewSession(uid, now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if _, err := generator.Generate(session, nil); err == nil {
		t.Fatalf("expected error when user is nil")
	}
}
