package repository

import (
	"context"
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

type FileMetadata interface {
	SaveFileMetadata(info model.MetadataInput) error
	GetFilesMetadata(ctx context.Context, userID uuid.UUID) ([]dto.MetadataOutput, error)
	GetFilePaths(ctx context.Context, downloadReq dto.FilesList, userID uuid.UUID) ([]entity.FilePath, error)
	DeleteFileMetadata(name string, userID uuid.UUID) error
	GetFilePath(ctx context.Context, name string, userID uuid.UUID) (string, error)
}

type UserAccount interface {
	Create(ctx context.Context, user entity.User) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (entity.User, error)
}

type Repository interface {
	FileMetadata
	UserAccount
	Close() error
}
