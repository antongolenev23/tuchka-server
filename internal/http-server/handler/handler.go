package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/service"
	resp "github.com/antongolenev23/tuchka-server/pkg/api/response"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Handler struct {
	service service.Service
	logger *slog.Logger
	cfg *config.Config
}

func New(service service.Service, logger *slog.Logger, cfg *config.Config) *Handler{
	return &Handler{
		service: service,
		logger: logger,
		cfg: cfg,
	}
}

func (h *Handler) Upload() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Upload"

		log := h.logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, 25 * 1024 * 1024)
		defer r.Body.Close()

		if err := r.ParseMultipartForm(25 * 1024 * 1024); err != nil {
			log.Error(fmt.Sprintf("failed to parse multipart data: %s", err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid body"))
			return
		}

		files := r.MultipartForm.File["files"]
		if len(files) == 0{
			log.Error("no files uploaded")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("no files uploaded"))
			return
		}

		domainFiles := make([]file.UploadFile, 0, len(files))

		for _, fh := range files {
			mf, err := fh.Open()
			if err != nil{
				continue
			}

			domainFiles = append(domainFiles, file.UploadFile{
				Name: fh.Filename,
				Size: fh.Size,
				Data: mf,
			})
		}

		defer func() {
			for _, f := range domainFiles {
				if f.Data != nil {
					f.Data.Close()
				}
			}
		}()

		result := h.service.UploadFiles(domainFiles)
		if len(result.Errors) > 0{
			render.Status(r, http.StatusMultiStatus)
		} else  {
			render.Status(r, http.StatusOK)
		}

		responce := dto.GetResultDTO(result)
		render.JSON(w, r, responce)
	})
}