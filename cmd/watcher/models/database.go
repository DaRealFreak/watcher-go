package models

type DatabaseInterface interface {
	// tracked item functions
	GetTrackedItems(module *Module) []*TrackedItem
	GetFirstOrCreateTrackedItem(uri string, module *Module) *TrackedItem
	CreateTrackedItem(uri string, module *Module)
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)

	// account functions
	CreateAccount(user string, password string, module *Module)
	GetFirstOrCreateAccount(user string, password string, module *Module) *Account
	GetAccount(module *Module) *Account
}

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
