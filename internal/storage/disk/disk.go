package disk

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"

	"path/filepath"

	"github.com/google/uuid"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/storage"
	"github.com/antongolenev23/tuchka-server/pkg/types"
)

type DiskStorage struct {
	Cfg *config.Config
}

func New(cfg *config.Config) *DiskStorage {
	return &DiskStorage{
		Cfg: cfg,
	}
}

func (s *DiskStorage) Save(ctx context.Context, fileName string, userID uuid.UUID, r io.Reader) (string, int64, error) {
	const op = "storage.disk.Save"

	userSubDirectoryName := userID.String()
	userDirPath := filepath.Join(s.Cfg.Files.StorageDir, userSubDirectoryName)
	dstPath := filepath.Join(s.Cfg.Files.StorageDir, userSubDirectoryName, fileName)

	if _, err := os.Stat(dstPath); err == nil {
		return "", 0, fmt.Errorf("%s: %w", op, storage.ErrFileAlreadyExists)
	} else if !os.IsNotExist(err) {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := os.MkdirAll(userDirPath, 0700); err != nil {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	size, err := io.Copy(outFile, &types.ContextReader{Ctx: ctx, R: r})
	outFile.Close()
	if err != nil {
		removeErr := s.Remove(dstPath)
		if removeErr != nil {
			return "", 0, fmt.Errorf("%s: %w, %w", op, err, removeErr)
		}

		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	return dstPath, size, nil
}

func (s *DiskStorage) Remove(path string) error {
	const op = "storage.disk.Remove"
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *DiskStorage) WriteZIP(ctx context.Context, w io.Writer, files []entity.FilePath) error {
	const op = "storage.disk.WriteZIP"

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

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

		if _, err := io.Copy(writer, &types.ContextReader{Ctx: ctx, R: file}); err != nil {
			file.Close()
			return fmt.Errorf("%s: %w", op, err)
		}

		file.Close()
	}

	return nil
}
