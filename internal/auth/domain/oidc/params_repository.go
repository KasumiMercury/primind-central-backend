package oidc

import "context"

//go:generate mockgen -source=params_repository.go -destination=mock_params_repository.go -package=oidc

type ParamsRepository interface {
	SaveParams(ctx context.Context, params *Params) error
	GetParamsByState(ctx context.Context, state string) (*Params, error)
}
