package model

import (
	"time"
)

type MetainfoToSave struct {
	Name string
	Path string
	Size int64
}

type MetainfoToGet struct {
	Name string
	Path string
	Size int64
	CreatedAt time.Time
}