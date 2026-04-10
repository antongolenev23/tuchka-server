package response

type Responce struct {
	Status string `json:"status"`
	Error string `json:"error,omitempty"`
}

const (
	statusOK = "OK"
	statusError = "Error"
)

func OK() Responce{
	return  Responce{
		Status: statusOK,
	}
}

func Error(msg string) Responce{
	return  Responce{
		Status: statusError,
		Error: msg,
	}
}