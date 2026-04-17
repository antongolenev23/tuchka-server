package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func New(storagePath string) (*PostgresRepository, error) {
	const op = "repository.postgres.New"

	db, err := sql.Open("pgx", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PostgresRepository{db: db}, nil
}

func (s *PostgresRepository) SaveFileMetadata(info model.MetadataInput) error {
	const op = "repository.postgres.SaveFileMetadata"

	query := `INSERT INTO files(name, path, size) VALUES($1, $2, $3)`

	_, err := s.db.Exec(query, info.Name, info.Path, info.Size)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == repository.UniqueViolation {
				return fmt.Errorf("%s: %w", op, repository.ErrMetadataAlreadyExists) 
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *PostgresRepository) GetFilesMetadata() ([]dto.MetadataOutput, error) {
	const op = "repository.postgres.GetFileMetadata"

	query := `
		SELECT name, size, created_at
		FROM files
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var allMetadata []dto.MetadataOutput

	for rows.Next() {
		var metadata dto.MetadataOutput

		err := rows.Scan(
			&metadata.Name,
			&metadata.Size,
			&metadata.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		allMetadata = append(allMetadata, metadata)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return allMetadata, nil
}

func (s *PostgresRepository) GetFilePaths(downloadReq dto.FilesList) ([]file.FilePath, error) {
	const op = "repository.postgres.GetFilePaths"

	query := `SELECT name, path FROM files WHERE name = $1`

	var files []file.FilePath

	stmt, err := s.db.Prepare(query)
    if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }
    defer stmt.Close()

	for _, name := range downloadReq.Files {
        var f file.FilePath
        
        err := stmt.QueryRow(name).Scan(&f.Name, &f.Path)
        if err != nil {
            if errors.Is(err, sql.ErrNoRows) {
                return nil, fmt.Errorf("%s: %w", op, repository.ErrMetadataNotFound)
            }
            return nil, fmt.Errorf("%s: %w", op, err)
        }
        
        files = append(files, f)
    }

	return files, nil
}

func (s *PostgresRepository) DeleteFileMetadata(name string) error {
    const op = "repository.postgres.DeleteFileMetadata"

    query := `DELETE FROM files WHERE name = $1`

    _, err := s.db.Exec(query, name)
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }

    return nil
}

func (s *PostgresRepository) GetFilePath(name string) (string, error) {
    const op = "repository.postgres.GetFilePath"

    query := `SELECT path FROM files WHERE name = $1`

    var path string
    err := s.db.QueryRow(query, name).Scan(&path)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, repository.ErrMetadataNotFound)
        }
        return "", fmt.Errorf("%s: %w", op, err)
    }

    return path, nil
}