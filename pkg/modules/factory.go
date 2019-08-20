package modules

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/raven"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/ehentai"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv"
	"github.com/DaRealFreak/watcher-go/pkg/modules/sankakucomplex"
)

// ModuleFactory contains all registered modules and URI Schemas
type ModuleFactory struct {
	modules    []*models.Module
	uriSchemas map[string][]*regexp.Regexp
}

// NewModuleFactory generates a new module factory and register all modules
func NewModuleFactory(dbIO models.DatabaseInterface) *ModuleFactory {
	factory := ModuleFactory{
		uriSchemas: make(map[string][]*regexp.Regexp),
	}
	factory.modules = append(factory.modules, sankakucomplex.NewModule(dbIO, factory.uriSchemas))
	factory.modules = append(factory.modules, ehentai.NewModule(dbIO, factory.uriSchemas))
	factory.modules = append(factory.modules, pixiv.NewModule(dbIO, factory.uriSchemas))
	return &factory
}

// GetAllModules returns all available modules
func (f ModuleFactory) GetAllModules() []*models.Module {
	return f.modules
}

// GetModule returns a module by it's key
func (f ModuleFactory) GetModule(moduleName string) *models.Module {
	for _, module := range f.modules {
		if module.Key() == moduleName {
			return module
		}
	}
	return nil
}

// GetModuleFromURI checks the registered URI schemas for a match and returns the module
func (f ModuleFactory) GetModuleFromURI(uri string) *models.Module {
	for key, patternCollection := range f.uriSchemas {
		for _, pattern := range patternCollection {
			if pattern.MatchString(uri) {
				return f.GetModule(key)
			}
		}
	}
	raven.CheckError(fmt.Errorf("no module is registered which can parse based on the url %s", uri))
	return nil
}
