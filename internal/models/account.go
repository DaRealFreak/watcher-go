package models

// Account contains all required data from accounts in the application
type Account struct {
	ID       int
	Module   string
	Username string
	Password string
	Disabled bool
}
