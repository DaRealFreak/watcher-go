package mobileapi

// APIError contains error messages of the API
type APIError struct {
	ErrorOccurred bool   `json:"error"`
	ErrorMessage  string `json:"message"`
}

// APIRequestError is the error struct of invalid requests
type APIRequestError struct {
	ErrorMessage string `json:"error"`
}

// DeletedIllustrationError will get returned from the illustration detail API request if the illustration got deleted
type DeletedIllustrationError struct {
	APIError
}

// DeletedUserError will get returned from the user information API request if the user got deleted
type DeletedUserError struct {
	APIError
}

// Error returns the occurred API error message
func (e APIError) Error() string {
	return e.ErrorMessage
}

// Error returns the occurred API error message
func (e APIRequestError) Error() string {
	return e.ErrorMessage
}
