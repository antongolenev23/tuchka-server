package dto

import (
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/pkg/api/response"
	"github.com/antongolenev23/tuchka-server/pkg/types"
)

type MetadataOutput struct {
	Name      string          `json:"name"`
	Size      int64           `json:"size"`
	CreatedAt types.HumanTime `json:"created_at"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type FilesList struct {
	Files []string `json:"files"`
}

type FileUploadResult struct {
	FileName string `json:"file_name"`
	Response response.Response
}

type ResponseBody struct {
	Results []FileUploadResult `json:"results"`
}

func GetResultDTO(r entity.OperationResult) ResponseBody {
	var respBody ResponseBody

	for _, name := range r.Success {
		respBody.Results = append(respBody.Results, FileUploadResult{
			FileName: name,
			Response: response.OK(),
		})
	}

	for name, msg := range r.Errors {
		respBody.Results = append(respBody.Results, FileUploadResult{
			Response: response.Error(msg),
			FileName: name,
		})
	}

	return respBody
}
