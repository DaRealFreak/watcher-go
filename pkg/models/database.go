package models

type DatabaseInterface interface {
	// tracked item functions
	GetTrackedItems(module ModuleInterface) []*TrackedItem
	GetFirstOrCreateTrackedItem(uri string, module ModuleInterface) *TrackedItem
	CreateTrackedItem(uri string, module ModuleInterface)
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)

	// account functions
	CreateAccount(user string, password string, module ModuleInterface)
	GetFirstOrCreateAccount(user string, password string, module ModuleInterface) *Account
	GetAccount(module ModuleInterface) *Account
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
