package ajaxapi

// APIError contains error messages of the API
type APIError struct {
	ErrorMessage string `json:"error"`
}

// Error returns the occurred API error message
func (e APIError) Error() string {
	return e.ErrorMessage
}
