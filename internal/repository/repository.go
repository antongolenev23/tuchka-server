package repository

import (
	"errors"

	"github.com/google/uuid"

	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
)

var (
	ErrMetadataAlreadyExists = errors.New("metadata already exists")
	ErrMetadataNotFound      = errors.New("metadata not found")
	ErrUserNotFound          = errors.New("user not found")
)

const (
	UniqueViolation = "23505"
)

type File interface {
	SaveFileMetadata(info model.MetadataInput) error
	GetFilesMetadata(userID uuid.UUID) ([]dto.MetadataOutput, error)
	GetFilePaths(downloadReq dto.FilesList, userID uuid.UUID) ([]entity.FilePath, error)
	DeleteFileMetadata(name string, userID uuid.UUID) error
	GetFilePath(name string, userID uuid.UUID) (string, error)
}

type User interface {
	Create(user entity.User) (uuid.UUID, error)
	GetByEmail(email string) (entity.User, error)
}

type Repository interface {
	File
	User
}
