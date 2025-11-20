package oidc

import "context"

type ParamsRepository interface {
	SaveParams(ctx context.Context, params Params) error
}
