// Package modules contains the management implementation of all different modules
package modules

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// nolint: gochecknoglobals
var moduleFactory *ModuleFactory

// ModuleFactory contains all registered modules and URI Schemas
type ModuleFactory struct {
	modules    []*models.Module
	uriSchemas map[string][]*regexp.Regexp
}

// GetModuleFactory returns already generated or else previously generated module factory
func GetModuleFactory() *ModuleFactory {
	if moduleFactory == nil {
		moduleFactory = newModuleFactory()
	}

	return moduleFactory
}

// newModuleFactory returns a module factory with empty uri schema and modules
func newModuleFactory() *ModuleFactory {
	return &ModuleFactory{
		uriSchemas: make(map[string][]*regexp.Regexp),
	}
}

// RegisterModule registers a module and appends the URI schema to the known schemas
func (f *ModuleFactory) RegisterModule(module *models.Module) {
	// register URI schema for module
	module.RegisterURISchema(f.uriSchemas)
	// append module to retrievable modules
	f.modules = append(f.modules, module)
}

// GetAllModules returns all available modules
func (f *ModuleFactory) GetAllModules() []*models.Module {
	return f.modules
}

// GetModule returns a module by it's key
func (f *ModuleFactory) GetModule(moduleName string) *models.Module {
	for _, module := range f.modules {
		if module.Key() == moduleName {
			return module
		}
	}

	return nil
}

// GetModuleFromURI checks the registered URI schemas for a match and returns the module
func (f *ModuleFactory) GetModuleFromURI(uri string) *models.Module {
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

// GetModulesFromURIs returns the selected modules in bulk for urls
func (f *ModuleFactory) GetModulesFromURIs(uri ...string) (modules []*models.Module) {
	for _, moduleURI := range uri {
		modules = append(modules, f.GetModuleFromURI(moduleURI))
	}

	return modules
}

// InitializeAllModules initializes all bare modules
func (f *ModuleFactory) InitializeAllModules() {
	for _, module := range f.modules {
		module.InitializeModule()
	}
}
