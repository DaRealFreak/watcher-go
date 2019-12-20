package api

// authInfo contains all required information for the app authorization request
type authInfo struct {
	State        string
	ResponseType string
	CSRFToken    string
}
