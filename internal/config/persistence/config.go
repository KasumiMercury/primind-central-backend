package persistence

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const (
	postgresDSNEnv   = "AUTH_POSTGRES_DSN"
	redisAddrEnv     = "AUTH_REDIS_ADDR"
	redisPasswordEnv = "AUTH_REDIS_PASSWORD"
	redisDBEnv       = "AUTH_REDIS_DB"

	defaultRedisAddr = "localhost:6379"
	defaultRedisDB   = 0
)

var (
	ErrPostgresDSNMissing = errors.New("postgres dsn is required")
	ErrRedisAddrMissing   = errors.New("redis address is required")
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
		if parsed, err := strconv.Atoi(raw); err == nil {
			redisDB = parsed
		}
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
