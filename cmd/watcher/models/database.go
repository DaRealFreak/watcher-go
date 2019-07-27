package models

type Account struct {
	Id       int
	Module   string
	Username string
	Password string
	Disabled bool
}

type TrackedItem struct {
	Id          int
	Uri         string
	CurrentItem string
	Module      string
	Complete    bool
}
