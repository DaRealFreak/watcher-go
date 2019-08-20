package models

type DatabaseInterface interface {
	// tracked item functions
	GetTrackedItems(module ModuleInterface, includeCompleted bool) []*TrackedItem
	GetFirstOrCreateTrackedItem(uri string, module ModuleInterface) *TrackedItem
	CreateTrackedItem(uri string, module ModuleInterface)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)

	// account functions
	CreateAccount(user string, password string, module ModuleInterface)
	GetFirstOrCreateAccount(user string, password string, module ModuleInterface) *Account
	GetAccount(module ModuleInterface) *Account
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)
}

type Account struct {
	ID       int
	Module   string
	Username string
	Password string
	Disabled bool
}

type TrackedItem struct {
	ID          int
	URI         string
	CurrentItem string
	Module      string
	Complete    bool
}
