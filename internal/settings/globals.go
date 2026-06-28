package settings

import (
	"reflect"

	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
)

// globalEntries returns the loose top-level scalars manageable via `config`.
func globalEntries() []Entry {
	str := reflect.TypeOf("")
	return []Entry{
		{Key: "download.directory", Type: str, Kind: KindScalar, Group: "global"},
		{Key: "database.path", Type: str, Kind: KindScalar, Group: "global"},
		{Key: "watcher.sentry", Type: reflect.TypeOf(true), Kind: KindScalar, Group: "global"},
	}
}

// crawljobDefaults mirrors the defaults LoadConfig applies (config.go).
var crawljobDefaults = map[string]any{
	"file":         "./watcher-go.crawljob",
	"auto_start":   true,
	"auto_confirm": true,
}

// crawljobEntries reflects over jdownloader.Config to register the crawljob
// block, carrying the same defaults LoadConfig applies.
func crawljobEntries() []Entry {
	var out []Entry
	for _, f := range walkSchema(jdownloader.Config{}) {
		k := classify(f.Type)
		out = append(out, Entry{
			Key:      "crawljob." + f.Path,
			Type:     f.Type,
			Kind:     k,
			Group:    "crawljob",
			ReadOnly: k == KindComplex,
			Default:  crawljobDefaults[f.Path],
		})
	}
	return out
}
