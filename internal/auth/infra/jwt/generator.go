package jwt

import (
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type Generator struct {
	secret   []byte
	duration time.Duration
}

func NewGenerator(secret string, duration time.Duration) *Generator {
	return &Generator{
		secret:   []byte(secret),
		duration: duration,
	}
}

type Claims struct {
	Sub  string `json:"sub"`
	Name string `json:"name"`
	jwt.Claims
}

func (g *Generator) Generate(sub, name string) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: g.secret},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		Sub:  sub,
		Name: name,
		Claims: jwt.Claims{
			IssuedAt: jwt.NewNumericDate(now),
			Expiry:   jwt.NewNumericDate(now.Add(g.duration)),
		},
	}

	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (g *Generator) Verify(token string) (*Claims, error) {
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	if err := parsed.Claims(g.secret, claims); err != nil {
		return nil, err
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, err
	}

	return claims, nil
}
