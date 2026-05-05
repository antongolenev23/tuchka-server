package main

import (
	"github.com/antongolenev23/tuchka-server/internal/app"
	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/pkg/logger"

	_ "github.com/antongolenev23/tuchka-server/docs"
)

var version string

func main() {
	cfg := config.MustLoad()
	log := logger.MustInit(cfg.Env)

	app := app.New(cfg, log, version)
	app.Run()
}
