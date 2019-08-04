package watcher

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"text/tabwriter"
	"watcher-go/pkg/database"
	"watcher-go/pkg/models"
	"watcher-go/pkg/modules"
)

type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
}

func NewWatcher() *Watcher {
	dbIO := database.NewConnection()
	watcher := Watcher{
		DbCon:         dbIO,
		ModuleFactory: modules.NewModuleFactory(dbIO),
	}
	return &watcher
}

// main functionality, update all tracked items
func (app *Watcher) Run() {
	for _, item := range app.DbCon.GetTrackedItems(nil) {
		module := app.ModuleFactory.GetModule(item.Module)
		if !module.IsLoggedIn() {
			app.loginToModule(module)
		}
		log.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
		module.Parse(item)
	}
}

// extract the module based on the uri and add account if not registered already
func (app *Watcher) AddAccountByUri(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	app.DbCon.GetFirstOrCreateAccount(user, password, module)
}

// list all accounts with the option to limit it to a module
func (app *Watcher) ListAccounts(uri string) {
	var accounts []*models.Account
	if uri == "" {
		accounts = app.DbCon.GetAllAccounts(nil)
	} else {
		module := app.ModuleFactory.GetModuleFromUri(uri)
		accounts = app.DbCon.GetAllAccounts(module)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	_, _ = fmt.Fprintln(w, "Id\tUsername\tPassword\tModule\tDisabled")
	for _, account := range accounts {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%t\n", account.Id, account.Username, account.Password, account.Module, account.Disabled)
	}
	_ = w.Flush()
}

func (app *Watcher) UpdateAccountDisabledStatusByUri(uri string, user string, disabled bool) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	app.DbCon.UpdateAccountDisabledStatus(user, disabled, module)
}

func (app *Watcher) UpdateAccountByUri(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	app.DbCon.UpdateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app *Watcher) AddItemByUri(uri string, currentItem string) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	trackedItem := app.DbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.DbCon.UpdateTrackedItem(trackedItem, currentItem)
	}
}

// list all tracked items with the option to limit it to a module
func (app *Watcher) ListTrackedItems(uri string) {
	var trackedItems []*models.TrackedItem
	if uri == "" {
		trackedItems = app.DbCon.GetTrackedItems(nil)
	} else {
		module := app.ModuleFactory.GetModuleFromUri(uri)
		trackedItems = app.DbCon.GetTrackedItems(module)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	_, _ = fmt.Fprintln(w, "Id\tModule\tUrl\tCurrent Item\tCompleted")
	for _, item := range trackedItems {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%t\n", item.Id, item.Module, item.Uri, item.CurrentItem, item.Complete)
	}
	_ = w.Flush()
}

// list all registered modules
func (app *Watcher) ListRegisteredModules() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	_, _ = fmt.Fprintln(w, "Id\tModule Key\tRequires Login")
	for index, module := range app.ModuleFactory.GetAllModules() {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%t\n", index, module.Key(), module.RequiresLogin())
	}
	_ = w.Flush()
}

// login into the module
func (app *Watcher) loginToModule(module *models.Module) {
	log.Info(fmt.Sprintf("logging in for module %s", module.Key()))
	account := app.DbCon.GetAccount(module)

	// no account available but module requires a login
	if account == nil {
		if module.RequiresLogin() {
			log.Fatal(fmt.Sprintf("Module \"%s\" requires a login, but no account could be found", module.Key()))
		} else {
			return
		}
	}

	// login into the module
	success := module.Login(account)
	if success {
		log.Info("login successful")
	} else {
		if module.RequiresLogin() {
			log.Fatal(fmt.Sprintf("Module \"%s\" requires a login, but the login failed", module.Key()))
		} else {
			log.Warning("login not successful")
		}
	}
}
