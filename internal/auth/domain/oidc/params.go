package oidc

import (
	"errors"
	"time"
)

type ProviderID string

const (
	ProviderGoogle ProviderID = "google"
)

var (
	ErrProviderInvalid = errors.New("provider must be specified")
	ErrStateEmpty      = errors.New("state must be specified")
	ErrNonceEmpty      = errors.New("nonce must be specified")
)

type Params struct {
	provider  ProviderID
	state     string
	nonce     string
	createdAt time.Time
}

func NewParams(provider ProviderID, state, nonce string, createdAt time.Time) (*Params, error) {
	if provider == "" {
		return nil, ErrProviderInvalid
	}
	if state == "" {
		return nil, ErrStateEmpty
	}
	if nonce == "" {
		return nil, ErrNonceEmpty
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Params{
		provider:  provider,
		state:     state,
		nonce:     nonce,
		createdAt: createdAt,
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

func (p *Params) CreatedAt() time.Time {
	return p.createdAt
}
