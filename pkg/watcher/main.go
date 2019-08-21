package watcher

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/raven"

	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	log "github.com/sirupsen/logrus"
)

// Watcher contains the database connection and module factory of the main application
type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
}

// NewWatcher initializes a new Watcher with the default settings
func NewWatcher() *Watcher {
	dbIO := database.NewConnection()
	watcher := Watcher{
		DbCon:         dbIO,
		ModuleFactory: modules.NewModuleFactory(dbIO),
	}
	return &watcher
}

// Run is the main functionality, updates all tracked items
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

// loginToModule handles the login for modules, if an account exists: login
func (app *Watcher) loginToModule(module *models.Module) {
	log.Info(fmt.Sprintf("logging in for module %s", module.Key()))
	account := app.DbCon.GetAccount(module)

	// no account available but module requires a login
	if account == nil {
		if module.RequiresLogin() {
			raven.CheckError(
				fmt.Errorf("module \"%s\" requires a login, but no account could be found", module.Key()),
			)
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
			raven.CheckError(
				fmt.Errorf("module \"%s\" requires a login, but the login failed", module.Key()),
			)
		} else {
			log.Warning("login not successful")
		}
	}
}
