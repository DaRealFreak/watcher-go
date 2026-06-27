package jdownloader

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
)

// defaultFile is the local accumulation file used when crawljob.file is unset.
const defaultFile = "./watcher-go.crawljob"

// Config holds the global "crawljob" settings block.
type Config struct {
	Enabled         bool     `mapstructure:"enabled"`
	File            string   `mapstructure:"file"`
	FolderwatchPath string   `mapstructure:"folderwatch_path"`
	Blacklist       []string `mapstructure:"blacklist"`
	AutoStart       bool     `mapstructure:"auto_start"`
	AutoConfirm     bool     `mapstructure:"auto_confirm"`
}

// LoadConfig reads the global "crawljob" config from viper. AutoStart and
// AutoConfirm default to true and File to defaultFile; mapstructure only
// overwrites fields that are actually present in the config, so unset booleans
// keep these defaults.
func LoadConfig() Config {
	cfg := Config{
		File:        defaultFile,
		AutoStart:   true,
		AutoConfirm: true,
	}
	if err := viper.UnmarshalKey("crawljob", &cfg); err != nil {
		slog.Warn(fmt.Sprintf("failed to parse crawljob config: %v", err))
	}
	if cfg.File == "" {
		cfg.File = defaultFile
	}
	return cfg
}

// Enabled reports whether the crawljob handoff is turned on. It reads viper
// directly so module gate checks don't need to build a Writer.
func Enabled() bool {
	return viper.GetBool("crawljob.enabled")
}
