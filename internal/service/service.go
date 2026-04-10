package service

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/storage"
	"github.com/antongolenev23/tuchka-server/internal/repository"
)

type Service interface {
	UploadFiles(files []file.UploadFile) file.UploadResult
}

type service struct {
	repo repository.Repository
	storage storage.Storage
	logger  *slog.Logger
	cfg  *config.Config
}

func New(repo repository.Repository, storage storage.Storage, logger *slog.Logger, cfg *config.Config) Service {
	return &service{
		repo: repo,
		storage: storage,
		logger: logger,
		cfg: cfg,
	}
}

func (s *service) UploadFiles(files []file.UploadFile) file.UploadResult {
	var res file.UploadResult

	for _, f := range files {
		fileName := f.Name
		safeName := filepath.Base(f.Name)

		path, size, err := s.storage.Save(safeName, f.Data)
		if err != nil{
			res.AddError(fileName, fmt.Sprintf("file save error: %s", err))
			continue
		}
		
		metainfo := file.MetaInfo{
			Name: safeName,
			Path: path,
			Size: size,
		}

		err = s.repo.SaveFileMetainfo(metainfo)
		if err != nil {
			if errors.Is(err, repository.ErrFileAlreadyExists){
				res.AddError(fileName, "file already exists")
			} else {
				res.AddError(fileName, fmt.Sprintf("failed to save file metainfo: %s", err))
			}

			err := s.storage.Remove(path)
			if err != nil {
				s.logger.Error(
					"failed to cleanup file",
					slog.String("path", path),
					slog.String("error", err.Error()),
				)
			}
			continue
		}

		res.AddSuccess(fileName)
	}

	return res
}