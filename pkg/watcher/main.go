package watcher

import (
	"fmt"
	"sync"

	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
)

// Watcher contains the database connection and module factory of the main application
type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
}

// AppConfiguration contains the persistent configurations/settings across all commands
type AppConfiguration struct {
	ConfigurationFile string
	LogLevel          string
	EnableSentry      bool
	DisableSentry     bool
	// cli specific options
	Cli struct {
		ForceColors bool
		ForceFormat bool
	}
	// database file location
	Database string
	// backup options
	Backup struct {
		Database struct {
			Accounts struct {
				Enabled bool
				URL     string
			}
			Items struct {
				Enabled bool
				URL     string
			}
			SQL bool
		}
		Settings bool
		Archive  struct {
			Zip  bool
			Tar  bool
			Gzip bool
		}
	}
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

// Run is the main functionality, updates all tracked items either parallel or linear
func (app *Watcher) Run(moduleURL string, parallel bool) {
	var trackedItems []*models.TrackedItem
	if moduleURL != "" {
		module := app.ModuleFactory.GetModuleFromURI(moduleURL)
		trackedItems = app.DbCon.GetTrackedItems(module, false)
	} else {
		trackedItems = app.DbCon.GetTrackedItems(nil, false)
	}

	if parallel {
		groupedItems := make(map[string][]*models.TrackedItem)
		for _, item := range trackedItems {
			groupedItems[item.Module] = append(groupedItems[item.Module], item)
		}
		var wg sync.WaitGroup
		wg.Add(len(groupedItems))
		for moduleKey, items := range groupedItems {
			go app.runForItems(moduleKey, items, &wg)
		}
		wg.Wait()
	} else {
		for _, item := range trackedItems {
			module := app.ModuleFactory.GetModule(item.Module)
			if !module.IsLoggedIn() {
				app.loginToModule(module)
			}
			log.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem))
			module.Parse(item)
		}
	}
}

// runForItems is the go routine to parse run parallel for groups
func (app *Watcher) runForItems(moduleKey string, trackedItems []*models.TrackedItem, wg *sync.WaitGroup) {
	defer wg.Done()
	module := app.ModuleFactory.GetModule(moduleKey)
	if !module.IsLoggedIn() {
		app.loginToModule(module)
	}

	for _, item := range trackedItems {
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
