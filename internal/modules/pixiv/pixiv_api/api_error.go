package pixivapi

// APIError contains error messages of the API
type APIError struct {
	ErrorOccurred bool   `json:"error"`
	ErrorMessage  string `json:"message"`
}

// APIRequestError is the error struct of invalid requests
type APIRequestError struct {
	ErrorMessage string `json:"error"`
}

// MobileAPIError is the error struct of the mobile API
type MobileAPIError struct {
	ErrorDetails struct {
		UserMessage string `json:"user_message"`
		Message     string `json:"message"`
		Reason      string `json:"reason"`
	} `json:"error"`
}

type OffsetError struct {
	APIError
}

// IllustrationUnavailableError will get returned from the illustration detail API request
// if the illustration got either deleted or made unavailable in general
type IllustrationUnavailableError struct {
	APIError
}

// UserUnavailableError will get returned from the user information API request
// if the user got deleted or made unavailable in general
type UserUnavailableError struct {
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

// Error returns the occurred API error message
func (e MobileAPIError) Error() string {
	if e.ErrorDetails.UserMessage != "" {
		return e.ErrorDetails.UserMessage
	}

	return e.ErrorDetails.Message
}
