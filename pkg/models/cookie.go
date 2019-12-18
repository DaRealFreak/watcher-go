package models

import "time"

// Cookie contains all the required details to set cookies for domains
type Cookie struct {
	ID         int
	Name       string
	Value      string
	Expiration time.Time
	Module     string
	Disabled   bool
}
