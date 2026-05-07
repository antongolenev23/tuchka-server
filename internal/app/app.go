package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler"
	"github.com/antongolenev23/tuchka-server/internal/http-server/router"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/repository/postgres"
	"github.com/antongolenev23/tuchka-server/internal/service"
	"github.com/antongolenev23/tuchka-server/internal/storage/disk"
)

type App struct {
	cfg     *config.Config
	log     *slog.Logger
	version string

	server *http.Server
	repo   repository.Repository
}

func New(cfg *config.Config, log *slog.Logger, version string) *App {
	a := &App{
		cfg:     cfg,
		log:     log,
		version: version,
	}

	a.bootstrap()
	return a
}

func (a *App) Run() {
	a.runServer()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-stop
	a.log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	a.shutdown(ctx)
}

func (a *App) bootstrap() {
	repo, err := postgres.New(a.cfg)
	if err != nil {
		a.log.Error("failed to init repo", "error", err)
		os.Exit(1)
	}

	storage := disk.New(a.cfg)
	svc := service.New(repo, storage, a.cfg)
	h := handler.New(svc, a.cfg, a.log)

	r := router.New(h, a.cfg, a.log)

	a.repo = repo

	a.server = &http.Server{
		Addr:         a.cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  a.cfg.HTTPServer.RequestReadTimeout,
		WriteTimeout: a.cfg.HTTPServer.ResponseWriteTimeout,
		IdleTimeout:  a.cfg.HTTPServer.IdleTimeout,
	}
}

func (a *App) runServer() {
	go func() {
		a.log.Info("starting server",
			slog.String("address", a.cfg.HTTPServer.Address),
		)

		if err := a.server.ListenAndServeTLS(
			a.cfg.HTTPServer.CertFile,
			a.cfg.HTTPServer.KeyFile,
		); err != nil && err != http.ErrServerClosed {
			a.log.Error("server stopped", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	var err error

	if err = a.server.Shutdown(ctx); err != nil {
		a.log.Error("shutdown error", slog.String("error", err.Error()))
	} else {
		a.log.Info("http server stopped")
	}

	done := make(chan error, 1)

	go func() {
		done <- a.repo.Close()
	}()

	select {
	case err = <-done:
		if err != nil {
			a.log.Error("repo close error", slog.String("error", err.Error()))
		} else {
			a.log.Info("repo connection pool closed")
		}
	case <-ctx.Done():
		a.log.Warn("repo close timeout")
	}

	if err != nil {
		a.log.Warn("uncorrect shutdown")
	} else {
		a.log.Info("graceful shutdown completed")
	}
}
