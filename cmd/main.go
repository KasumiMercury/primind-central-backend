package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	mux := http.NewServeMux()

	appCfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(appCfg.Persistence.PostgresDSN), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to obtain postgres handle", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer sqlDB.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     appCfg.Persistence.RedisAddr,
		Password: appCfg.Persistence.RedisPassword,
		DB:       appCfg.Persistence.RedisDB,
	})

	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect redis", slog.String("error", err.Error()))
		os.Exit(1)
	}

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx, authmodule.Repositories{
		Params:       repository.NewOIDCParamsRepository(redisClient),
		Sessions:     repository.NewSessionRepository(redisClient),
		Users:        repository.NewUserRepository(db),
		OIDCIdentity: repository.NewOIDCIdentityRepository(db),
		UserIdentity: repository.NewUserWithIdentityRepository(db),
	})
	if err != nil {
		logger.Error("failed to initialize auth service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	mux.Handle(authPath, authHandler)

	addr := ":8080"
	logger.Info("starting Connect API server", slog.String("address", addr))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Error("connect api server exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
