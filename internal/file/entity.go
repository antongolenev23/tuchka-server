package file

import(
	"io"
)

type MetaInfo struct {
	Name string
	Path string
	Size int64
}

type UploadFile struct {
	Name string
	Size int64
	Data io.ReadCloser
}

type UploadResult struct {
	Success []string
	Errors  map[string]string
}

func (u *UploadResult) AddSuccess(fileName string) {
	u.Success = append(u.Success, fileName)
}

func (u *UploadResult) AddError(fileName string, errStr string) {
	u.Errors[fileName] = errStr
}

