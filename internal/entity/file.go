package entity

import (
	"io"
)

type File struct {
	Name string
	Size int64
	Data io.ReadCloser
}

type FilePath struct {
	Name string
	Path string
}

type OperationResult struct {
	Success []string
	Errors  map[string]string
}

func (u *OperationResult) AddSuccess(fileName string) {
	u.Success = append(u.Success, fileName)
}

func (u *OperationResult) AddError(fileName string, errStr string) {
	if u.Errors == nil {
		u.Errors = make(map[string]string)
	}
	u.Errors[fileName] = errStr
}
