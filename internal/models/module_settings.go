package models

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// settingsEntry represents a single valid setting extracted from a settings struct
type settingsEntry struct {
	Key  string
	Type reflect.Type
}

// AddSettingsCommand adds generic "set", "get", and "settings" subcommands to configure module settings
func (t *Module) AddSettingsCommand(command *cobra.Command) {
	validSettings := t.extractSettings()

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "set a module-specific setting",
		Long:  "set a module-specific setting in the configuration file (e.g. set crt my-token)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			value := args[1]

			if entry, ok := validSettings[key]; ok {
				parsed, err := parseTypedValue(value, entry.Type)
				if err != nil {
					fmt.Printf("invalid value for %s (expected %s): %s\n", key, friendlyTypeName(entry.Type), err)
					return
				}

				viperKey := fmt.Sprintf("Modules.%s.%s", t.GetViperModuleKey(), key)
				viper.Set(viperKey, parsed)
				raven.CheckError(viper.WriteConfig())
				fmt.Printf("set %s = %s\n", key, value)
			} else if len(validSettings) > 0 {
				fmt.Printf("unknown setting %q for module %s\n\navailable settings:\n", key, t.Key)
				printSettings(validSettings)
			} else {
				// no schema registered, allow setting arbitrary keys
				viperKey := fmt.Sprintf("Modules.%s.%s", t.GetViperModuleKey(), key)
				viper.Set(viperKey, parseValue(value))
				raven.CheckError(viper.WriteConfig())
				fmt.Printf("set %s = %s\n", key, value)
			}
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "get a module-specific setting",
		Long:  "get a module-specific setting from the configuration file (e.g. get crt)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			viperKey := fmt.Sprintf("Modules.%s.%s", t.GetViperModuleKey(), key)
			value := viper.Get(viperKey)
			if value == nil {
				fmt.Printf("%s is not set\n", key)
			} else {
				fmt.Printf("%s = %v\n", key, value)
			}
		},
	}

	settingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "list all available settings for this module",
		Run: func(cmd *cobra.Command, args []string) {
			if len(validSettings) == 0 {
				fmt.Printf("no module-specific settings available for %s\n", t.Key)
				return
			}

			fmt.Printf("available settings for %s:\n", t.Key)
			printSettings(validSettings)
		},
	}

	command.AddCommand(setCmd)
	command.AddCommand(getCmd)
	command.AddCommand(settingsCmd)
}

// extractSettings uses reflection to extract valid setting keys and types from the SettingsSchema
func (t *Module) extractSettings() map[string]settingsEntry {
	settings := make(map[string]settingsEntry)

	if t.SettingsSchema == nil {
		return settings
	}

	extractFromType(reflect.TypeOf(t.SettingsSchema), "", settings)

	return settings
}

// extractFromType recursively walks a struct type and collects leaf fields with mapstructure tags
func extractFromType(t reflect.Type, prefix string, settings map[string]settingsEntry) {
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

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			extractFromType(fieldType, key, settings)
		} else {
			settings[key] = settingsEntry{Key: key, Type: fieldType}
		}
	}
}

// parseTypedValue parses a string value into the target type
func parseTypedValue(s string, targetType reflect.Type) (interface{}, error) {
	switch targetType.Kind() {
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
		if targetType.Elem().Kind() == reflect.String {
			return strings.Split(s, ","), nil
		}

		return nil, fmt.Errorf("unsupported slice type %s, edit the config file directly", targetType.Elem().Kind())
	default:
		return nil, fmt.Errorf("unsupported type %s, edit the config file directly", targetType.Kind())
	}
}

// friendlyTypeName returns a human-readable name for a reflect.Type
func friendlyTypeName(t reflect.Type) string {
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
		return "[]" + friendlyTypeName(t.Elem())
	default:
		return t.String()
	}
}

// printSettings prints all available settings with their types
func printSettings(settings map[string]settingsEntry) {
	// collect and sort keys for consistent output
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}

	// simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		entry := settings[key]
		fmt.Printf("  %-40s %s\n", key, friendlyTypeName(entry.Type))
	}
}

// parseValue attempts to convert the string value to the appropriate Go type (used when no schema is available)
func parseValue(s string) interface{} {
	if strings.EqualFold(s, "true") {
		return true
	}

	if strings.EqualFold(s, "false") {
		return false
	}

	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}
