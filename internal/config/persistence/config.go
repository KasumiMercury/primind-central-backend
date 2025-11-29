package persistence

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const (
	postgresDSNEnv   = "POSTGRES_DSN"
	redisAddrEnv     = "REDIS_ADDR"
	redisPasswordEnv = "REDIS_PASSWORD"
	redisDBEnv       = "REDIS_DB"

	defaultRedisAddr = "localhost:6379"
	defaultRedisDB   = 0
)

var (
	ErrPostgresDSNMissing = errors.New("postgres dsn is required")
	ErrRedisAddrMissing   = errors.New("redis address is required")
	ErrInvalidRedisDB     = errors.New("invalid redis db value")
)

type Config struct {
	PostgresDSN   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() (*Config, error) {
	pgDSN := os.Getenv(postgresDSNEnv)
	if pgDSN == "" {
		return nil, fmt.Errorf("%w: %s", ErrPostgresDSNMissing, postgresDSNEnv)
	}

	redisAddr := os.Getenv(redisAddrEnv)
	if redisAddr == "" {
		redisAddr = defaultRedisAddr
	}

	redisPassword := os.Getenv(redisPasswordEnv)

	redisDB := defaultRedisDB

	if raw := os.Getenv(redisDBEnv); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, ErrInvalidRedisDB
		}

		redisDB = parsed
	}

	cfg := &Config{
		PostgresDSN:   pgDSN,
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrPostgresDSNMissing)
	}

	if c.PostgresDSN == "" {
		return ErrPostgresDSNMissing
	}

	if c.RedisAddr == "" {
		return ErrRedisAddrMissing
	}

	return nil
}
