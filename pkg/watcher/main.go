// Package watcher is the implementation of the application regardless of CLI or UI
package watcher

import (
	"fmt"
	"sync"

	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	log "github.com/sirupsen/logrus"

	// registered modules imported for registering into the module factory
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/deviantart"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/ehentai"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/giantessworld"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/jinjamodoki"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/patreon"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/sankakucomplex"
	_ "github.com/DaRealFreak/watcher-go/pkg/modules/twitter"
)

// DefaultDatabasePath is the default path for the database file
const DefaultDatabasePath = "./watcher.db"

// DefaultConfigurationPath is the default path for the settings file
const DefaultConfigurationPath = "./.watcher.yaml"

// Watcher contains the database connection and module factory of the main application
type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
	Cfg           *AppConfiguration
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
		DisableColors            bool
		ForceColors              bool
		DisableTimestamp         bool
		UseUppercaseLevel        bool
		UseTimePassedAsTimestamp bool
	}
	// sentry toggles
	EnableSentry  bool
	DisableSentry bool
	// run specific options
	Run struct {
		RunParallel       bool
		Items             []string
		DownloadDirectory string
		ModuleURL         []string
		DisableURL        []string
	}
}

// NewWatcher initializes a new Watcher with the default settings
func NewWatcher(cfg *AppConfiguration) *Watcher {
	watcher := &Watcher{
		DbCon:         database.NewConnection(),
		ModuleFactory: modules.GetModuleFactory(),
		Cfg:           cfg,
	}

	for _, module := range watcher.ModuleFactory.GetAllModules() {
		module.SetDbIO(watcher.DbCon)
	}

	return watcher
}

// Run is the main functionality, updates all tracked items either parallel or linear
func (app *Watcher) Run() {
	trackedItems := app.getRelevantTrackedItems()

	app.initializeUsedModules(trackedItems)

	if app.Cfg.Run.RunParallel {
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
			if !module.LoggedIn && !module.TriedLogin {
				app.loginToModule(module)
			}

			log.WithField("module", module.Key).Info(
				fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
			)

			if err := module.Parse(item); err != nil {
				log.WithField("module", item.Module).Warningf(
					"error occurred parsing item %s, skipping", item.URI,
				)
			}
		}
	}
}

// getRelevantTrackedItems returns the relevant tracked items based on the passed app configuration
func (app *Watcher) getRelevantTrackedItems() []*models.TrackedItem {
	var trackedItems []*models.TrackedItem

	switch {
	case len(app.Cfg.Run.Items) > 0:
		for _, itemURL := range app.Cfg.Run.Items {
			module := app.ModuleFactory.GetModuleFromURI(itemURL)
			if !app.ModuleFactory.IsModuleIncluded(module, app.Cfg.Run.ModuleURL) ||
				app.ModuleFactory.IsModuleExcluded(module, app.Cfg.Run.DisableURL) {
				log.WithField("module", module.Key).Warningf(
					"ignoring directly passed item %s due to not matching the module constraints",
					itemURL,
				)

				continue
			}

			trackedItems = append(trackedItems, app.DbCon.GetFirstOrCreateTrackedItem(itemURL, module))
		}
	case len(app.Cfg.Run.ModuleURL) > 0:
		for _, moduleURL := range app.Cfg.Run.ModuleURL {
			module := app.ModuleFactory.GetModuleFromURI(moduleURL)
			if app.ModuleFactory.IsModuleExcluded(module, app.Cfg.Run.DisableURL) {
				continue
			}

			trackedItems = append(trackedItems, app.DbCon.GetTrackedItems(module, false)...)
		}
	default:
		trackedItems = app.DbCon.GetTrackedItems(nil, false)
	}

	return trackedItems
}

// runForItems is the go routine to parse run parallel for groups
func (app *Watcher) runForItems(moduleKey string, trackedItems []*models.TrackedItem, wg *sync.WaitGroup) {
	defer wg.Done()

	module := app.ModuleFactory.GetModule(moduleKey)
	if app.ModuleFactory.IsModuleExcluded(module, app.Cfg.Run.DisableURL) {
		// don't run for excluded modules
		return
	}

	if !module.LoggedIn && !module.TriedLogin {
		app.loginToModule(module)
	}

	for _, item := range trackedItems {
		log.WithField("module", module.Key).Info(
			fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
		)

		if err := module.Parse(item); err != nil {
			log.WithField("module", item.Module).Warningf(
				"error occurred parsing item %s, skipping", item.URI,
			)
		}
	}
}

// loginToModule handles the login for modules, if an account exists: login
func (app *Watcher) loginToModule(module *models.Module) {
	log.WithField("module", module.Key).Info(
		fmt.Sprintf("logging in for module %s", module.Key),
	)

	account := app.DbCon.GetAccount(module)

	// no account available but module requires a login
	if account == nil {
		if module.RequiresLogin {
			log.WithField("module", module.Key).Errorf(
				"module requires a login, but no account could be found",
			)
		} else {
			return
		}
	}

	// login into the module
	if module.Login(account) {
		log.WithField("module", module.Key).Info("login successful")
	} else {
		if module.RequiresLogin {
			log.WithField("module", module.Key).Errorf(
				"module requires a login, but the login failed",
			)
		} else {
			log.WithField("module", module.Key).Warning("login not successful")
		}
	}
}

func (app *Watcher) initializeUsedModules(items []*models.TrackedItem) {
	var initializedModules []string

	for _, item := range items {
		foundModule := false

		for _, initializedModule := range initializedModules {
			if item.Module == initializedModule {
				foundModule = true
				break
			}
		}

		if !foundModule {
			module := app.ModuleFactory.GetModuleFromURI(item.URI)
			if app.ModuleFactory.IsModuleExcluded(module, app.Cfg.Run.DisableURL) {
				// don't initialize excluded modules
				continue
			}

			log.WithField("module", module.Key).Debug(
				"initializing module",
			)
			module.InitializeModule()

			initializedModules = append(initializedModules, item.Module)
		}
	}
}
