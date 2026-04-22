package entity

import "github.com/google/uuid"

type User struct {
    ID           uuid.UUID `json:"-"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
}