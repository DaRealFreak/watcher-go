// Package settings builds a unified registry of every persisted watcher-go
// setting (module schemas + global config blocks) and provides typed parsing
// for the `watcher config` command.
package settings

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// fieldInfo is one settable leaf discovered in a settings schema.
type fieldInfo struct {
	Path string       // dotted mapstructure path, e.g. "search.format"
	Type reflect.Type // pointer-unwrapped leaf type
}

// walkSchema reflects over a settings struct and returns its leaf fields.
// Anonymous inline structs (logical groupings) are recursed; named struct
// fields (e.g. http.ProxySettings) and all slices are treated as leaves;
// pointers are unwrapped. Fields without a usable mapstructure tag are skipped.
func walkSchema(schema any) []fieldInfo {
	var out []fieldInfo
	if schema == nil {
		return out
	}
	walkType(reflect.TypeOf(schema), "", &out)
	return out
}

func walkType(t reflect.Type, prefix string, out *[]fieldInfo) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		// Recurse only into anonymous inline structs (groupings). Named struct
		// types (http.ProxySettings) and slices are leaves.
		if ft.Kind() == reflect.Struct && ft.Name() == "" {
			walkType(ft, key, out)
			continue
		}
		*out = append(*out, fieldInfo{Path: key, Type: ft})
	}
}

// ParseValue converts a string into the target leaf type. Supports scalar
// kinds and []string (comma-split). Other types (structs, non-string slices)
// are rejected — they are read-only in the config command.
func ParseValue(s string, t reflect.Type) (any, error) {
	switch t.Kind() {
	case reflect.String:
		return s, nil
	case reflect.Bool:
		return strconv.ParseBool(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(s, 10, 64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(s, 10, 64)
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(s, 64)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return strings.Split(s, ","), nil
		}
		return nil, fmt.Errorf("unsupported slice type %s", t.Elem().Kind())
	default:
		return nil, fmt.Errorf("unsupported type %s", t.Kind())
	}
}

// FriendlyType returns a human-readable type name for messages and listings.
func FriendlyType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Slice:
		return "[]" + FriendlyType(t.Elem())
	case reflect.Struct:
		return t.Name()
	default:
		return t.String()
	}
}
