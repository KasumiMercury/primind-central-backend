package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	authrepository "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/config"
	devicemodule "github.com/KasumiMercury/primind-central-backend/internal/device"
	deviceconfig "github.com/KasumiMercury/primind-central-backend/internal/device/config"
	devicerepository "github.com/KasumiMercury/primind-central-backend/internal/device/infra/repository"
	taskmodule "github.com/KasumiMercury/primind-central-backend/internal/task"
	taskconfig "github.com/KasumiMercury/primind-central-backend/internal/task/config"
	taskrepository "github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
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

	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Warn("failed to close postgres connection", slog.String("error", err.Error()))
		}
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     appCfg.Persistence.RedisAddr,
		Password: appCfg.Persistence.RedisPassword,
		DB:       appCfg.Persistence.RedisDB,
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("failed to close redis client", slog.String("error", err.Error()))
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect redis", slog.String("error", err.Error()))
		os.Exit(1)
	}

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx, authmodule.Repositories{
		Params:       authrepository.NewOIDCParamsRepository(redisClient),
		Sessions:     authrepository.NewSessionRepository(redisClient),
		Users:        authrepository.NewUserRepository(db),
		OIDCIdentity: authrepository.NewOIDCIdentityRepository(db),
		UserIdentity: authrepository.NewUserWithIdentityRepository(db),
	})
	if err != nil {
		logger.Error("failed to initialize auth service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	mux.Handle(authPath, authHandler)

	taskCfg, err := taskconfig.Load()
	if err != nil {
		logger.Error("failed to load task config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	taskPath, taskHandler, err := taskmodule.NewHTTPHandler(
		ctx,
		taskrepository.NewTaskRepository(db),
		taskCfg.AuthServiceURL,
		taskCfg.DeviceServiceURL,
	)
	if err != nil {
		logger.Error("failed to initialize task service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	mux.Handle(taskPath, taskHandler)

	deviceCfg, err := deviceconfig.Load()
	if err != nil {
		logger.Error("failed to load device config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	devicePath, deviceHandler, err := devicemodule.NewHTTPHandler(
		ctx,
		devicerepository.NewDeviceRepository(db),
		deviceCfg.AuthServiceURL,
	)
	if err != nil {
		logger.Error("failed to initialize device service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	mux.Handle(devicePath, deviceHandler)

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
