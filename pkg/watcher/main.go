package watcher

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	log "github.com/sirupsen/logrus"
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
func (app *Watcher) Run(moduleURL string) {
	var trackedItems []*models.TrackedItem
	if moduleURL != "" {
		module := app.ModuleFactory.GetModuleFromURI(moduleURL)
		trackedItems = app.DbCon.GetTrackedItems(module, false)
	} else {
		trackedItems = app.DbCon.GetTrackedItems(nil, false)
	}
	for _, item := range trackedItems {
		module := app.ModuleFactory.GetModule(item.Module)
		if !module.IsLoggedIn() {
			app.loginToModule(module)
		}
		log.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem))
		module.Parse(item)
	}
}

// extract the module based on the uri and add account if not registered already
func (app *Watcher) AddAccountByURI(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.GetFirstOrCreateAccount(user, password, module)
}

// list all accounts with the option to limit it to a module
func (app *Watcher) ListAccounts(uri string) {
	var accounts []*models.Account
	if uri == "" {
		accounts = app.DbCon.GetAllAccounts(nil)
	} else {
		module := app.ModuleFactory.GetModuleFromURI(uri)
		accounts = app.DbCon.GetAllAccounts(module)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tUsername\tPassword\tModule\tDisabled")
	for _, account := range accounts {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%t\n",
			account.ID,
			account.Username,
			account.Password,
			account.Module,
			account.Disabled,
		)
	}
	_ = w.Flush()
}

func (app *Watcher) UpdateAccountDisabledStatusByURI(uri string, user string, disabled bool) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateAccountDisabledStatus(user, disabled, module)
}

func (app *Watcher) UpdateAccountByURI(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app *Watcher) AddItemByURI(uri string, currentItem string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	trackedItem := app.DbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.DbCon.UpdateTrackedItem(trackedItem, currentItem)
	}
}

// list all tracked items with the option to limit it to a module
func (app *Watcher) ListTrackedItems(uri string, includeCompleted bool) {
	var trackedItems []*models.TrackedItem
	if uri == "" {
		trackedItems = app.DbCon.GetTrackedItems(nil, includeCompleted)
	} else {
		module := app.ModuleFactory.GetModuleFromURI(uri)
		trackedItems = app.DbCon.GetTrackedItems(module, includeCompleted)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tModule\tUrl\tCurrent Item\tCompleted")
	for _, item := range trackedItems {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%t\n",
			item.ID,
			item.Module,
			item.URI,
			item.CurrentItem,
			item.Complete,
		)
	}
	_ = w.Flush()
}

// list all registered modules
func (app *Watcher) ListRegisteredModules() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tModule Key\tRequires Login")
	for index, module := range app.ModuleFactory.GetAllModules() {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%t\n",
			index,
			module.Key(),
			module.RequiresLogin(),
		)
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
