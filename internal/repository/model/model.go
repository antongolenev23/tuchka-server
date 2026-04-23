package model

import "github.com/google/uuid"

type MetadataInput struct {
	Name   string
	Path   string
	Size   int64
	UserID uuid.UUID
}
