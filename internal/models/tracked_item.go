package models

import "database/sql"

// TrackedItem contains all required data from tracked items in the application
type TrackedItem struct {
	ID           int
	URI          string
	SubFolder    string
	CurrentItem  string
	Module       string
	LastModified sql.NullTime
	Favorite     bool
	Complete     bool
}
