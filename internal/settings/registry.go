package settings

import (
	"reflect"
	"sort"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/spf13/viper"
)

// Kind classifies how a setting is edited.
type Kind int

const (
	KindScalar     Kind = iota // bool/string/int/uint/float — set via `config set`
	KindStringList             // []string — set / list-add / list-remove
	KindComplex                // named struct or []struct — read-only here
)

// Entry is one addressable setting. Key is both the address and the viper key.
type Entry struct {
	Key      string
	Type     reflect.Type
	Kind     Kind
	Group    string // "global", "crawljob", or the human module key (for list grouping)
	ReadOnly bool
	Default  any
}

// Registry holds all known settings in a stable order plus a lookup map.
type Registry struct {
	entries []Entry
	byKey   map[string]int
}

// classify maps a leaf type to its editing Kind.
func classify(t reflect.Type) Kind {
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return KindScalar
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return KindStringList
		}
		return KindComplex
	default:
		return KindComplex
	}
}

// Build constructs the registry from the module factory and the global blocks.
func Build() *Registry {
	r := &Registry{byKey: make(map[string]int)}

	// Globals + crawljob first (defined in globals.go).
	for _, e := range globalEntries() {
		r.add(e)
	}
	for _, e := range crawljobEntries() {
		r.add(e)
	}

	// Module entries, sorted by module key for stable output.
	mods := modules.GetModuleFactory().GetAllModules()
	sort.Slice(mods, func(i, j int) bool { return mods[i].Key < mods[j].Key })
	for _, m := range mods {
		prefix := "modules." + m.GetViperModuleKey() + "."
		for _, f := range walkSchema(m.SettingsSchema) {
			k := classify(f.Type)
			r.add(Entry{
				Key:      prefix + f.Path,
				Type:     f.Type,
				Kind:     k,
				Group:    m.Key,
				ReadOnly: k == KindComplex,
			})
		}
		// per-module download.directory override (not part of any schema)
		r.add(Entry{
			Key:   prefix + "download.directory",
			Type:  reflect.TypeOf(""),
			Kind:  KindScalar,
			Group: m.Key,
		})
	}

	return r
}

func (r *Registry) add(e Entry) {
	key := strings.ToLower(e.Key)
	e.Key = key
	if _, exists := r.byKey[key]; exists {
		return // first registration wins; avoids duplicate download.directory etc.
	}
	r.entries = append(r.entries, e)
	r.byKey[key] = len(r.entries) - 1
}

// Resolve returns the entry for a key (case-insensitive), if known.
func (r *Registry) Resolve(key string) (*Entry, bool) {
	idx, ok := r.byKey[strings.ToLower(key)]
	if !ok {
		return nil, false
	}
	return &r.entries[idx], true
}

// Entries returns all registered settings in registration order.
func (r *Registry) Entries() []Entry {
	return r.entries
}

// EffectiveValue returns the value that will actually be used: the configured
// value if set, otherwise the entry's default.
func (r *Registry) EffectiveValue(e Entry) any {
	if viper.IsSet(e.Key) {
		return viper.Get(e.Key)
	}
	return e.Default
}
