package persistence

import "errors"

var (
	ErrPostgresDSNMissing = errors.New("postgres dsn is required")
	ErrRedisAddrMissing   = errors.New("redis address is required")
	ErrInvalidRedisDB     = errors.New("invalid redis db value")
)
