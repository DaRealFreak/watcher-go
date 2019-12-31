package api

import (
	"encoding/json"
	"fmt"
)

// Error is the struct of the error API responses
type Error struct {
	ErrorMessage     string            `json:"error"`
	ErrorDescription string            `json:"error_description"`
	ErrorDetails     map[string]string `json:"error_details"`
	ErrorCode        json.Number       `json:"error_code"`
	Status           string            `json:"status"`
}

// Error returns the occurred API error message
func (e Error) Error() string {
	return fmt.Sprintf("%s (%s)", e.ErrorMessage, e.ErrorDescription)
}
