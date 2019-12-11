package ajaxapi

// APIError contains error messages of the API
type APIError struct {
	ErrorOccurred bool   `json:"error"`
	ErrorMessage  string `json:"message"`
}

// APIRequestError is the error struct of invalid requests
type APIRequestError struct {
	ErrorMessage string `json:"error"`
}

// Error returns the occurred API error message
func (e APIError) Error() string {
	return e.ErrorMessage
}

// Error returns the occurred API error message
func (e APIRequestError) Error() string {
	return e.ErrorMessage
}
