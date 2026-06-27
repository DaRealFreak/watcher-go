package jdownloader

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfigDefaults(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	cfg := LoadConfig()
	if cfg.Enabled {
		t.Errorf("Enabled should default to false")
	}
	if cfg.File != "./watcher-go.crawljob" {
		t.Errorf("File default = %q, want ./watcher-go.crawljob", cfg.File)
	}
	if !cfg.AutoStart {
		t.Errorf("AutoStart should default to true")
	}
	if !cfg.AutoConfirm {
		t.Errorf("AutoConfirm should default to true")
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("crawljob.enabled", true)
	viper.Set("crawljob.file", "/tmp/x.crawljob")
	viper.Set("crawljob.folderwatch_path", "/jd/folderwatch")
	viper.Set("crawljob.auto_start", false)
	viper.Set("crawljob.blacklist", []string{"discord.gg", "patreon.com"})

	cfg := LoadConfig()
	if !cfg.Enabled {
		t.Errorf("Enabled override not applied")
	}
	if cfg.File != "/tmp/x.crawljob" {
		t.Errorf("File = %q", cfg.File)
	}
	if cfg.FolderwatchPath != "/jd/folderwatch" {
		t.Errorf("FolderwatchPath = %q", cfg.FolderwatchPath)
	}
	if cfg.AutoStart {
		t.Errorf("AutoStart override (false) not applied")
	}
	if cfg.AutoConfirm != true {
		t.Errorf("AutoConfirm should still default true when unset")
	}
	if len(cfg.Blacklist) != 2 || cfg.Blacklist[0] != "discord.gg" {
		t.Errorf("Blacklist = %v", cfg.Blacklist)
	}
	if !Enabled() {
		t.Errorf("Enabled() should report true")
	}
}
