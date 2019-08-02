package watcher

import (
	"fmt"
	"github.com/kubernetes/klog"
	"log"
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
		klog.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
		module.Parse(item)
	}
}

// extract the module based on the uri and add account if not registered already
func (app *Watcher) AddAccountByUri(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	app.DbCon.GetFirstOrCreateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app *Watcher) AddItemByUri(uri string, currentItem string) {
	module := app.ModuleFactory.GetModuleFromUri(uri)
	trackedItem := app.DbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.DbCon.UpdateTrackedItem(trackedItem, currentItem)
	}
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
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%t", account.Id, account.Username, account.Password, account.Module, account.Disabled)
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

// login into the module
func (app *Watcher) loginToModule(module *models.Module) {
	klog.Info(fmt.Sprintf("logging in for module %s", module.Key()))
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
		klog.Info("login successful")
	} else {
		if module.RequiresLogin() {
			log.Fatal(fmt.Sprintf("Module \"%s\" requires a login, but the login failed", module.Key()))
		} else {
			klog.Warning("login not successful")
		}
	}
}
