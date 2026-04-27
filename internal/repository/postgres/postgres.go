package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/http-server/handler/dto"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/repository/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func New(cfg *config.Config) (repository.Repository, error) {
	const op = "repository.postgres.New"

	dsn := fmt.Sprintf(
		"postgres://%s:%s@db:5432/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}

		time.Sleep(2 * time.Second)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("%s: failed to connect to DB after retries: %w", op, pingErr)
	}

	return &PostgresRepository{db: db}, nil
}

func (p *PostgresRepository) Create(user entity.User) (uuid.UUID, error) {
	const op = "repository.postgres.Create"

	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id
		`
	var id uuid.UUID

	err := p.db.QueryRow(query, user.Email, user.PasswordHash).Scan(&id)
	if err != nil {
		return id, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (p *PostgresRepository) GetByEmail(email string) (entity.User, error) {
	const op = "repository.postgres.GetByEmail"

	var user entity.User
	query := `SELECT id, email, password_hash FROM users WHERE email = $1`

	err := p.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, repository.ErrUserNotFound
		}
		return user, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (p *PostgresRepository) SaveFileMetadata(info model.MetadataInput) error {
	const op = "repository.postgres.SaveFileMetadata"

	query := `INSERT INTO files(name, path, size, user_id) VALUES($1, $2, $3, $4)`

	_, err := p.db.Exec(query, info.Name, info.Path, info.Size, info.UserID)
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

func (p *PostgresRepository) GetFilesMetadata(userID uuid.UUID) ([]dto.MetadataOutput, error) {
	const op = "repository.postgres.GetFileMetadata"

	query := `
		SELECT name, size, created_at
		FROM files
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := p.db.Query(query, userID)
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

func (p *PostgresRepository) GetFilePaths(downloadReq dto.FilesList, userID uuid.UUID) ([]entity.FilePath, error) {
	const op = "repository.postgres.GetFilePaths"

	query := `
		SELECT name, path 
		FROM files 
		WHERE user_id = $1 AND name = $2 
	`

	var files []entity.FilePath

	stmt, err := p.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	for _, name := range downloadReq.Files {
		var f entity.FilePath

		err := stmt.QueryRow(userID, name).Scan(&f.Name, &f.Path)
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

func (p *PostgresRepository) DeleteFileMetadata(name string, userID uuid.UUID) error {
	const op = "repository.postgres.DeleteFileMetadata"

	query := `DELETE FROM files WHERE user_id = $1 AND name = $2`

	_, err := p.db.Exec(query, userID, name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *PostgresRepository) GetFilePath(name string, userID uuid.UUID) (string, error) {
	const op = "repository.postgres.GetFilePath"

	query := `SELECT path FROM files WHERE user_id = $1 AND name = $2`

	var path string
	err := p.db.QueryRow(query, userID, name).Scan(&path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, repository.ErrMetadataNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return path, nil
}
