package deps

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/submission"
	"github.com/justblue/samsa/internal/infras/aws/s3"
	"github.com/justblue/samsa/internal/infras/cache"
	"github.com/justblue/samsa/internal/infras/email"
	"github.com/justblue/samsa/internal/infras/logger"
	"github.com/justblue/samsa/internal/infras/postgres"
	"github.com/justblue/samsa/internal/infras/redis"
	"github.com/justblue/samsa/internal/infras/telemetry"
	"github.com/justblue/samsa/internal/transport/http/middleware"

	httpTransport "github.com/justblue/samsa/internal/transport/http"
	workerTransport "github.com/justblue/samsa/internal/transport/worker"
	wsTransport "github.com/justblue/samsa/internal/transport/ws"
	redisLib "github.com/redis/go-redis/v9"
)

type AppDeps struct {
	Cfg    *config.Config
	Logger *slog.Logger

	Infra    InfraDeps
	Shared   SharedDeps
	Services ServiceDeps
}

type InfraDeps struct {
	DB        *pgxpool.Pool
	Redis     *redisLib.Client
	Cache     *cache.Client
	Worker    *workerTransport.Server
	Email     *email.Client
	Otlp      *telemetry.Telemetry
	S3        map[string]*s3.Client
	Hub       *wsTransport.Hub
	Publisher *wsTransport.Publisher
}

type SharedDeps struct {
	Validator *validator.Validate
	OAuth2    map[sqlc.OAuthProvider]*oauth2.Config
}

type ServiceDeps struct {
	Repositories *Repositories
	UseCases     *UseCases
}

func New(ctx context.Context, cfg *config.Config) (*AppDeps, error) {
	appDeps := &AppDeps{
		Cfg: cfg,
		Shared: SharedDeps{
			// Currently, only support oauth2 through google and github, so fixed sized is 2
			OAuth2: make(map[sqlc.OAuthProvider]*oauth2.Config, 2),
		},
		Infra: InfraDeps{
			// Currently, only 2 buckets support.
			S3: make(map[string]*s3.Client, 2),
		},
	}

	logger := slog.New(logger.NewTraceHandler(
		os.Stdout,
		&slog.HandlerOptions{},
	))
	appDeps.Logger = logger
	slog.SetDefault(logger)

	pool, err := postgres.New(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("initilize database failed: %w", err)
	}
	appDeps.Infra.DB = pool

	redisOpt := redis.NewRedisOpts(cfg)
	rdb, err := redis.New(ctx, redisOpt)
	if err != nil {
		return nil, fmt.Errorf("initilize redis failed: %w", err)
	}
	appDeps.Infra.Redis = rdb
	appDeps.Infra.Cache = cache.New(&cache.Options{
		Redis: rdb,
	})

	return appDeps, nil
}

func (d *AppDeps) Close() {
	if d.Infra.DB != nil {
		d.Infra.DB.Close()
	}
	if d.Infra.Redis != nil {
		d.Infra.Redis.Close()
	}
	if d.Infra.Worker != nil {
		d.Infra.Worker.Close()
	}
	if d.Infra.Otlp != nil {
		d.Infra.Otlp.Cancel()
	}
}

func (d *AppDeps) InitHTTP(version string) (*httpTransport.Server, error) {
	presenceStore := redis.NewPresenceStore(d.Infra.Redis)
	repos := NewRepositories(d.Infra.DB, d.Cfg)
	usecases := NewUseCases(d.Cfg, d.Infra.Cache, repos, presenceStore, d.Infra.Publisher)
	h := NewHTTPHandlers(usecases, d.Cfg, d.Shared.Validator)

	mw := &httpTransport.Middlewares{
		Otlp:        middleware.Otlp(d.Cfg.OpenTelemetry.Enable),
		RateLimit:   middleware.RateLimiter(d.Infra.Redis, nil),
		AuthSubject: middleware.AuthSubject(d.Cfg, repos.Session),
	}

	return httpTransport.New(d.Cfg, version, mw, h), nil
}

func (d *AppDeps) InitWorker() (*workerTransport.Server, error) {
	wServer, err := workerTransport.NewServer(d.Cfg)
	if err != nil {
		slog.Error("failed to initialize worker server", "error", err)
		return nil, err
	}

	// Initialize repositories needed for tasks
	repos := NewRepositories(d.Infra.DB, d.Cfg)

	// Register submission tasks
	submissionTaskHandler := submission.NewTaskHandler(
		repos.Submission,
		repos.SubmissionStatusHistory,
		repos.Notification,
	)
	wServer.Register(submissionTaskHandler.RegisterTasks()...)

	return wServer, nil
}

func (d *AppDeps) InitWS() (*wsTransport.Server, error) {
	wsRegistry := wsTransport.NewRegistry()
	wsHub := wsTransport.NewHub(wsRegistry, d.Infra.Redis)
	wsPublisher := wsTransport.NewPublisher(wsHub)

	d.Infra.Hub = wsHub
	d.Infra.Publisher = wsPublisher

	presenceStore := redis.NewPresenceStore(d.Infra.Redis)
	typingStore := redis.NewTypingStore(d.Infra.Redis)

	repos := NewRepositories(d.Infra.DB, d.Cfg)
	usecases := NewUseCases(d.Cfg, d.Infra.Cache, repos, presenceStore, d.Infra.Publisher)

	wsHandlers := NewWSHandlers(presenceStore, typingStore, wsHub, usecases)
	wsHandlers.Register(wsRegistry)

	return wsTransport.New(d.Cfg, wsHub, wsRegistry), nil
}
