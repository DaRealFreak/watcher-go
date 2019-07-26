package modules

import (
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
