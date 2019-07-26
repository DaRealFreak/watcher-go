package template

import (
	"regexp"
	"watcher-go/database"
)

type ModuleInterface interface {
	Key() (key string)
	RegisterUriSchema(map[string][]*regexp.Regexp)
	Login(*database.Account)
	Parse(*database.TrackedItem)
}

type Module struct {
	Module ModuleInterface
}

func (t *Module) GetFileName(uri string) string {
	// ToDo: implement getting the file name
	return uri
}

func (t *Module) GetFileExtension(uri string) string {
	// ToDo: implement getting the file extension
	return uri
}
