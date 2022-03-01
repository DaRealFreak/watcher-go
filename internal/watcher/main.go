// Package watcher is the implementation of the application regardless of CLI or UI
package watcher

import (
	"fmt"
	"sync"

	"github.com/DaRealFreak/watcher-go/internal/configuration"

	"github.com/DaRealFreak/watcher-go/internal/database"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	log "github.com/sirupsen/logrus"

	// registered modules imported for registering into the module factory
	_ "github.com/DaRealFreak/watcher-go/internal/modules/chounyuu"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/deviantart"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/ehentai"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/fourchan"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/gdrive"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/giantessworld"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/jinjamodoki"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/nhentai"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/patreon"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/pixiv"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/sankakucomplex"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/twitter"
)

// DefaultDatabasePath is the default path for the database file
const DefaultDatabasePath = "./watcher.db"

// DefaultConfigurationPath is the default path for the settings file
const DefaultConfigurationPath = "./.watcher.yaml"

// Watcher contains the database connection and module factory of the main application
type Watcher struct {
	DbCon         *database.DbIO
	ModuleFactory *modules.ModuleFactory
	Cfg           *configuration.AppConfiguration
}

// NewWatcher initializes a new Watcher with the default settings
func NewWatcher(cfg *configuration.AppConfiguration) *Watcher {
	watcher := &Watcher{
		DbCon:         database.NewConnection(),
		ModuleFactory: modules.GetModuleFactory(),
		Cfg:           cfg,
	}

	for _, module := range watcher.ModuleFactory.GetAllModules() {
		module.SetDbIO(watcher.DbCon)
		module.SetCfg(cfg)
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

			if app.Cfg.Run.ForceNew && item.CurrentItem != "" {
				log.WithField("module", module.Key).Info(
					fmt.Sprintf("resetting progress for item %s (current id: %s)", item.URI, item.CurrentItem),
				)
				item.CurrentItem = ""
			}

			log.WithField("module", module.Key).Info(
				fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
			)

			if err := module.Parse(item); err != nil {
				log.WithField("module", item.Module).Warningf(
					"error occurred parsing item %s (%s), skipping", item.URI, err.Error(),
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
		if app.Cfg.Run.ForceNew && item.CurrentItem != "" {
			log.WithField("module", module.Key).Info(
				fmt.Sprintf("resetting progress for item %s (current id: %s)", item.URI, item.CurrentItem),
			)
			item.CurrentItem = ""
		}

		log.WithField("module", module.Key).Info(
			fmt.Sprintf("parsing item %s (current id: %s)", item.URI, item.CurrentItem),
		)

		if err := module.Parse(item); err != nil {
			log.WithField("module", item.Module).Warningf(
				"error occurred parsing item %s (%s), skipping", item.URI, err.Error(),
			)
		}
	}
}

// loginToModule handles the login for modules, if an account exists: login
func (app *Watcher) loginToModule(module *models.Module) {
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

	log.WithField("module", module.Key).Info(
		fmt.Sprintf("logging in for module %s", module.Key),
	)

	// login into the module
	if module.Login(account) {
		log.WithField("module", module.Key).Info("login successful")
	} else {
		if module.RequiresLogin {
			log.WithField("module", module.Key).Fatalf(
				"module requires a login, but the login failed",
			)
		} else {
			log.WithField("module", module.Key).Fatalf("login not successful")
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
