package template

import "regexp"

type ModuleInterface interface {
	Key() (key string)
	RegisterUriSchema(map[string][]*regexp.Regexp)
	Login(user string, password string)
	Parse(uri string, currentItem string)
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
