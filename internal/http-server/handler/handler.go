package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	mw "github.com/antongolenev23/tuchka-server/internal/http-server/middleware"
	"github.com/antongolenev23/tuchka-server/internal/service"
	"github.com/antongolenev23/tuchka-server/pkg/api/response"
	"github.com/antongolenev23/tuchka-server/pkg/size"
)

var (
	ErrNoFilesFound = errors.New("request body contains no files")
)

type ErrTooManyFiles struct {
	MaxFiles int
}

func (e ErrTooManyFiles) Error() string {
	return fmt.Sprintf("too many files. maximum %d", e.MaxFiles)
}

type Handler struct {
	service service.Service
	log     *slog.Logger
	cfg     *config.Config
}

func New(service service.Service, cfg *config.Config, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
		cfg:     cfg,
	}
}

// Register godoc
// @Summary Регистрация пользователя
// @Description Создает нового пользователя и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Данные для регистрации"
// @Success 200 {object} dto.AuthResponse
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/register [post]
func (h *Handler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Register"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		req, ok := decodeJSON[dto.AuthRequest](w, r, 4*size.KB, log)
		if !ok {
			return
		}

		token, user, err := h.service.Register(ctx, req.Email, req.Password)
		if err != nil {
			handleAuthError(w, r, log, err)
			return
		}

		log.Info("register completed",
			slog.String("user_id", user.ID.String()),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, dto.AuthResponse{Email: user.Email, Token: token})
	}
}

// Login godoc
// @Summary Авторизация пользователя
// @Description Выполняет вход и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Данные для входа"
// @Success 200 {object} dto.AuthResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/login [post]
func (h *Handler) Login() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Login"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		req, ok := decodeJSON[dto.AuthRequest](w, r, 4*size.KB, log)
		if !ok {
			return
		}

		token, user, err := h.service.Login(ctx, req.Email, req.Password)
		if err != nil {
			handleAuthError(w, r, log, err)
			return
		}

		log.Info("login completed",
			slog.String("user_id", user.ID.String()),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, dto.AuthResponse{Email: user.Email, Token: token})
	})
}

// Upload godoc
// @Summary Загрузка файлов
// @Description Загружает один или несколько файлов
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param files formData file true "Files to upload"
// @Success 200 {object} dto.ResponseBody
// @Failure 400 {object} response.Response
// @Failure 207 {object} dto.ResponseBody
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /files/upload [post]
func (h *Handler) Upload() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Upload"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		userID, ok := getUserID(ctx, w, r, log)
		if !ok {
			return
		}

		fileHeaders, ok := validateAndParseBody(w, r, 32*size.MB, log)
		if !ok {
			return
		}

		log.Info("request body decoded",
			slog.Int("files_count", len(fileHeaders)),
		)

		var result entity.OperationResult

		files := getFileEntities(fileHeaders, &result, log)
		defer closeFiles(files, log)

		h.service.Upload(ctx, files, &result, userID, log)
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

// Files godoc
// @Summary Получить список файлов
// @Description Возвращает список сохраненных файлов пользователя
// @Tags files
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.MetadataOutput
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /files [get]
func (h *Handler) Files() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Files"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		userID, ok := getUserID(ctx, w, r, log)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1*size.MB)

		metadata, err := h.service.GetSavedFilesInfo(ctx, userID)
		if err != nil {
			handleFilesError(w, r, log, err)
			return
		}

		log.Info("get saved files info",
			slog.Any("files_info", metadata),
		)

		render.JSON(w, r, metadata)
	})
}

// Download godoc
// @Summary Скачать файлы
// @Description Архивирует и возвращает указанные файлы
// @Tags files
// @Accept json
// @Produce application/zip
// @Security BearerAuth
// @Param request body dto.FilesList true "Список файлов"
// @Success 200 {object} []byte "ZIP file content"
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /files/download [post]
func (h *Handler) Download() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Download"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		userID, ok := getUserID(ctx, w, r, log)
		if !ok {
			return
		}

		req, ok := decodeJSON[dto.FilesList](w, r, 1*size.MB, log)
		if !ok {
			return
		}

		if ok := h.validateFilesRequest(w, r, req, log, h.cfg.Files.MaxDownload); !ok {
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

		err := h.service.Download(ctx, req, w, userID)
		if err != nil {
			handleFilesError(w, r, log, err)
			return
		}
	}
}

// Delete godoc
// @Summary Удалить файлы
// @Description Удаляет указанные файлы пользователя
// @Tags files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.FilesList true "Список файлов"
// @Success 200 {object} dto.ResponseBody
// @Failure 207 {object} dto.ResponseBody
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /files/delete [post]
func (h *Handler) Delete() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.Delete"

		ctx := r.Context()
		log := h.requestLogger(ctx, op)

		userID, ok := getUserID(ctx, w, r, log)
		if !ok {
			return
		}

		req, ok := decodeJSON[dto.FilesList](w, r, 1*size.MB, log)
		if !ok {
			return
		}

		if ok := h.validateFilesRequest(w, r, req, log, h.cfg.Files.MaxDelete); !ok {
			return
		}

		log.Info("request decoded",
			slog.Int("files_count", len(req.Files)),
		)

		result := h.service.DeleteFiles(ctx, req, userID, log)

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

func(h *Handler) Health() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
}

func validateAndParseBody(w http.ResponseWriter, r *http.Request, maxSize int64, log *slog.Logger) ([]*multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	defer r.Body.Close()

	if err := r.ParseMultipartForm(maxSize); err != nil {
		log.Info("failed to parse body",
			slog.String("error", err.Error()),
		)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, "invalid request body")
		return nil, false
	}

	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		log.Info("request body is empty")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, "no files found")
		return nil, false
	}

	return fileHeaders, true
}

func getFileEntities(files []*multipart.FileHeader, result *entity.OperationResult, log *slog.Logger) []entity.File {
	fileEntities := make([]entity.File, 0, len(files))

	for _, fh := range files {
		mf, err := fh.Open()
		if err != nil {
			log.Error("can not open fileheader",
				slog.String("filename", fh.Filename),
				slog.String("error", err.Error()),
			)
			result.AddError(fh.Filename, "could not parse file")
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

func (h *Handler) validateFilesRequest(w http.ResponseWriter, r *http.Request, req dto.FilesList, log *slog.Logger, maxFiles int) bool {
	if len(req.Files) == 0 {
		log.Info("validation failed", slog.Any("error", "request body contains no files"))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("request body contains no files"))
		return false
	}

	if len(req.Files) > maxFiles {
		err := ErrTooManyFiles{MaxFiles: maxFiles}
		log.Info("validation failed", slog.Any("error", err.Error()))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))
		return false
	}

	return true
}

func (h *Handler) requestLogger(ctx context.Context, op string) *slog.Logger {
	return h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)
}

func getUserID(ctx context.Context, w http.ResponseWriter, r *http.Request, log *slog.Logger) (uuid.UUID, bool) {
	id, ok := ctx.Value(mw.UserIDKey).(uuid.UUID)
	if !ok {
		log.Error("failed to get user id from request context")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to upload files"))
	}
	return id, ok
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request, limit int64, log *slog.Logger) (T, bool) {
	var req T

	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		log.Info("invalid request body",
			slog.String("error", err.Error()),
		)

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("invalid request body"))
		return req, false
	}

	return req, true
}

func handleAuthError(
	w http.ResponseWriter,
	r *http.Request,
	log *slog.Logger,
	err error,
) {
	switch {
	case errors.Is(err, context.Canceled):
		log.Info("request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		log.Warn("request timeout")

	case errors.Is(err, service.ErrUserAlreadyExists):
		log.Info("user already exists")
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, response.Error("user already exists"))

	case errors.Is(err, service.ErrUserNotExists):
		log.Info("user not exists")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, response.Error("user not exists"))

	default:
		log.Error("internal error", slog.String("error", err.Error()))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
	}
}

func handleFilesError(
	w http.ResponseWriter,
	r *http.Request,
	log *slog.Logger,
	err error,
) {
	switch {
	case errors.Is(err, context.Canceled):
		log.Info("request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		log.Warn("request timeout")

	case errors.Is(err, service.ErrNotAllFilesExist):
		log.Info("not all files found")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))

	default:
		log.Error("internal error", slog.String("error", err.Error()))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
	}
}
