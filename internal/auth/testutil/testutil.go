package testutil

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	postgresmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	redismodule "github.com/testcontainers/testcontainers-go/modules/redis"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupRedisContainer(ctx context.Context, t *testing.T) (*redis.Client, func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("failed to start redis container (docker unavailable?): %v", r)
		}
	}()

	container, err := redismodule.Run(ctx, "redis:8-alpine")
	if err != nil {
		t.Skipf("failed to start redis container (docker unavailable?): %v", err)
	}

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Skipf("failed to get redis endpoint: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	cleanup := func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close redis client: %v", err)
		}

		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	}

	return client, cleanup
}

func SetupPostgresContainer(ctx context.Context, t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("failed to start postgres container (docker unavailable?): %v", r)
		}
	}()

	container, err := postgresmodule.Run(ctx,
		"postgres:18-alpine",
		postgresmodule.WithDatabase("testdb"),
		postgresmodule.WithUsername("testuser"),
		postgresmodule.WithPassword("testpass"),
		postgresmodule.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("failed to start postgres container (docker unavailable?): %v", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Skipf("failed to get postgres connection string: %v", err)
	}

	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Skipf("failed to connect to postgres: %v", err)
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return db, cleanup
}
