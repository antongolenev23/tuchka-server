package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func New(storagePath string) (*PostgresRepository, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("pgx", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PostgresRepository{db: db}, nil
}

func (s *PostgresRepository) SaveFileMetainfo(info file.MetaInfo) error {
	const op = "storage.postgres.SaveFileMetaInfo"

	query := `INSERT INTO files(name, path, size) VALUES($1, $2, $3)`

	_, err := s.db.Exec(query, info.Name, info.Path, info.Size)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == "23505" {
				return repository.ErrFileAlreadyExists
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// func (s *PostgresRepository) GetFilesMetainfo() ([]file.MetainfoToGet, error) {
// 	const op = "storage.postgres.GetFileMetaInfo"

// 	query := `
// 		SELECT name, path, size, created_at
// 		FROM files
// 	`

// 	rows, err := s.db.Query(query)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s: %w", op, err)
// 	}
// 	defer rows.Close()

// 	var metainformations []file.MetainfoToGet

// 	for rows.Next() {
// 		var metainfo file.MetainfoToGet

// 		err := rows.Scan(
// 			&metainfo.Name,
// 			&metainfo.Path,
// 			&metainfo.Size,
// 			&metainfo.CreatedAt,
// 		)

// 		if err != nil {
// 			return nil, fmt.Errorf("%s: %w", op, err)
// 		}

// 		metainformations = append(metainformations, metainfo)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, fmt.Errorf("%s: %w", op, err)
// 	}

// 	return metainformations, nil
// }

// func (s *PostgresRepository) DeleteFileMetainfo(name string) error {
// 	const op = "storage.postgres.DeleteFileMetaInfo"

// 	query := `DELETE FROM files WHERE name = $1`

// 	_, err := s.db.Exec(query, name)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }
