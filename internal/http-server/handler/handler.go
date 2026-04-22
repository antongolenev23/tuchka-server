package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/service"
	resp "github.com/antongolenev23/tuchka-server/pkg/api/response"
	mw "github.com/antongolenev23/tuchka-server/internal/http-server/middleware"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

const(
	invalidRequestBody = "invalid request body"
	couldNotParseFile = "could no parse file"
	notAllFilesFound = "not all files found"
	userAlreadyExists = "user already exists"
	userNotExists = "user not exists"
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
	log  *slog.Logger
	cfg     *config.Config
}

func New(service service.Service, cfg *config.Config, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		log:  log,
		cfg: cfg,
	}
}

func (h *Handler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Register"

		log := h.log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, 4 * 1024) // 4 KB
		defer r.Body.Close()


		var req dto.AuthRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Info("failed to parse body",
				slog.String("error", err.Error()),
			)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error(invalidRequestBody))
			return
		}

		token, user, err := h.service.Register(req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserAlreadyExists) {
				log.Info("can not register user",
					slog.String("error", userAlreadyExists),
				)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Error(userAlreadyExists))
			} else {
				log.Error("can not register user",
					slog.String("error", err.Error()),
				)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("failed to register"))
			}
			return
		}

		log.Info("register completed",
			slog.String("user_id", user.ID.String()),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, dto.AuthResponse{Email: user.Email, Token: token})
	}
}

func (h *Handler) Login() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Login"

		log := h.log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, 4 * 1024) // 4 KB
		defer r.Body.Close()

		var req dto.AuthRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Info("failed to parse body",
				slog.String("error", err.Error()),
			)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error(invalidRequestBody))
			return
		}

		token, user, err := h.service.Login(req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserNotExists) {
				log.Info("can not login",
					slog.String("error", userNotExists),
				)
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error(userNotExists))
			} else {
				log.Error("can not login",
					slog.String("error", err.Error()),
				)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("failed to login"))
			}
			return
		}

		log.Info("login completed",
			slog.String("user_id", user.ID.String()),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, dto.AuthResponse{Email: user.Email, Token: token})
	})
}

func (h *Handler) Upload() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {		
		const op = "handler.Upload"

		log := h.log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, ok := getUserID(r)
		if !ok {
			log.Error("failed to get user id from request context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to upload files"))
			return
		}

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

		var result entity.OperationResult

		files := getFileEntities(fileHeaders, &result, log)
		defer closeFiles(files, log)

		h.service.Upload(files, &result, userID, log)
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

func getFileEntities(files []*multipart.FileHeader, result *entity.OperationResult, log *slog.Logger) []entity.File {
	fileEntities := make([]entity.File, 0, len(files))

	for _, fh := range files {
		mf, err := fh.Open()
		if err != nil {
			log.Error("can not open fileheader of %s: %s", fh.Filename, err)
			result.AddError(fh.Filename, couldNotParseFile)
			continue
		}

		fileEntities = append(fileEntities, entity.File{
			Name: fh.Filename,
			Size: fh.Size,
			Data: mf,
		})
	}

	return fileEntities
}

func closeFiles(files []entity.File, log *slog.Logger) {
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

		log := h.log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, ok := getUserID(r)
		if !ok {
			log.Error("failed to get user id from request context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to get files info"))
			return
		}

		metadata, err := h.service.GetSavedFilesInfo(userID)
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

		log := h.log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, ok := getUserID(r)
		if !ok {
			log.Error("failed to get user_id")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to upload files"))
			return
		}

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

		err := h.service.Download(req, w, userID)
		if err != nil {
			if errors.Is(err, service.ErrNotAllFilesExist) {
				log.Info(notAllFilesFound)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.Error(err.Error()))
			} else {
				log.Error("download failed", slog.Any("error", err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error(err.Error()))
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

        log := h.log.With(
            slog.String("op", op),
            slog.String("request_id", middleware.GetReqID(r.Context())),
        )

		userID, ok := getUserID(r)
		if !ok {
			log.Error("failed to get user_id")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to upload files"))
			return
		}

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

        result := h.service.DeleteFiles(req, userID, log)

        if len(result.Errors) > 0 {
            render.Status(r, http.StatusMultiStatus)
        } else {
            render.Status(r, http.StatusOK)
        }

        log.Info("delete process finished",
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

func getUserID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(mw.UserIDKey).(uuid.UUID)
	return id, ok
}