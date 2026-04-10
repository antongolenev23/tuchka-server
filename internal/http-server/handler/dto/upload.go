package dto

import (
	"github.com/antongolenev23/tuchka-server/internal/file"
	resp "github.com/antongolenev23/tuchka-server/pkg/api/response"
)

type FileUploadResult struct {
	resp.Responce
	FileName string `json:"file_name"`
}

type ResponceBody struct {
	Results []FileUploadResult `json:"results"`
}

func GetResultDTO(r file.UploadResult) ResponceBody {
	var respBody ResponceBody

	for _, name := range r.Success {
		respBody.Results = append(respBody.Results, FileUploadResult{
			Responce: resp.OK(),
			FileName: name,
		})
	}

	for name, msg := range r.Errors {
		respBody.Results = append(respBody.Results, FileUploadResult{
			Responce: resp.Error(msg),
			FileName: name,
		})
	}

	return respBody
}