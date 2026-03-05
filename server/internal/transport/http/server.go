package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/feature/chapter"
	"github.com/justblue/samsa/internal/feature/document"
	document_folder "github.com/justblue/samsa/internal/feature/document_folder"
	"github.com/justblue/samsa/internal/feature/flag"
	"github.com/justblue/samsa/internal/feature/genre"
	"github.com/justblue/samsa/internal/feature/spinnet"
	"github.com/justblue/samsa/internal/feature/story"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/internal/transport/http/middleware"
)

type Server struct {
	version string
	server  *http.Server
	cfg     *config.Config
	router  *chi.Mux
}

// HTTPHandlers holds all HTTP handlers for features.
// Add new feature handlers here; Register wires them to the router.
type HTTPHandlers struct {
	User           *user.HTTPHandler
	Flag           *flag.HTTPHandler
	Spinnet        *spinnet.HTTPHandler
	Story          *story.HTTPHandler
	Genre          *genre.HTTPHandler
	Chapter        *chapter.HTTPHandler
	Document       *document.HTTPHandler
	DocumentFolder *document_folder.HTTPHandler
}

// Register mounts all feature routes onto the provided router.
func (h *HTTPHandlers) Register(version string, r chi.Router) {
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"version": "%s"}`, version)
	})

	r.Route("/api/v1", func(r chi.Router) {
		user.RegisterHTTPEndpoints(r, h.User)
		flag.RegisterHTTPEndpoints(r, h.Flag)
		spinnet.RegisterHTTPEndpoints(r, h.Spinnet)
		story.RegisterStoryRoutes(r, h.Story)
		genre.RegisterGenreRoutes(r, h.Genre)
		chapter.RegisterChapterRoutes(r, h.Chapter)
		document.RegisterDocumentRoutes(r, h.Document)
		document_folder.RegisterDocumentFolderRoutes(r, h.DocumentFolder)
	})
}

// Middlewares groups all pre-constructed middlewares
// They are built in bootstrap/wire.go with their deps injected
// This struct receives them as ready-to-use http.Handler wrappers
type Middlewares struct {
	Otlp        func(http.Handler) http.Handler
	AuthSubject func(http.Handler) http.Handler
	RateLimit   func(http.Handler) http.Handler // nil if not production
}

func New(cfg *config.Config, version string, mw *Middlewares, h *HTTPHandlers) *Server {
	srv := &Server{
		version: version,
		cfg:     cfg,
		router:  chi.NewRouter(),
	}

	srv.setupMiddleware(mw)
	h.Register(srv.version, srv.router)

	srv.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.HTTPPort),
		Handler:           srv.router,
		ReadHeaderTimeout: 60 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return srv
}

func (s *Server) setupMiddleware(mw *Middlewares) {
	s.router.Use(middleware.Recovery)
	s.router.Use(chiMiddleware.RealIP)

	// OTLP tracing
	if mw.Otlp != nil {
		s.router.Use(mw.Otlp)
	}

	// Request logging
	if s.cfg.EnableRequestLogging {
		s.router.Use(chiMiddleware.Logger)
	}

	// Rate limiting — only set in production, bootstrap passes nil otherwise
	if mw.RateLimit != nil {
		s.router.Use(mw.RateLimit)
	}

	s.router.Use(cors.New(s.buildCORSOptions()).Handler)

	// Auth subject — constructed in bootstrap with its deps, arrives as middleware
	if mw.AuthSubject != nil {
		s.router.Use(mw.AuthSubject)
	}

	// Custom 404
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "endpoint not found"}`))
	})
}

func (s *Server) buildCORSOptions() cors.Options {
	if !s.cfg.IsProduction() {
		return cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"*"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           300,
		}
	}
	return cors.Options{
		AllowedOrigins:   s.cfg.CorsOrigin,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}
}

// Start is blocking — call inside a goroutine or errgroup
func (s *Server) Start() error {
	slog.Info("http server starting", "addr", s.server.Addr)
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil // expected on clean shutdown
	}
	return err
}

// Shutdown gracefully stops the HTTP server and cleans up transport-owned resources.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("http server shutting down")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
