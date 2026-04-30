package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler"
	"github.com/antongolenev23/tuchka-server/internal/http-server/router"
	"github.com/antongolenev23/tuchka-server/internal/repository/postgres"
	"github.com/antongolenev23/tuchka-server/internal/service"
	"github.com/antongolenev23/tuchka-server/internal/storage/disk"
	"github.com/antongolenev23/tuchka-server/pkg/logger"

	_ "github.com/antongolenev23/tuchka-server/docs"
)

var(
	version string
)

// @title Tuchka Server API
// @version 0.0.4
// @description API для загрузки и управления файлами
// @termsOfService http://swagger.io/terms/

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.MustLoad()
	log := logger.MustInit(cfg.Env)

	log.Info("starting tuchka-server", 
		slog.String("env", cfg.Env),
		slog.String("version", version),
	)
	log.Debug("debug messages are enabled")

	repo, err := postgres.New(cfg)
	if err != nil {
		log.Error("failed to init repository", "error", err)
		os.Exit(1)
	}

	storage := disk.New(cfg)
	service := service.New(repo, storage, cfg)
	handler := handler.New(service, cfg, log)

	r := router.New(handler, cfg, log)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.RequestReadTimeout,
		WriteTimeout: cfg.HTTPServer.ResponseWriteTimeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server",
			slog.String("error", err.Error()),
		)
	}

	log.Error("server stopped")
}
