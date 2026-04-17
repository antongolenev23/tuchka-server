package service

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
	"github.com/antongolenev23/tuchka-server/internal/storage"
)

const(
	fileAlreadyExists = "file already exists"
	fileNotFound = "file not found"
	failedToSaveFile = "failed to save file"
	failedToDeleteFile = "failed to delete file"
)

var(
	ErrNotAllFilesExist = errors.New("not all files exist")
)

type Service interface {
	Upload(files []file.File, result *file.Result, log *slog.Logger)
	GetSavedFilesInfo() ([]dto.MetadataOutput, error)
	GetFilePaths(fileNames dto.FilesList) ([]file.FilePath, error)
	DeleteFiles(req dto.FilesList, log *slog.Logger) file.Result
	Download(filesReq dto.FilesList, w io.Writer) error
}

type service struct {
	repo    repository.Repository
	storage storage.Storage
}

func New(repo repository.Repository, storage storage.Storage) Service {
	return &service{
		repo:    repo,
		storage: storage,
	}
}

func (s *service) Upload(files []file.File, result *file.Result, log *slog.Logger) {
	const op = "service.Upload"

	for _, f := range files {
		safeName := filepath.Base(f.Name)

		path, size, err := s.storage.Save(safeName, f.Data)
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
			Name: safeName,
			Path: path,
			Size: size,
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

func (s *service) GetSavedFilesInfo() ([]dto.MetadataOutput, error) {
	const op = "service.GetSavedFilesInfo"

	output, err := s.repo.GetFilesMetadata();
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return output, nil
}

func (s *service) GetFilePaths(fileNames dto.FilesList) ([]file.FilePath, error) {
	const op = "service.GetFilePaths"

	output, err := s.repo.GetFilePaths(fileNames)
	if err != nil {
		if errors.Is(err, repository.ErrMetadataNotFound) {
			return nil, ErrNotAllFilesExist
		} 
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return output, nil
}

func (s *service) DeleteFiles(req dto.FilesList, log *slog.Logger) file.Result {
	const op = "service.DeleteFiles"

	var result file.Result

	for _, name := range req.Files {
		filePath, err := s.repo.GetFilePath(name)
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

		if err := s.repo.DeleteFileMetadata(name); err != nil {
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

func (s *service) Download(req dto.FilesList, w io.Writer) error {
	const op = "service.Download"

	files, err := s.repo.GetFilePaths(req)
	if err != nil {
		if errors.Is(err, repository.ErrMetadataNotFound) {
			return ErrNotAllFilesExist
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	var storageFiles []file.FilePath
	for _, f := range files {
		storageFiles = append(storageFiles, file.FilePath{
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
