package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	connect "connectrpc.com/connect"
	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
	sessioncfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	sessionjwt "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/config"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
	taskv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1"
	taskv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	taskmodule "github.com/KasumiMercury/primind-central-backend/internal/task"
	taskrepository "github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	authHeaderName = "Authorization"
	bearerPrefix   = "Bearer "
)

func main() {
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	db, err := gorm.Open(postgres.Open(appCfg.Persistence.PostgresDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("obtain postgres handle: %w", err)
	}

	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close postgres connection: %v", err)
		}
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     appCfg.Persistence.RedisAddr,
		Password: appCfg.Persistence.RedisPassword,
		DB:       appCfg.Persistence.RedisDB,
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}

	authServer, authRepos, err := startAuthServer(ctx, db, redisClient)
	if err != nil {
		return err
	}
	defer authServer.Close()

	taskServer, err := startTaskServer(ctx, db, authServer.URL)
	if err != nil {
		return err
	}
	defer taskServer.Close()

	sessionToken, userID, err := issueSession(ctx, authRepos)
	if err != nil {
		return err
	}

	log.Printf("issued session for user %s", userID)

	authClient := authv1connect.NewAuthServiceClient(authServer.Client(), authServer.URL)
	if err := verifySession(ctx, authClient, sessionToken, userID); err != nil {
		return err
	}

	taskClient := taskv1connect.NewTaskServiceClient(taskServer.Client(), taskServer.URL)

	if err := assertInterceptorBlocksMissingToken(ctx, taskClient); err != nil {
		return err
	}

	taskClientWithAuth := taskv1connect.NewTaskServiceClient(
		taskServer.Client(),
		taskServer.URL,
		connect.WithInterceptors(sessionHeaderInterceptor(sessionToken)),
	)

	createResp, err := taskClientWithAuth.CreateTask(ctx, buildCreateTaskRequest("Task interceptor e2e"))
	if err != nil {
		return fmt.Errorf("create task via connectrpc: %w", err)
	}

	log.Printf("task created with id %s", createResp.GetTask().GetTaskId())

	getResp, err := taskClientWithAuth.GetTask(ctx, &taskv1.GetTaskRequest{
		TaskId: createResp.GetTask().GetTaskId(),
	})
	if err != nil {
		return fmt.Errorf("get task via connectrpc: %w", err)
	}

	log.Println("task retrieved successfully:")
	log.Printf("  title: %s", getResp.GetTask().GetTitle())
	log.Printf("  type: %s", getResp.GetTask().GetTaskType().String())
	log.Printf("  status: %s", getResp.GetTask().GetTaskStatus().String())
	log.Printf("  description: %s", getResp.GetTask().GetDescription())

	return nil
}

func startAuthServer(
	ctx context.Context,
	db *gorm.DB,
	redisClient *redis.Client,
) (*httptest.Server, authRepositories, error) {
	mux := http.NewServeMux()

	paramsRepo := repository.NewOIDCParamsRepository(redisClient)
	sessionRepo := repository.NewSessionRepository(redisClient)
	userRepo := repository.NewUserRepository(db)
	oidcIdentityRepo := repository.NewOIDCIdentityRepository(db)
	userIdentityRepo := repository.NewUserWithIdentityRepository(db)

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx, authmodule.Repositories{
		Params:       paramsRepo,
		Sessions:     sessionRepo,
		Users:        userRepo,
		OIDCIdentity: oidcIdentityRepo,
		UserIdentity: userIdentityRepo,
	})
	if err != nil {
		return nil, authRepositories{}, fmt.Errorf("initialize auth handler: %w", err)
	}

	mux.Handle(authPath, authHandler)

	server := httptest.NewServer(mux)

	return server, authRepositories{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}, nil
}

func startTaskServer(
	ctx context.Context,
	db *gorm.DB,
	authServiceURL string,
) (*httptest.Server, error) {
	mux := http.NewServeMux()

	taskPath, taskHandler, err := taskmodule.NewHTTPHandler(
		ctx,
		taskrepository.NewTaskRepository(db),
		authServiceURL,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize task handler: %w", err)
	}

	mux.Handle(taskPath, taskHandler)

	return httptest.NewServer(mux), nil
}

type authRepositories struct {
	sessionRepo domainsession.SessionRepository
	userRepo    domainuser.UserRepository
}

func issueSession(ctx context.Context, repos authRepositories) (string, string, error) {
	sessionCfg, err := sessioncfg.Load()
	if err != nil {
		return "", "", fmt.Errorf("load session config: %w", err)
	}

	u, err := domainuser.CreateUserWithRandomColor()
	if err != nil {
		return "", "", fmt.Errorf("create user: %w", err)
	}

	if err := repos.userRepo.SaveUser(ctx, u); err != nil {
		return "", "", fmt.Errorf("persist user: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(sessionCfg.Duration)

	session, err := domainsession.NewSession(u.ID(), now, expiresAt)
	if err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}

	if err := repos.sessionRepo.SaveSession(ctx, session); err != nil {
		return "", "", fmt.Errorf("persist session: %w", err)
	}

	token, err := sessionjwt.NewSessionJWTGenerator(sessionCfg).Generate(session, u)
	if err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}

	return token, u.ID().String(), nil
}

func verifySession(
	ctx context.Context,
	client authv1connect.AuthServiceClient,
	token string,
	expectedUserID string,
) error {
	resp, err := client.ValidateSession(ctx, &authv1.ValidateSessionRequest{
		SessionToken: token,
	})
	if err != nil {
		return fmt.Errorf("validate issued session over connectrpc: %w", err)
	}

	if resp.GetUserId() != expectedUserID {
		return fmt.Errorf("validate session mismatch: got %s want %s", resp.GetUserId(), expectedUserID)
	}

	log.Println("session validation via auth service succeeded")

	return nil
}

func assertInterceptorBlocksMissingToken(
	ctx context.Context,
	client taskv1connect.TaskServiceClient,
) error {
	_, err := client.CreateTask(ctx, buildCreateTaskRequest("unauthorized request"))
	if err == nil {
		return errors.New("expected unauthenticated error without session token")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeUnauthenticated {
		return fmt.Errorf("expected unauthenticated error, got: %w", err)
	}

	log.Println("interceptor rejected request without session token as expected")

	return nil
}

func sessionHeaderInterceptor(sessionToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient && sessionToken != "" {
				req.Header().Set(authHeaderName, bearerPrefix+sessionToken)
			}

			return next(ctx, req)
		}
	}
}

func buildCreateTaskRequest(title string) *taskv1.CreateTaskRequest {
	description := fmt.Sprintf("generated at %s", time.Now().Format(time.RFC3339))

	//exhaustruct:ignore
	return &taskv1.CreateTaskRequest{
		Title:       title,
		TaskType:    taskv1.TaskType_TASK_TYPE_NORMAL,
		Description: description,
	}
}
