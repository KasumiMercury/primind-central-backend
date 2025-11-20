package oidc

import "context"

type ParamsRepository interface {
	SaveParams(ctx context.Context, params Params) error
	GetParamsByState(ctx context.Context, state string) (*Params, error)
}
