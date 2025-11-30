package oidc

import (
	"context"
	"errors"
)

//go:generate mockgen -source=params_repository.go -destination=mock_params_repository.go -package=oidc

var ErrParamsNotFound = errors.New("params not found")

type ParamsRepository interface {
	SaveParams(ctx context.Context, params *Params) error
	GetParamsByState(ctx context.Context, state string) (*Params, error)
}
