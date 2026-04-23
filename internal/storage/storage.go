package storage

import (
	"errors"
	"io"

	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/google/uuid"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrFileNotFound      = errors.New("file not found")
)

type Storage interface {
	Save(fileName string, userID uuid.UUID, r io.Reader) (path string, size int64, err error)
	Remove(path string) error
	WriteZIP(w io.Writer, files []entity.FilePath) error
}
