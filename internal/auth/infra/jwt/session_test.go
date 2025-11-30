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
		Secret:   "test-secret-2",
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

func TestSessionJWTDoesNotIncludeUserIDInSubject(t *testing.T) {
	t.Parallel()

	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-3",
	}

	generator := NewSessionJWTGenerator(cfg)

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	color := user.MustColor("#abcdef")
	u := user.NewUser(userID, color)

	now := time.Now().UTC()

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

	var claims jwt.Claims
	if err := parsed.Claims(deriveHMACKey(cfg.Secret), &claims); err != nil {
		t.Fatalf("failed to extract claims: %v", err)
	}

	if claims.Subject != "" {
		t.Fatalf("expected empty subject, got %q", claims.Subject)
	}
}

func TestSessionJWTGeneratorGenerateSuccess(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-success",
	}
	generator := NewSessionJWTGenerator(cfg)

	tests := []struct {
		name string
	}{
		{name: "generates valid token"},
		{name: "token is non-empty"},
		{name: "token contains session ID"},
		{name: "token contains expiry"},
		{name: "token contains issued-at"},
		{name: "token contains color claim"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := user.NewID()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			color := user.MustColor("#ABCDEF")
			u := user.NewUser(userID, color)

			now := time.Now().UTC().Truncate(time.Second)
			expiresAt := now.Add(time.Hour)

			session, err := domain.NewSession(u.ID(), now, expiresAt)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			token, err := generator.Generate(session, u)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			if token == "" {
				t.Fatal("Generate() returned empty token")
			}

			parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
			if err != nil {
				t.Fatalf("failed to parse token: %v", err)
			}

			var claims SessionClaims
			if err := parsed.Claims(deriveHMACKey(cfg.Secret), &claims); err != nil {
				t.Fatalf("failed to extract claims: %v", err)
			}

			if claims.ID != session.ID().String() {
				t.Errorf("claims.ID = %q, want %q", claims.ID, session.ID().String())
			}

			if claims.Expiry == nil {
				t.Error("claims.Expiry is nil")
			} else if !claims.Expiry.Time().Equal(expiresAt) {
				t.Errorf("claims.Expiry.Time() = %v, want %v", claims.Expiry.Time(), expiresAt)
			}

			if claims.IssuedAt == nil {
				t.Error("claims.IssuedAt is nil")
			} else if !claims.IssuedAt.Time().Equal(now) {
				t.Errorf("claims.IssuedAt.Time() = %v, want %v", claims.IssuedAt.Time(), now)
			}

			if claims.Color != color.String() {
				t.Errorf("claims.Color = %q, want %q", claims.Color, color.String())
			}
		})
	}
}

func TestSessionJWTGeneratorGenerateErrors(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-errors",
	}
	generator := NewSessionJWTGenerator(cfg)

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	color := user.MustColor("#123456")
	validUser := user.NewUser(userID, color)

	now := time.Now().UTC()
	validSession, err := domain.NewSession(userID, now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name        string
		session     *domain.Session
		user        *user.User
		expectedErr error
	}{
		{
			name:        "nil session",
			session:     nil,
			user:        validUser,
			expectedErr: ErrSessionRequiredForTokenForToken,
		},
		{
			name:        "nil user",
			session:     validSession,
			user:        nil,
			expectedErr: ErrUserRequiredForTokenForToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := generator.Generate(tt.session, tt.user)

			if err == nil {
				t.Fatalf("Generate() expected error %v, got token: %s", tt.expectedErr, token)
			}

			if token != "" {
				t.Errorf("Generate() expected empty token on error, got: %s", token)
			}

			if tt.expectedErr != nil {
				if err != tt.expectedErr && !ErrorContains(err, tt.expectedErr.Error()) {
					t.Errorf("Generate() error = %v, want %v", err, tt.expectedErr)
				}
			}
		})
	}
}

func TestSessionJWTValidatorVerifySuccess(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-validator",
	}
	generator := NewSessionJWTGenerator(cfg)
	validator := NewSessionJWTValidator(cfg)

	tests := []struct {
		name string
	}{
		{name: "valid token passes verification"},
		{name: "round-trip generate and verify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := user.NewID()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			color := user.MustColor("#FF0000")
			u := user.NewUser(userID, color)

			now := time.Now().UTC().Truncate(time.Second)
			session, err := domain.NewSession(u.ID(), now, now.Add(time.Hour))
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			token, err := generator.Generate(session, u)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			err = validator.Verify(token)
			if err != nil {
				t.Errorf("Verify() unexpected error: %v", err)
			}
		})
	}
}

func TestSessionJWTValidatorVerifyErrors(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-verify-errors",
	}
	generator := NewSessionJWTGenerator(cfg)
	validator := NewSessionJWTValidator(cfg)

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	color := user.MustColor("#00FF00")
	u := user.NewUser(userID, color)

	past := time.Now().UTC().Add(-2 * time.Hour)
	expiredSession, err := domain.NewSession(u.ID(), past, past.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	expiredToken, err := generator.Generate(expiredSession, u)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "invalid token format",
			token:       "not-a-jwt-token",
			expectError: true,
		},
		{
			name:        "expired token",
			token:       expiredToken,
			expectError: true,
		},
		{
			name:        "malformed JWT",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Verify(tt.token)

			if tt.expectError && err == nil {
				t.Error("Verify() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Verify() unexpected error: %v", err)
			}
		})
	}
}

func TestSessionJWTExtractSessionIDSuccess(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-extract",
	}
	generator := NewSessionJWTGenerator(cfg)
	validator := NewSessionJWTValidator(cfg)

	tests := []struct {
		name string
	}{
		{name: "extracts session ID from valid token"},
		{name: "session ID matches original"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := user.NewID()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			color := user.MustColor("#0000FF")
			u := user.NewUser(userID, color)

			now := time.Now().UTC()
			session, err := domain.NewSession(u.ID(), now, now.Add(time.Hour))
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			expectedSessionID := session.ID().String()

			token, err := generator.Generate(session, u)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			extractedID, err := validator.ExtractSessionID(token)
			if err != nil {
				t.Fatalf("ExtractSessionID() error: %v", err)
			}

			if extractedID != expectedSessionID {
				t.Errorf("ExtractSessionID() = %q, want %q", extractedID, expectedSessionID)
			}
		})
	}
}

func TestSessionJWTExtractSessionIDErrors(t *testing.T) {
	cfg := &sessionCfg.Config{
		Duration: time.Hour,
		Secret:   "test-secret-extract-errors",
	}
	validator := NewSessionJWTValidator(cfg)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "invalid token",
			token:       "invalid-jwt",
			expectError: true,
		},
		{
			name:        "malformed JWT",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, err := validator.ExtractSessionID(tt.token)

			if tt.expectError && err == nil {
				t.Errorf("ExtractSessionID() expected error, got sessionID: %s", sessionID)
			}

			if !tt.expectError && err != nil {
				t.Errorf("ExtractSessionID() unexpected error: %v", err)
			}

			if tt.expectError && sessionID != "" {
				t.Errorf("ExtractSessionID() expected empty sessionID on error, got: %s", sessionID)
			}
		})
	}
}

func ErrorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return Contains(err.Error(), substr)
}

func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
