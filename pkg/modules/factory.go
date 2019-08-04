package modules

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/ehentai"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv"
	"github.com/DaRealFreak/watcher-go/pkg/modules/sankakucomplex"
	log "github.com/sirupsen/logrus"
	"regexp"
)

type ModuleFactory struct {
	modules    []*models.Module
	uriSchemas map[string][]*regexp.Regexp
}

// generate new module factory and register modules
func NewModuleFactory(dbIO *database.DbIO) *ModuleFactory {
	factory := ModuleFactory{
		uriSchemas: make(map[string][]*regexp.Regexp),
	}
	factory.modules = append(factory.modules, sankakucomplex.NewModule(dbIO, factory.uriSchemas))
	factory.modules = append(factory.modules, ehentai.NewModule(dbIO, factory.uriSchemas))
	factory.modules = append(factory.modules, pixiv.NewModule(dbIO, factory.uriSchemas))
	return &factory
}

// retrieve all available modules
func (f ModuleFactory) GetAllModules() []*models.Module {
	return f.modules
}

// retrieve module by it's key
func (f ModuleFactory) GetModule(moduleName string) *models.Module {
	for _, module := range f.modules {
		if module.Key() == moduleName {
			return module
		}
	}
	return nil
}

// check the registered uri schemas for a match and return the module
func (f ModuleFactory) GetModuleFromUri(uri string) *models.Module {
	for key, patternCollection := range f.uriSchemas {
		for _, pattern := range patternCollection {
			if pattern.MatchString(uri) {
				return f.GetModule(key)
			}
		}
	}
	log.Fatal(fmt.Sprintf("no module is registered which can parse based on the url %s", uri))
	return nil
}
