package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	authrepository "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/config"
	devicemodule "github.com/KasumiMercury/primind-central-backend/internal/device"
	"github.com/KasumiMercury/primind-central-backend/internal/health"
	deviceconfig "github.com/KasumiMercury/primind-central-backend/internal/device/config"
	devicerepository "github.com/KasumiMercury/primind-central-backend/internal/device/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/middleware"
	taskmodule "github.com/KasumiMercury/primind-central-backend/internal/task"
	taskconfig "github.com/KasumiMercury/primind-central-backend/internal/task/config"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/deviceclient"
	taskrepository "github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
	"connectrpc.com/grpchealth"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	obs, err := initObservability(ctx)
	if err != nil {
		slog.Error("failed to initialize observability", slog.String("error", err.Error()))

		return err
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := obs.Shutdown(shutdownCtx); err != nil {
			slog.Warn("failed to shutdown observability", slog.String("error", err.Error()))
		}
	}()

	slog.SetDefault(obs.Logger())

	mux := http.NewServeMux()

	appCfg, err := config.Load()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load config",
			slog.String("event", "config.load.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	db, err := gorm.Open(postgres.Open(appCfg.Persistence.PostgresDSN), &gorm.Config{
		Logger: logging.NewGormLogger(200 * time.Millisecond),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect postgres",
			slog.String("event", "db.connect.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	if err := db.Use(tracing.NewPlugin()); err != nil {
		slog.ErrorContext(ctx, "failed to register GORM tracing plugin",
			slog.String("event", "db.tracing.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.ErrorContext(ctx, "failed to obtain postgres handle",
			slog.String("event", "db.handle.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close postgres connection", slog.String("error", err.Error()))
		}
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     appCfg.Persistence.RedisAddr,
		Password: appCfg.Persistence.RedisPassword,
		DB:       appCfg.Persistence.RedisDB,
	})

	// OpenTelemetry tracing for Redis
	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		slog.ErrorContext(ctx, "failed to instrument redis tracing",
			slog.String("event", "redis.otel.tracing.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	// OpenTelemetry metrics for Redis
	if err := redisotel.InstrumentMetrics(redisClient); err != nil {
		slog.ErrorContext(ctx, "failed to instrument redis metrics",
			slog.String("event", "redis.otel.metrics.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Warn("failed to close redis client", slog.String("error", err.Error()))
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to connect redis",
			slog.String("event", "redis.connect.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx, authmodule.Repositories{
		Params:       authrepository.NewOIDCParamsRepository(redisClient),
		Sessions:     authrepository.NewSessionRepository(redisClient),
		Users:        authrepository.NewUserRepository(db),
		OIDCIdentity: authrepository.NewOIDCIdentityRepository(db),
		UserIdentity: authrepository.NewUserWithIdentityRepository(db),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize auth service",
			slog.String("event", "auth.init.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	mux.Handle(authPath, authHandler)

	taskCfg, err := taskconfig.Load()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load task config",
			slog.String("event", "config.load.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	remindQueue, cancelRemindQueue, taskQueueClient, err := taskmodule.NewRemindQueues(ctx, &taskCfg.TaskQueue)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create task remind queues",
			slog.String("event", "queue.init.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	taskRepos := taskmodule.Repositories{
		Tasks:               taskrepository.NewTaskRepository(db),
		TaskArchive:         taskrepository.NewTaskArchiveRepository(db),
		AuthClient:          authclient.NewAuthClient(taskCfg.AuthServiceURL),
		DeviceClient:        deviceclient.NewDeviceClient(taskCfg.DeviceServiceURL),
		RemindRegisterQueue: remindQueue,
		RemindCancelQueue:   cancelRemindQueue,
		TaskQueueClient:     taskQueueClient,
	}

	var closeTaskReposOnce sync.Once

	closeTaskRepos := func() {
		closeTaskReposOnce.Do(func() {
			if err := taskRepos.Close(); err != nil {
				slog.Warn("failed to close task repositories", slog.String("error", err.Error()))
			}
		})
	}

	defer closeTaskRepos()

	taskPath, taskHandler, err := taskmodule.NewHTTPHandlerWithRepositories(ctx, taskRepos)
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize task service",
			slog.String("event", "task.init.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	mux.Handle(taskPath, taskHandler)

	deviceCfg, err := deviceconfig.Load()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load device config",
			slog.String("event", "config.load.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	devicePath, deviceHandler, err := devicemodule.NewHTTPHandler(
		ctx,
		devicerepository.NewDeviceRepository(db),
		deviceCfg.AuthServiceURL,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize device service",
			slog.String("event", "device.init.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	mux.Handle(devicePath, deviceHandler)

	// Health check setup
	healthChecker := health.NewChecker(sqlDB, redisClient, Version)

	// gRPC Health Checking Protocol (grpc.health.v1.Health/Check)
	grpcHealthChecker := health.NewGRPCChecker(healthChecker)
	grpcHealthPath, grpcHealthHandler := grpchealth.NewHandler(grpcHealthChecker)
	mux.Handle(grpcHealthPath, grpcHealthHandler)

	// HTTP Health endpoints
	mux.HandleFunc("GET /health/live", healthChecker.LiveHandler)
	mux.HandleFunc("GET /health/ready", healthChecker.ReadyHandler)
	mux.HandleFunc("GET /health", healthChecker.ReadyHandler)

	// wrap by middleware
	handler := middleware.PanicRecoveryHTTP(mux)

	addr := ":8080"
	slog.InfoContext(ctx, "starting Connect API server",
		slog.String("event", "server.start"),
		slog.String("address", addr),
		slog.String("version", Version),
	)

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown connect api server", slog.String("error", err.Error()))
		}

		closeTaskRepos()
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "connect api server exited",
			slog.String("event", "server.exit.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	slog.InfoContext(ctx, "server stopped",
		slog.String("event", "server.stop"),
	)

	return nil
}
