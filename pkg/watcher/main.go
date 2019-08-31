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

// DefaultDatabasePath is the default path for the database file
const DefaultDatabasePath = "./watcher.db"

// DefaultConfigurationPath is the default path for the settings file
const DefaultConfigurationPath = "./.watcher.yaml"

// Watcher contains the database connection and module factory of the main application
type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
}

// BackupSettings are the possible configuration settings for backups and recoveries
type BackupSettings struct {
	Database struct {
		Accounts struct {
			Enabled bool
		}
		Items struct {
			Enabled bool
		}
		SQL bool
	}
	Settings bool
}

// AppConfiguration contains the persistent configurations/settings across all commands
type AppConfiguration struct {
	ConfigurationFile string
	LogLevel          string
	// database file location
	Database string
	// backup options
	Backup struct {
		BackupSettings
		Archive struct {
			Zip  bool
			Tar  bool
			Gzip bool
		}
	}
	Restore struct {
		BackupSettings
	}
	// cli specific options
	Cli struct {
		ForceColors bool
		ForceFormat bool
	}
	// sentry toggles
	EnableSentry  bool
	DisableSentry bool
	// run specific options
	Run struct {
		RunParallel       bool
		Items             []string
		DownloadDirectory string
		ModuleURL         string
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
func (app *Watcher) Run(cfg *AppConfiguration) {
	trackedItems := app.getRelevantTrackedItems(cfg)
	if cfg.Run.RunParallel {
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
			log.WithField("module", module.Key()).Info(
				fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
			)
			module.Parse(item)
		}
	}
}

// getRelevantTrackedItems returns the relevant tracked items based on the passed app configuration
func (app *Watcher) getRelevantTrackedItems(cfg *AppConfiguration) []*models.TrackedItem {
	var trackedItems []*models.TrackedItem
	switch {
	case len(cfg.Run.Items) > 0:
		for _, itemURL := range cfg.Run.Items {
			module := app.ModuleFactory.GetModuleFromURI(itemURL)
			if cfg.Run.ModuleURL != "" {
				selectedModule := app.ModuleFactory.GetModuleFromURI(cfg.Run.ModuleURL)
				if selectedModule.Key() != module.Key() {
					log.WithField("module", module.Key()).Warningf(
						"ignoring directly passed item %s due to not matching the passed module %s",
						itemURL, selectedModule.Key(),
					)
					continue
				}
			}
			trackedItems = append(trackedItems, app.DbCon.GetFirstOrCreateTrackedItem(itemURL, module))
		}
	case cfg.Run.ModuleURL != "":
		module := app.ModuleFactory.GetModuleFromURI(cfg.Run.ModuleURL)
		trackedItems = app.DbCon.GetTrackedItems(module, false)
	default:
		trackedItems = app.DbCon.GetTrackedItems(nil, false)
	}
	return trackedItems
}

// runForItems is the go routine to parse run parallel for groups
func (app *Watcher) runForItems(moduleKey string, trackedItems []*models.TrackedItem, wg *sync.WaitGroup) {
	defer wg.Done()
	module := app.ModuleFactory.GetModule(moduleKey)
	if !module.IsLoggedIn() {
		app.loginToModule(module)
	}

	for _, item := range trackedItems {
		log.WithField("module", module.Key()).Info(
			fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
		)
		module.Parse(item)
	}
}

// loginToModule handles the login for modules, if an account exists: login
func (app *Watcher) loginToModule(module *models.Module) {
	log.WithField("module", module.Key()).Info(fmt.Sprintf("logging in for module %s", module.Key()))
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
		log.WithField("module", module.Key()).Info("login successful")
	} else {
		if module.RequiresLogin() {
			raven.CheckError(
				fmt.Errorf("module \"%s\" requires a login, but the login failed", module.Key()),
			)
		} else {
			log.WithField("module", module.Key()).Warning("login not successful")
		}
	}
}
