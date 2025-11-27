package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
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

	authSrv, err := startAuthServer()
	if err != nil {
		return err
	}
	defer authSrv.Close()

	client := authv1connect.NewAuthServiceClient(authSrv.Client(), authSrv.URL)

	stopCallback, callbackCh, err := startCallbackServer(redirectURI)
	if err != nil {
		return err
	}
	defer stopCallback()

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

func startAuthServer() (*httptest.Server, error) {
	ctx := context.Background()
	mux := http.NewServeMux()

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("wire auth module: %w", err)
	}

	mux.Handle(authPath, authHandler)

	server := httptest.NewServer(mux)
	return server, nil
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

		fmt.Fprintln(w, "OIDC login callback received. You can close this tab and return to the CLI.")
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
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
