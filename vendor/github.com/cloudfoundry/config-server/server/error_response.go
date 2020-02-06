package server

import "encoding/json"

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{Error: err.Error()}
}

func (e ErrorResponse) GenerateErrorMsg() string {
	response, err := json.Marshal(e)
	if err != nil {
		return `{"error": "Unknown Error"}`
	}

	return string(response)
}
