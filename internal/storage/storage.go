package storage

import(
	"io"
)

type Storage interface {
	Save(fileName string, r io.Reader) (path string, size int64, err error)
	Remove(path string) error
}