package main

import (
	"os"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/logger"
	"github.com/antongolenev23/tuchka-server/internal/storage/postgres"
)

func main() {
	cfg := config.MustLoad()
	logger := logger.MustInit(cfg.Env)

	storage, err := postgres.New(cfg.DatabaseDSN)
	if err != nil{
		logger.Error("failed to init database", "error", err)
		os.Exit(1)
	}

	_ = storage
}
