package service

import (
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

const (
	fileAlreadyExists  = "file already exists"
	fileNotFound       = "file not found"
	failedToSaveFile   = "failed to save file"
	failedToDeleteFile = "failed to delete file"
)

var (
	ErrNotAllFilesExist  = errors.New("not all files exist")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotExists     = errors.New("user not exists")
	ErrInvalidPassword   = errors.New("invalid password")
)

type Service interface {
	Upload(files []entity.File, result *entity.OperationResult, userID uuid.UUID, log *slog.Logger)
	GetSavedFilesInfo(userID uuid.UUID) ([]dto.MetadataOutput, error)
	DeleteFiles(req dto.FilesList, userID uuid.UUID, log *slog.Logger) entity.OperationResult
	Download(filesReq dto.FilesList, w io.Writer, userID uuid.UUID) error
	Register(email, password string) (string, entity.User, error)
	Login(email, password string) (string, entity.User, error)
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

func (s *service) Register(email, password string) (string, entity.User, error) {
	const op = "service.Register"

	_, err := s.repo.GetByEmail(email)
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

	id, err := s.repo.Create(user)
	if err != nil {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	token, err := auth.GenerateToken(id, s.cfg)
	if err != nil {
		return "", entity.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return token, user, nil
}

func (s *service) Login(email, password string) (string, entity.User, error) {
	const op = "service.Login"

	user, err := s.repo.GetByEmail(email)
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

func (s *service) Upload(files []entity.File, result *entity.OperationResult, userID uuid.UUID, log *slog.Logger) {
	const op = "service.Upload"

	for _, f := range files {
		safeName := filepath.Base(f.Name)

		path, size, err := s.storage.Save(safeName, userID, f.Data)
		if err != nil {
			if errors.Is(err, storage.ErrFileAlreadyExists) {
				log.Info("file already exists",
					slog.String("filename", safeName),
				)
				result.AddError(safeName, fileAlreadyExists)
			} else {
				log.Error("failed to save file",
					slog.String("filename", safeName),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
				result.AddError(safeName, failedToSaveFile)
			}
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
			if errors.Is(err, repository.ErrMetadataAlreadyExists) {
				log.Info("file metadata already exists", slog.String("filename", safeName))
				result.AddError(safeName, fileAlreadyExists)
			} else {
				log.Error("failed to save file metadata",
					slog.String("filename", safeName),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
				result.AddError(safeName, failedToSaveFile)
			}

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

func (s *service) GetSavedFilesInfo(userID uuid.UUID) ([]dto.MetadataOutput, error) {
	const op = "service.GetSavedFilesInfo"

	output, err := s.repo.GetFilesMetadata(userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return output, nil
}

func (s *service) DeleteFiles(req dto.FilesList, userID uuid.UUID, log *slog.Logger) entity.OperationResult {
	const op = "service.DeleteFiles"

	var result entity.OperationResult

	for _, name := range req.Files {
		filePath, err := s.repo.GetFilePath(name, userID)
		if err != nil {
			if errors.Is(err, repository.ErrMetadataNotFound) {
				log.Info("file metadata not found",
					slog.String("filename", name),
				)
				result.AddError(name, fileNotFound)
			} else {
				log.Error("failed to get file metadata",
					slog.String("filename", name),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
				result.AddError(name, failedToDeleteFile)
			}
			continue
		}

		if err := s.storage.Remove(filePath); err != nil {
			if errors.Is(err, storage.ErrFileNotFound) {
				log.Info("file not found",
					slog.String("filename", name),
				)
				result.AddError(name, fileNotFound)
			} else {
				log.Error("failed to delete file from disk",
					slog.String("filename", name),
					slog.String("path", filePath),
					slog.String("error", fmt.Errorf("%s: %w", op, err).Error()),
				)
				result.AddError(name, failedToDeleteFile)
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
			result.AddError(name, failedToDeleteFile)
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

func (s *service) Download(req dto.FilesList, w io.Writer, userID uuid.UUID) error {
	const op = "service.Download"

	files, err := s.repo.GetFilePaths(req, userID)
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

	if err := s.storage.WriteZIP(w, storageFiles); err != nil {
		if errors.Is(err, storage.ErrFileNotFound) {
			return ErrNotAllFilesExist
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
