// Package modules contains the management implementation of all different modules
package modules

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
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

// GetModule returns a module by its key
func (f *ModuleFactory) GetModule(moduleName string) *models.Module {
	for _, module := range f.modules {
		if module.Key == moduleName {
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

// CanParse can be used to check if f.e. a found URL can be parsed
func (f *ModuleFactory) CanParse(uri string) bool {
	for _, patternCollection := range f.uriSchemas {
		for _, pattern := range patternCollection {
			if pattern.MatchString(uri) {
				return true
			}
		}
	}

	return false
}

// GetModulesFromURIs returns the selected modules in bulk for urls
func (f *ModuleFactory) GetModulesFromURIs(uri ...string) (modules []*models.Module) {
	for _, moduleURI := range uri {
		modules = append(modules, f.GetModuleFromURI(moduleURI))
	}

	return modules
}

// IsModuleIncluded checks if the passed module is in the list of enabled URIs
func (f *ModuleFactory) IsModuleIncluded(module *models.Module, enabledURIs []string) bool {
	if len(enabledURIs) > 0 {
		usedModules := f.GetModulesFromURIs(enabledURIs...)
		for _, usedModule := range usedModules {
			if module.Key == usedModule.Key {
				return true
			}
		}
		// module didn't get found
		return false
	}
	// default return value is true if no module got specified
	return true
}

// IsModuleExcluded checks if the passed module is in the list of disabled URIs
func (f *ModuleFactory) IsModuleExcluded(module *models.Module, disabledURIs []string) bool {
	if len(disabledURIs) > 0 {
		disabledModules := f.GetModulesFromURIs(disabledURIs...)
		for _, disabledModule := range disabledModules {
			if module.Key == disabledModule.Key {
				return true
			}
		}
		// module isn't excluded
		return false
	}
	// no modules got explicitly disabled
	return false
}
