package bootstrap

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/justblue/samsa/bootstrap/deps"
	"github.com/justblue/samsa/config"
	transportHTTP "github.com/justblue/samsa/internal/transport/http"
	transportWorker "github.com/justblue/samsa/internal/transport/worker"
	transportWS "github.com/justblue/samsa/internal/transport/ws"

	_ "github.com/justblue/samsa/gen/swagger"
)

type Server struct {
	Version string

	Deps *deps.AppDeps

	HTTP   *transportHTTP.Server
	Worker *transportWorker.Server
	WS     *transportWS.Server
}

func Init(version string, cfg *config.Config) (*Server, error) {
	ctx := context.Background()
	appDeps, err := deps.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	app := &Server{Version: version, Deps: appDeps}

	httpServer, err := appDeps.InitHTTP(version)
	if err != nil {
		slog.Error("failed to initialize HTTP server", "error", err)
		return nil, err
	}
	app.HTTP = httpServer

	workerServer, err := appDeps.InitWorker()
	if err != nil {
		slog.Error("failed to initialize Worker server", "error", err)
		return nil, err
	}
	app.Worker = workerServer

	wsServer, err := appDeps.InitWS()
	if err != nil {
		slog.Error("failed to initialize WS server", "error", err)
		return nil, err
	}
	app.WS = wsServer

	return app, nil
}

func (a *Server) InitGRPC() *Server {
	return a
}

// Run starts the application and waits for termination signals to gracefully shut down.
func Run(app *Server) error {
	defer app.Deps.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return app.HTTP.Start() })
	g.Go(func() error { return app.Worker.Start() })
	g.Go(func() error { return app.WS.Start(ctx) })

	g.Go(func() error {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		app.HTTP.Shutdown(shutCtx)
		app.Worker.Close()
		app.WS.Stop()
		return nil
	})

	return g.Wait()
}
