package oidc

import (
	"context"
	"errors"
)

var ErrParamsNotFound = errors.New("params not found")

type ParamsRepository interface {
	SaveParams(ctx context.Context, params Params) error
	GetParamsByState(ctx context.Context, state string) (*Params, error)
}
