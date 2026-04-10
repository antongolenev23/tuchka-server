package repository

import (
	"errors"

	"github.com/antongolenev23/tuchka-server/internal/file"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
)

type File interface {
	SaveFileMetainfo(info file.MetaInfo) error
}

type Repository interface {
	File
}
