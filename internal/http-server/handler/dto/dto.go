package dto

import (
	"github.com/antongolenev23/tuchka-server/internal/file"
	resp "github.com/antongolenev23/tuchka-server/pkg/api/response"
	"github.com/antongolenev23/tuchka-server/pkg/types"
)

type MetadataOutput struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	CreatedAt types.HumanTime `json:"created_at"`
}

type FilesList struct {
    Files []string `json:"files"`
}

type FileUploadResult struct {
	FileName string `json:"file_name"`
	resp.Response
}

type ResponseBody struct {
	Results []FileUploadResult `json:"results"`
}

func GetResultDTO(r file.Result) ResponseBody {
	var respBody ResponseBody

	for _, name := range r.Success {
		respBody.Results = append(respBody.Results, FileUploadResult{
			Response: resp.OK(),
			FileName: name,
		})
	}

	for name, msg := range r.Errors {
		respBody.Results = append(respBody.Results, FileUploadResult{
			Response: resp.Error(msg),
			FileName: name,
		})
	}

	return respBody
}
