package router

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/antongolenev23/tuchka-server/internal/http-server/handler"
	mw "github.com/antongolenev23/tuchka-server/internal/http-server/middleware"
)

func New(handler *handler.Handler, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(mw.New(logger))
	r.Use(chimw.Recoverer)

	return r
}
