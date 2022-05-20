package graphql_api

import "fmt"

// TwitterError is the struct of the error API responses
type TwitterError struct {
	Errors []struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Details string `json:"details"`
	} `json:"errors"`
}

// Error returns the occurred API error message
func (e TwitterError) Error() string {
	errorMessage := ""

	for _, singleError := range e.Errors {
		errorMessage += fmt.Sprintf("%s (%s)", singleError.Details, singleError.Message)
	}

	return errorMessage
}
