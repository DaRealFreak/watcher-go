package modules

import (
	"fmt"
	"regexp"
	"watcher-go/modules/sankakucomplex"
	"watcher-go/modules/template"
)

type ModuleFactory struct {
	modules    []*template.Module
	uriSchemas map[string][]*regexp.Regexp
}

// generate new module factory and register modules
func NewModuleFactory() *ModuleFactory {
	factory := ModuleFactory{
		uriSchemas: make(map[string][]*regexp.Regexp),
	}
	factory.modules = append(factory.modules, sankakucomplex.NewModule(factory.uriSchemas))
	return &factory
}

// retrieve all available modules
func (f ModuleFactory) GetAllModules() []*template.Module {
	return f.modules
}

// retrieve module by it's key
func (f ModuleFactory) GetModule(moduleName string) *template.Module {
	for _, module := range f.modules {
		if module.Module.Key() == moduleName {
			return module
		}
	}
	return nil
}

// check the registered uri schemas for a match and return the module
func (f ModuleFactory) GetModuleFromUri(uri string) (*template.Module, error) {
	for key, patternCollection := range f.uriSchemas {
		for _, pattern := range patternCollection {
			if pattern.MatchString(uri) {
				return f.GetModule(key), nil
			}
		}
	}
	return nil, fmt.Errorf("no module is registered which can parse based on the url %s", uri)
}