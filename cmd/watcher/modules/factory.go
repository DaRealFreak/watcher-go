package modules

import (
	"fmt"
	"regexp"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/models"
	"watcher-go/cmd/watcher/modules/ehentai"
	"watcher-go/cmd/watcher/modules/sankakucomplex"
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
func (f ModuleFactory) GetModuleFromUri(uri string) (*models.Module, error) {
	for key, patternCollection := range f.uriSchemas {
		for _, pattern := range patternCollection {
			if pattern.MatchString(uri) {
				return f.GetModule(key), nil
			}
		}
	}
	return nil, fmt.Errorf("no module is registered which can parse based on the url %s", uri)
}
