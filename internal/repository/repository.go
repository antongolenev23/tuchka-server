package repository

import (
	"errors"

	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
)

var (
	ErrMetadataAlreadyExists = errors.New("metadata already exists")
	ErrMetadataNotFound = errors.New("metadata not found")
)

const (
	UniqueViolation = "23505"
)

type File interface {
	SaveFileMetadata(info model.MetadataInput) error
	GetFilesMetadata() ([]dto.MetadataOutput, error)
	GetFilePaths(downloadReq dto.FilesList) ([]file.FilePath, error)
	DeleteFileMetadata(name string) error
	GetFilePath(name string) (string, error)
}

type Repository interface {
	File
}
