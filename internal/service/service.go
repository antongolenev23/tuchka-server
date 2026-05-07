package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/antongolenev23/tuchka-server/internal/auth"
	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
	"github.com/antongolenev23/tuchka-server/internal/storage"
)

var (
	ErrNotAllFilesExist  = errors.New("not all files exist")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotExists     = errors.New("user not exists")
	ErrInvalidPassword   = errors.New("invalid password")
)

type Service interface {
	Upload(ctx context.Context, files []entity.File, result *entity.OperationResult, userID uuid.UUID, log *slog.Logger)
	GetSavedFilesInfo(ctx context.Context, userID uuid.UUID) ([]dto.MetadataOutput, error)
	DeleteFiles(ctx context.Context, req dto.FilesList, userID uuid.UUID, log *slog.Logger) entity.OperationResult
	Download(ctx context.Context, filesReq dto.FilesList, w io.Writer, userID uuid.UUID) error
	Register(ctx context.Context, email, password string) (string, entity.User, error)
	Login(ctx context.Context, email, password string) (string, entity.User, error)
}

type service struct {
	repo    repository.Repository
	storage storage.Storage
	cfg     *config.Config
}

func New(repo repository.Repository, storage storage.Storage, cfg *config.Config) Service {
	return &service{
		repo:    repo,
		storage: storage,
		cfg:     cfg,
	}
}

func (s *service) Register(ctx context.Context, email, password string) (string, entity.User, error) {
	const op = "service.Register"

	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return "", entity.User{}, ErrUserAlreadyExists
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	user := entity.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	id, err := s.repo.Create(ctx, user)
	if err != nil {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	token, err := auth.GenerateToken(id, s.cfg)
	if err != nil {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return token, user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, entity.User, error) {
	const op = "service.Login"

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", entity.User{}, ErrUserNotExists
		}
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", entity.User{}, ErrInvalidPassword
	}

	token, err := auth.GenerateToken(user.ID, s.cfg)
	if err != nil {
		return "", entity.User{}, err
	}

	return token, user, nil
}

func (s *service) Upload(ctx context.Context, files []entity.File, result *entity.OperationResult, userID uuid.UUID, log *slog.Logger) {
	const op = "service.Upload"

	for _, f := range files {
		if err := ctx.Err(); err != nil {
			log.Warn("context cancelled",
				slog.String("error", err.Error()),
			)
			break
		}

		safeName := filepath.Base(f.Name)

		path, size, err := s.storage.Save(ctx, safeName, userID, f.Data)
		if err != nil {
			handleFilesError(result, safeName, "failed to save file", op, log, err)
			continue
		}

		metadata := model.MetadataInput{
			Name:   safeName,
			Path:   path,
			Size:   size,
			UserID: userID,
		}

		err = s.repo.SaveFileMetadata(metadata)
		if err != nil {
			handleFilesError(result, safeName, "failed to save file metadata", op, log, err)

			err := s.storage.Remove(path)
			if err != nil {
				log.Error(
					"failed to cleanup file",
					slog.String("path", path),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
			}
			continue
		}

		result.AddSuccess(safeName)
	}
}

func (s *service) GetSavedFilesInfo(ctx context.Context, userID uuid.UUID) ([]dto.MetadataOutput, error) {
	const op = "service.GetSavedFilesInfo"

	output, err := s.repo.GetFilesMetadata(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return output, nil
}

func (s *service) DeleteFiles(ctx context.Context, req dto.FilesList, userID uuid.UUID, log *slog.Logger) entity.OperationResult {
	const op = "service.DeleteFiles"

	var result entity.OperationResult

	for _, name := range req.Files {
		if err := ctx.Err(); err != nil {
			log.Warn("context cancelled",
				slog.String("error", err.Error()),
			)
			break
		}

		filePath, err := s.repo.GetFilePath(ctx, name, userID)
		if err != nil {
			handleFilesError(&result, name, "failed to get file path", op, log, err)
			continue
		}

		if err := s.storage.Remove(filePath); err != nil {
			if errors.Is(err, storage.ErrFileNotFound) {
				log.Info("file not found",
					slog.String("filename", name),
				)
				result.AddError(name, "file not found")
			} else {
				log.Error("failed to delete file from disk",
					slog.String("filename", name),
					slog.String("path", filePath),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
				result.AddError(name, "failed to delete file")
			}
			log.Warn("file not found on disk, deleting metadata only",
				slog.String("op", op),
				slog.String("filename", name),
				slog.String("path", filePath),
			)
			continue
		}

		if err := s.repo.DeleteFileMetadata(name, userID); err != nil {
			log.Error("failed to delete metadata from DB",
				slog.String("filename", name),
				slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
			)
			result.AddError(name, "failed to delete file")
			continue
		}

		result.AddSuccess(name)

		log.Info("file deleted",
			slog.String("op", op),
			slog.String("filename", name),
		)
	}

	return result
}

func (s *service) Download(ctx context.Context, req dto.FilesList, w io.Writer, userID uuid.UUID) error {
	const op = "service.Download"

	files, err := s.repo.GetFilePaths(ctx, req, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMetadataNotFound) {
			return ErrNotAllFilesExist
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	var storageFiles []entity.FilePath
	for _, f := range files {
		storageFiles = append(storageFiles, entity.FilePath{
			Name: f.Name,
			Path: f.Path,
		})
	}

	if err := s.storage.WriteZIP(ctx, w, storageFiles); err != nil {
		if errors.Is(err, storage.ErrFileNotFound) {
			return ErrNotAllFilesExist
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func handleFilesError(
	result *entity.OperationResult,
	fileName string,
	defaultMessage string,
	op string,
	log *slog.Logger,
	err error,
) {
	switch {
	case errors.Is(err, context.Canceled):
		log.Info("request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		log.Warn("request timeout")

	case errors.Is(err, storage.ErrFileAlreadyExists):
		log.Info("file already exists",
			slog.String("filename", fileName),
		)
		result.AddError(fileName, "file already exists")

	case errors.Is(err, repository.ErrMetadataAlreadyExists):
		log.Info("file metadata already exists", slog.String("filename", fileName))
		result.AddError(fileName, "file already exists")

	case errors.Is(err, repository.ErrMetadataNotFound):
		log.Info("file metadata not found",
			slog.String("filename", fileName),
		)
		result.AddError(fileName, "file not found")
	default:
		log.Error(defaultMessage,
			slog.String("filename", fileName),
			slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
		)
		result.AddError(fileName, "operation failed")
	}
}
