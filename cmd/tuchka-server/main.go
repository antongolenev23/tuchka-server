package main

import (
	"os"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/storage/disk"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler"
	"github.com/antongolenev23/tuchka-server/internal/http-server/router"
	"github.com/antongolenev23/tuchka-server/pkg/logger"
	"github.com/antongolenev23/tuchka-server/internal/repository/postgres"
	"github.com/antongolenev23/tuchka-server/internal/service"
)

func main() {
	cfg := config.MustLoad()
	logger := logger.MustInit(cfg.Env)

	repo, err := postgres.New(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to init repository", "error", err)
		os.Exit(1)
	}

	storage := disk.New(cfg)
	service := service.New(repo, storage, logger, cfg)
	handler := handler.New(service, logger, cfg)

	r := router.New(handler, logger)
	_ = r
}
