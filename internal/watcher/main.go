// Package watcher is the implementation of the application regardless of CLI or UI
package watcher

import (
	"fmt"
	"sync"

	"github.com/DaRealFreak/watcher-go/internal/configuration"
	"github.com/DaRealFreak/watcher-go/internal/database"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"

	// registered modules imported for registering into the module factory
	_ "github.com/DaRealFreak/watcher-go/internal/modules/chounyuu"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/deviantart"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/ehentai"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/fourchan"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/giantessworld"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/jinjamodoki"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/kemono"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/nhentai"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/patreon"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/pixiv"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/sankakucomplex"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/twitter"
	_ "github.com/DaRealFreak/watcher-go/internal/modules/youtube"
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
			raven.CheckError(module.Load())

			if (app.Cfg.Run.Force || app.Cfg.Run.ResetProgress) && item.CurrentItem != "" {
				log.WithField("module", module.Key).Info(
					fmt.Sprintf("resetting progress for item %s (current id: %s)", item.URI, item.CurrentItem),
				)
				item.CurrentItem = ""
				app.DbCon.ChangeTrackedItemCompleteStatus(item, false)
				app.DbCon.UpdateTrackedItem(item, "")
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

			normalizedUri, err := module.ModuleInterface.AddItem(itemURL)
			raven.CheckError(err)

			items := app.DbCon.GetAllOrCreateTrackedItemIgnoreSubFolder(normalizedUri, module)
			for _, trackedItem := range items {
				// skip completed item if we aren't forcing new
				if trackedItem.Complete && !app.Cfg.Run.Force {
					continue
				}

				trackedItems = append(trackedItems, trackedItem)
			}
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

	// remove duplicates
	var results []*models.TrackedItem
	for _, newItem := range trackedItems {
		found := false
		for _, uniqueItem := range results {
			if newItem.URI == uniqueItem.URI {
				found = true
				break
			}
		}

		if !found {
			results = append(results, newItem)
		}
	}

	return results
}

// runForItems is the go routine to parse run parallel for groups
func (app *Watcher) runForItems(moduleKey string, trackedItems []*models.TrackedItem, wg *sync.WaitGroup) {
	defer wg.Done()

	module := app.ModuleFactory.GetModule(moduleKey)
	if app.ModuleFactory.IsModuleExcluded(module, app.Cfg.Run.DisableURL) {
		// don't run for excluded modules
		return
	}

	raven.CheckError(module.Load())

	for _, item := range trackedItems {
		if (app.Cfg.Run.Force || app.Cfg.Run.ResetProgress) && item.CurrentItem != "" {
			log.WithField("module", module.Key).Info(
				fmt.Sprintf("resetting progress for item %s (current id: %s)", item.URI, item.CurrentItem),
			)
			item.CurrentItem = ""
			app.DbCon.ChangeTrackedItemCompleteStatus(item, false)
			app.DbCon.UpdateTrackedItem(item, "")
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
