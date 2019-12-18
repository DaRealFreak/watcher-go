package models

import (
	"database/sql"
	"time"
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

// GetDisplayExpirationDate returns an empty string if the date is null or else an RFC822 formatted string
func (c *Cookie) GetDisplayExpirationDate() string {
	if !c.Expiration.Valid {
		return ""
	}

	return c.Expiration.Time.Format(time.RFC822)
}
