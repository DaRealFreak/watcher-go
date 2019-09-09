package session

// FileNotFoundError will get returned if the status code of the response is 404
type FileNotFoundError struct {
	message string
}

// Error returns the set message for the error struct
func (e FileNotFoundError) Error() string {
	return e.message
}
