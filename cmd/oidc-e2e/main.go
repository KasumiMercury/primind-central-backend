package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/config"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	redirectURI := requireEnv("OIDC_GOOGLE_REDIRECT_URI")
	requireEnv("SESSION_SECRET")
	requireEnv("OIDC_GOOGLE_CLIENT_ID")
	requireEnv("OIDC_GOOGLE_CLIENT_SECRET")

	authSrv, cleanup, err := startAuthServer()
	if err != nil {
		return err
	}
	defer authSrv.Close()
	defer cleanup()

	client := authv1connect.NewAuthServiceClient(authSrv.Client(), authSrv.URL)

	stopCallback, callbackCh, err := startCallbackServer(redirectURI)
	if err != nil {
		return err
	}
	defer stopCallback()

	//exhaustruct:ignore
	paramsResp, err := client.OIDCParams(ctx, &authv1.OIDCParamsRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
	})
	if err != nil {
		return fmt.Errorf("get oidc params: %w", err)
	}

	log.Println("------------------------------------------------------------")
	log.Println(paramsResp.GetAuthorizationUrl())
	log.Println("------------------------------------------------------------")

	var callback oidcCallback
	select {
	case callback = <-callbackCh:
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for oauth callback: %w", ctx.Err())
	}

	if callback.code == "" || callback.state == "" {
		return fmt.Errorf("callback missing code/state: code=%q state=%q", callback.code, callback.state)
	}

	if callback.state != paramsResp.GetState() {
		return fmt.Errorf("state mismatch: got %s want %s", callback.state, paramsResp.GetState())
	}

	loginResp, err := client.OIDCLogin(ctx, &authv1.OIDCLoginRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
		Code:     callback.code,
		State:    callback.state,
	})
	if err != nil {
		return fmt.Errorf("oidc login failed: %w", err)
	}

	sessionToken := loginResp.GetSessionToken()
	if sessionToken == "" {
		return fmt.Errorf("empty session token returned")
	}

	log.Printf("Login succeeded. Session token length=%d", len(sessionToken))
	log.Println("Session token:")
	log.Println(sessionToken)

	return nil
}

func startAuthServer() (*httptest.Server, func(), error) {
	ctx := context.Background()
	mux := http.NewServeMux()

	appCfg, err := config.Load()
	if err != nil {
		return nil, func() {}, fmt.Errorf("load config: %w", err)
	}

	db, err := gorm.Open(postgres.Open(appCfg.Persistence.PostgresDSN), &gorm.Config{})
	if err != nil {
		return nil, func() {}, fmt.Errorf("connect postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, func() {}, fmt.Errorf("obtain postgres handle: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     appCfg.Persistence.RedisAddr,
		Password: appCfg.Persistence.RedisPassword,
		DB:       appCfg.Persistence.RedisDB,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close postgres connection: %v", err)
		}

		return nil, func() {}, fmt.Errorf("connect redis: %w", err)
	}

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx, authmodule.Repositories{
		Params:       repository.NewOIDCParamsRepository(redisClient),
		Sessions:     repository.NewSessionRepository(redisClient),
		Users:        repository.NewUserRepository(db),
		OIDCIdentity: repository.NewOIDCIdentityRepository(db),
		UserIdentity: repository.NewUserWithIdentityRepository(db),
	})
	if err != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
		}

		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close postgres connection: %v", err)
		}

		return nil, func() {}, fmt.Errorf("wire auth module: %w", err)
	}

	mux.Handle(authPath, authHandler)

	server := httptest.NewServer(mux)

	cleanup := func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
		}

		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close postgres connection: %v", err)
		}
	}

	return server, cleanup, nil
}

type oidcCallback struct {
	code  string
	state string
}

func startCallbackServer(redirectURI string) (func(), <-chan oidcCallback, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redirect uri %q: %w", redirectURI, err)
	}

	if u.Host == "" || u.Path == "" {
		return nil, nil, fmt.Errorf("redirect uri must include host, port, and path; got %q", redirectURI)
	}

	results := make(chan oidcCallback, 1)

	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("open listener on %s: %w", u.Host, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		select {
		case results <- oidcCallback{code: code, state: state}:
		default:
		}

		if _, err := fmt.Fprintln(w, "OIDC login callback received. You can close this tab and return to the CLI."); err != nil {
			log.Printf("failed to write callback response: %v", err)
		}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("callback server error: %v", err)
		}
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("callback server shutdown error: %v", err)
		}
	}

	return shutdown, results, nil
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("environment variable %s must be set for this CLI", key)
	}

	return val
}
