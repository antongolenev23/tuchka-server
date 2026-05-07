package storage

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/antongolenev23/tuchka-server/internal/entity"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrFileNotFound      = errors.New("file not found")
)

type Storage interface {
	Save(ctx context.Context, fileName string, userID uuid.UUID, r io.Reader) (path string, size int64, err error)
	Remove(path string) error
	WriteZIP(ctx context.Context, w io.Writer, files []entity.FilePath) error
}
