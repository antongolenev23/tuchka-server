package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct{
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New" 

	db, err := sql.Open("pgx", storagePath)
	if err != nil{
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil{
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}