package router

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler"
	mw "github.com/antongolenev23/tuchka-server/internal/http-server/middleware"
)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token in format: Bearer <token>

func New(handler *handler.Handler, cfg *config.Config, log *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(mw.LoggerMiddleware(log))
	r.Use(chimw.Recoverer)

	r.Post("/auth/register", handler.Register())
	r.Post("/auth/login", handler.Login())

	r.Route("/files", func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg, log))

		r.Post("/upload", handler.Upload())
		r.Get("/", handler.Files())
		r.Post("/download", handler.Download())
		r.Post("/delete", handler.Delete())
	})

	
	if cfg.Env == config.EnvDev {
    	r.Get("/swagger/*", httpSwagger.WrapHandler)
	}

	return r
}
