package config

import (
	"net/url"
	"time"

	"github.com/joho/godotenv"
	"github.com/justblue0312/envx"
)

const (
	DevelopmentMode string = "development"
	ProductionMode  string = "production"
	TestingMode     string = "testing"
)

var (
	EmptyPath string = "/"

	APIPaginationMaxLimit     int32 = 100
	APIPaginationDefaultLimit int32 = 25
	APIPaginationDefaultPage  int32 = 1
)

// Config struct holds all the configuration for the application.
type Config struct {
	SecretKey string `required:"true" envx:"SECRET_KEY"`
	Mode      string `required:"true" default:"development"`
	Host      string `required:"true" default:"localhost"`
	HTTPPort  int    `required:"true" default:"8000" envx:"HTTP_PORT"`
	GRPCPort  int    `required:"true" default:"8001" envx:"GRPC_PORT"`
	WSPort    int    `required:"true" default:"8002" envx:"WS_PORT"`
	Location  string `default:"UTC"`

	CorsOrigin   []string `required:"true" envx:"CORS_ORIGINS" split_words:"true" default:"http://localhost:3000"`
	AllowedHosts []string `required:"true" envx:"ALLOWED_HOSTS" split_words:"true" default:"http://localhost:3000"`
	FrontEndURL  string   `required:"true" envx:"FRONTEND_URL" default:"http://localhost:3000"`

	IgnoredPaths []string `required:"true" envx:"IGNORED_PATHS" split_words:"true" default:"/health,/metrics"`

	Cache struct {
		EnableCache   bool          `default:"true" envx:"ENABLE_CACHE"`
		QueryCacheTTL time.Duration `default:"300s" envx:"QUERY_CACHE_TTL"`
	}

	EnableRunMigrationAtStartup bool `default:"true" envx:"ENABLE_RUN_MIGRATION_AT_STARTUP"`
	EnableRequestLogging        bool `default:"true" envx:"ENABLE_REQUEST_LOGGING"`

	UserSessionTTL         time.Duration `default:"2678400s"` // 31 days in seconds
	UserSessionCookieName  string        `default:"samsa_session"`
	UserSessionTokenPrefix string        `default:"samsa_us_"`
	UserSessionDomain      string        `default:"127.0.0.1"`

	OpenTelemetry struct {
		Enable         bool    `required:"true" default:"true"`
		Endpoint       string  `required:"true" default:"0.0.0.0:4317"`
		ServiceName    string  `required:"true" default:"samsa" envx:"SERVICE_NAME"`
		ServiceVersion string  `required:"true" default:"0.1.0" envx:"SERVICE_VERSION"`
		MeterName      string  `required:"true" default:"samsa-meter"`
		SamplerRatio   float64 `required:"true" default:"0.1"`
	}

	Postgres struct {
		Host            string        `required:"true" default:"127.0.0.1"`
		Port            int           `required:"true" default:"5432"`
		User            string        `required:"true"`
		Pwd             string        `required:"true"`
		Database        string        `required:"true"`
		TestDatabase    string        `envx:"TEST_DATABASE" default:"samsa_test"`
		SslMode         string        `default:"disable"`
		MaxConns        int32         `required:"true" default:"4"`
		MaxConnIdleTime time.Duration `required:"true" default:"10m"`
		MaxConnLifetime time.Duration `required:"true" default:"300s"`
	}

	Redis struct {
		Enable bool   `default:"true"`
		Host   string `required:"true" default:"127.0.0.1"`
		Port   string `required:"true" default:"6379"`
		DB     int    `required:"true" default:"1"`
		User   string
		Pwd    string
	}

	Jwt struct {
		Issuer          string        `default:"samsa"`
		AccessTokenTTL  time.Duration `default:"1h"`
		RefreshTokenTTL time.Duration `default:"72h"`
	}

	Session struct {
		Name     string        `default:"session"`
		Path     string        `default:"/"`
		Domain   string        `default:"127.0.0.1"`
		Duration time.Duration `default:"720h"`
		HTTPOnly bool          `default:"true"`
		Secure   bool          `default:"false"`
	}

	Email struct {
		Host      string `required:"true" default:"localhost"`
		Port      int    `required:"true" default:"1025"`
		Username  string `required:"true" default:""`
		Pwd       string `required:"true" default:""`
		FromEmail string `required:"true" default:"no-reply@samsa.com"`
		FromName  string `default:"samsa"`
		UseTLS    bool   `default:"false" required:"true"`
		UseSSL    bool   `default:"false" required:"true"`

		VerifyEmailTokenTTL time.Duration `default:"10m"`
	}

	OAuth2 struct {
		GoogleClientID     string `required:"true" envx:"GOOGLE_CLIENT_ID"`
		GoogleClientSecret string `required:"true" envx:"GOOGLE_CLIENT_SECRET"`
		GoogleRedirectURL  string `required:"true" envx:"GOOGLE_REDIRECT_URL" default:"http://localhost:8000/v1/auth/callback/google"`
		GithubClientID     string `required:"true" envx:"GITHUB_CLIENT_ID"`
		GithubClientSecret string `required:"true" envx:"GITHUB_CLIENT_SECRET"`
		GithubRedirectURL  string `required:"true" envx:"GITHUB_REDIRECT_URL" default:"http://localhost:8000/v1/auth/callback/github"`

		OAuthStateTTL       time.Duration `default:"10m"`
		OAuthStateCookieKey string        `default:"samsa_oauth_state"`
	}

	AWS struct {
		AccessKeyID     string        `required:"true" envx:"ACCESS_KEY_ID"`
		SecretAccessKey string        `required:"true" envx:"SECRET_ACCESS_KEY"`
		Region          string        `default:"us-east-1"`
		S3Buckets       []string      `required:"true" envx:"S3_BUCKETS"`
		S3EndpointURL   string        `required:"true" envx:"S3_ENDPOINT_URL"`
		PresignTTL      time.Duration `default:"15m"`
	}

	File struct {
		PayloadSizeThreshold int64 `default:"1048576" envx:"FILE_PAYLOAD_SIZE_THRESHOLD"` // 1MB default
	}
}

func New() (*Config, error) {
	var c Config

	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	envx.MustProcess("SAMSA", &c)
	return &c, nil
}

// GetReturnURL constructs a full URL by combining the FrontEndURL with the provided path.
func (c *Config) GetReturnURL(path string) string {
	base, err := url.Parse(c.FrontEndURL)
	if err != nil {
		return ""
	}

	if path == "" {
		path = EmptyPath
	}

	ref, err := url.Parse(path)
	if err != nil || ref.IsAbs() {
		return base.String()
	}

	return base.ResolveReference(ref).String()
}

func (c *Config) IsProduction() bool {
	return c.Mode == ProductionMode
}
