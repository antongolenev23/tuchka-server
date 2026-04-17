package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/service"
	resp "github.com/antongolenev23/tuchka-server/pkg/api/response"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const(
	invalidRequestBody = "invalid request body"
	couldNotParseFile = "could no parse file"
	notAllFilesFound = "not all files found"
)

var(
	ErrNoFilesFound = errors.New("request body contains no files")
)

type ErrTooManyFiles struct{
	MaxFiles int
}

func(e ErrTooManyFiles) Error() string {
	return fmt.Sprintf("too many files. maximum %d", e.MaxFiles)
}

type Handler struct {
	service service.Service
	logger  *slog.Logger
	cfg     *config.Config
}

func New(service service.Service, logger *slog.Logger, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
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

		fileHeaders, err := validateAndParseBody(w, r)
		if err != nil {
			log.Info("failed to parse body",
				slog.String("error", err.Error()),
			)

			render.Status(r, http.StatusBadRequest)
			if errors.Is(err, ErrNoFilesFound) {
				render.JSON(w, r, resp.Error(err.Error()))
			} else {
				render.JSON(w, r, resp.Error(invalidRequestBody))
			}
			return
		}

		log.Info("request body decoded",
			slog.Int("files_count", len(fileHeaders)),
		)

		var result file.Result

		files := getFileEntities(fileHeaders, &result, log)
		defer closeFiles(files, log)

		h.service.Upload(files, &result, log)
		if len(result.Errors) > 0 {
			render.Status(r, http.StatusMultiStatus)
		}

		log.Info("files uploaded",
			slog.Any("result", result),
		)

		response := dto.GetResultDTO(result)
		render.JSON(w, r, response)
	})
}

func validateAndParseBody(w http.ResponseWriter, r *http.Request) ([]*multipart.FileHeader, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024*1024) // 32 MB
	defer r.Body.Close()

	if err := r.ParseMultipartForm(32 * 1024 * 1024); err != nil {
		return nil, err
	}

	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		return nil, ErrNoFilesFound
	}

	return fileHeaders, nil
}

func getFileEntities(files []*multipart.FileHeader, result *file.Result, log *slog.Logger) []file.File {
	fileEntities := make([]file.File, 0, len(files))

	for _, fh := range files {
		mf, err := fh.Open()
		if err != nil {
			log.Error("can not open fileheader of %s: %s", fh.Filename, err)
			result.AddError(fh.Filename, couldNotParseFile)
			continue
		}

		fileEntities = append(fileEntities, file.File{
			Name: fh.Filename,
			Size: fh.Size,
			Data: mf,
		})
	}

	return fileEntities
}

func closeFiles(files []file.File, log *slog.Logger) {
	for _, f := range files {
		if f.Data != nil {
			if err := f.Data.Close(); err != nil {
				log.Warn("failed to close file",
					slog.String("filename", f.Name),
					slog.String("error", err.Error()),
				)
			} else {
				log.Debug("file closed",
					slog.String("filename", f.Name),
				)
			}
		}
	}
}

func (h *Handler) Files() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Files" 

		log := h.logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		metadata, err := h.service.GetSavedFilesInfo()
		if err != nil{
			log.Error("failed to get saved files info",
				slog.String("error", err.Error()),
			)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to get saved files info"))
			return
		}

		log.Info("get saved files info",
			slog.Any("files_info", metadata),
		)

		render.JSON(w, r, metadata)
	})
}

func (h *Handler) Download() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Download"

		log := h.logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024)

		var req dto.FilesList
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Info("failed to parse body", slog.Any("error", err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error(invalidRequestBody))
			return
		}

		if err := h.validateDownloadRequest(&req); err != nil {
			log.Info("validation failed", slog.Any("error", err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

		err := h.service.Download(req, w)
		if err != nil {
			if errors.Is(err, service.ErrNotAllFilesExist) {
				log.Info(notAllFilesFound)
				http.Error(w, notAllFilesFound, http.StatusBadRequest)
			} else {
				log.Error("download failed", slog.Any("error", err))
				http.Error(w, "download failed", http.StatusInternalServerError)
			}
		}
	}
}

func (h *Handler) validateDownloadRequest(req *dto.FilesList) error {
    if len(req.Files) == 0 {
        return ErrNoFilesFound
    }

	maxDownload := h.cfg.Files.MaxDownload
    
    if len(req.Files) > maxDownload {
        return ErrTooManyFiles{MaxFiles: maxDownload}
    }

	return nil
}


func (h *Handler) Delete() http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        const op = "handler.Delete"

        log := h.logger.With(
            slog.String("op", op),
            slog.String("request_id", middleware.GetReqID(r.Context())),
        )

        r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1 MB

        var req dto.FilesList
        if err := render.DecodeJSON(r.Body, &req); err != nil {
            log.Info("failed to decode JSON",
                slog.String("error", err.Error()),
            )
            render.Status(r, http.StatusBadRequest)
            render.JSON(w, r, resp.Error(invalidRequestBody))
            return
        }

        if err := h.validateDeleteRequest(&req); err != nil {
            log.Info("validation failed",
                slog.String("error", err.Error()),
            )
            render.Status(r, http.StatusBadRequest)
            render.JSON(w, r, resp.Error(err.Error()))
            return
        }

        log.Info("request decoded",
            slog.Int("files_count", len(req.Files)),
        )

        result := h.service.DeleteFiles(req, log)

        if len(result.Errors) > 0 {
            render.Status(r, http.StatusMultiStatus)
        } else {
            render.Status(r, http.StatusOK)
        }

        log.Info("files deleted",
            slog.Int("success_count", len(result.Success)),
            slog.Int("error_count", len(result.Errors)),
        )

        render.JSON(w, r, dto.GetResultDTO(result))
    })
}

func (h *Handler) validateDeleteRequest(req *dto.FilesList) error {
    if len(req.Files) == 0 {
        return ErrNoFilesFound
    }

	maxDelete := h.cfg.Files.MaxDelete

    if len(req.Files) > maxDelete {
        return ErrTooManyFiles{MaxFiles: maxDelete}
    }

    return nil
}