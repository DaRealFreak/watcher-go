package models

import (
	"database/sql"
)

// Cookie contains all the required details to set cookies for domains
type Cookie struct {
	ID         int
	Name       string
	Value      string
	Expiration sql.NullTime
	Module     string
	Disabled   bool
}
