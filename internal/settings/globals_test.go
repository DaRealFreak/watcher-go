package settings

import (
	"testing"

	_ "github.com/DaRealFreak/watcher-go/internal/watcher" // register modules
	"github.com/spf13/viper"
)

func TestGlobalEntriesPresent(t *testing.T) {
	r := Build()
	for _, key := range []string{"download.directory", "database.path", "watcher.sentry"} {
		if e, ok := r.Resolve(key); !ok || e.Group != "global" {
			t.Errorf("global %q missing or wrong group (ok=%v)", key, ok)
		}
	}
}

func TestCrawljobEntriesAndDefaults(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	r := Build()

	enabled, ok := r.Resolve("crawljob.enabled")
	if !ok || enabled.Group != "crawljob" || enabled.Kind != KindScalar {
		t.Fatalf("crawljob.enabled missing/wrong: ok=%v %+v", ok, enabled)
	}

	bl, ok := r.Resolve("crawljob.blacklist")
	if !ok || bl.Kind != KindStringList {
		t.Errorf("crawljob.blacklist should be a string list, got ok=%v %+v", ok, bl)
	}

	// defaults show through EffectiveValue when unset
	autoStart, _ := r.Resolve("crawljob.auto_start")
	if v := r.EffectiveValue(*autoStart); v != true {
		t.Errorf("crawljob.auto_start default should be true, got %v", v)
	}
	file, _ := r.Resolve("crawljob.file")
	if v := r.EffectiveValue(*file); v != "./watcher-go.crawljob" {
		t.Errorf("crawljob.file default wrong, got %v", v)
	}
}
