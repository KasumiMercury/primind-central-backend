package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	mux := http.NewServeMux()

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx)
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
