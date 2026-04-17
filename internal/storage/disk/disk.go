package disk

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"path/filepath"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/storage"
)

type DiskStorage struct {
	Cfg *config.Config
}

func New(cfg *config.Config) *DiskStorage {
	return &DiskStorage{
		Cfg: cfg,
	}
}

func (s *DiskStorage) Save(fileName string, r io.Reader) (string, int64, error) {
	const op = "storage.disk.Save"

	dstPath := filepath.Join(s.Cfg.Files.StorageDir, fileName)

	if _, err := os.Stat(dstPath); err == nil {
		return "", 0, fmt.Errorf("%s: %w", op, storage.ErrFileAlreadyExists)
	} else if !os.IsNotExist(err) {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	size, err := io.Copy(outFile, r)
	outFile.Close()
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	return dstPath, size, nil
}

func (s *DiskStorage) Remove(path string) error {
	const op = "storage.disk.Remove"
	if err := os.Remove(path); err != nil{
		if os.IsNotExist(err) {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *DiskStorage) WriteZIP(w io.Writer, files []file.FilePath) error {
	const op = "storage.disk.WriteZIP"

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, f := range files {
		file, err := os.Open(f.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return storage.ErrFileNotFound
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		writer, err := zipWriter.Create(f.Name)
		if err != nil {
			file.Close()
			return fmt.Errorf("%s: %w", op, err)
		}

		if _, err := io.Copy(writer, file); err != nil {
			file.Close()
			return fmt.Errorf("%s: %w", op, err)
		}

		file.Close()
	}

	return nil
}
