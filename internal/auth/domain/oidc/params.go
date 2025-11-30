package oidc

import "time"

type ProviderID string

const (
	ProviderGoogle ProviderID = "google"

	ParamsExpirationDuration = 10 * time.Minute
)

type Params struct {
	provider     ProviderID
	state        string
	nonce        string
	codeVerifier string
	createdAt    time.Time
}

func NewParams(provider ProviderID, state, nonce, codeVerifier string, createdAt time.Time) (*Params, error) {
	if provider == "" {
		return nil, ErrProviderInvalid
	}

	if state == "" {
		return nil, ErrStateEmpty
	}

	if nonce == "" {
		return nil, ErrNonceEmpty
	}

	if codeVerifier == "" {
		return nil, ErrCodeVerifierEmpty
	}

	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Params{
		provider:     provider,
		state:        state,
		nonce:        nonce,
		codeVerifier: codeVerifier,
		createdAt:    createdAt,
	}, nil
}

func (p *Params) Provider() ProviderID {
	return p.provider
}

func (p *Params) State() string {
	return p.state
}

func (p *Params) Nonce() string {
	return p.nonce
}

func (p *Params) CodeVerifier() string {
	return p.codeVerifier
}

func (p *Params) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Params) ExpiresAt() time.Time {
	return p.createdAt.Add(ParamsExpirationDuration)
}

func (p *Params) IsExpired(now time.Time) bool {
	return now.After(p.ExpiresAt())
}
