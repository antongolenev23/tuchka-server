package disk

import (
	"errors"
	"fmt"
	"io"
	"os"

	"path/filepath"

	"github.com/antongolenev23/tuchka-server/internal/config"
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
	dstPath := filepath.Join(s.Cfg.StorageDir, fileName)

	if _, err := os.Stat(dstPath); err == nil {
		return "", 0, errors.New("file already exists")
	} else if !os.IsNotExist(err) {
		return "", 0, fmt.Errorf("can not check path %s: %w", dstPath, err)
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return "", 0, fmt.Errorf("can not create file %s: %w", dstPath, err)
	}

	size, err := io.Copy(outFile, r)
	outFile.Close()
	if err != nil {
		return "", 0, fmt.Errorf("failed to save to file %s: %w", dstPath, err)
	}

	return dstPath, size, nil
}

func (s *DiskStorage) Remove(path string) error {
	return os.Remove(path)
}
