package models

import (
	"regexp"
)

type ModuleInterface interface {
	Key() (key string)
	IsLoggedIn() (loggedIn bool)
	RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp)
	Login(account *Account) (success bool)
	Parse(item *TrackedItem)
}

type Module struct {
	ModuleInterface
}

func (t *Module) GetFileName(uri string) string {
	// ToDo: implement getting the file name
	return uri
}

func (t *Module) GetFileExtension(uri string) string {
	// ToDo: implement getting the file extension
	return uri
}
