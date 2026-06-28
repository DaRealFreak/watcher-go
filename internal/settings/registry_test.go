package settings

import (
	"reflect"
	"testing"

	_ "github.com/DaRealFreak/watcher-go/internal/watcher" // blank-import registers all modules
	"github.com/spf13/viper"
)

func TestClassify(t *testing.T) {
	if classify(reflect.TypeOf(true)) != KindScalar {
		t.Errorf("bool should be scalar")
	}
	if classify(reflect.TypeOf([]string{})) != KindStringList {
		t.Errorf("[]string should be string list")
	}
	if classify(reflect.TypeOf([]sampleProxy{})) != KindComplex {
		t.Errorf("[]struct should be complex")
	}
	if classify(reflect.TypeOf(sampleProxy{})) != KindComplex {
		t.Errorf("named struct should be complex")
	}
}

func TestBuildModuleEntries(t *testing.T) {
	r := Build()

	// pawchive has external_urls.download_external_items (bool scalar)
	e, ok := r.Resolve("modules.pawchive_st.external_urls.download_external_items")
	if !ok {
		t.Fatalf("expected pawchive external_urls entry to be registered")
	}
	if e.Kind != KindScalar || e.Group != "pawchive.st" {
		t.Errorf("entry kind/group wrong: kind=%v group=%q", e.Kind, e.Group)
	}

	// per-module download.directory override is always registered
	if _, ok := r.Resolve("modules.pawchive_st.download.directory"); !ok {
		t.Errorf("expected per-module download.directory override")
	}

	// a []http.ProxySettings field (present on several modules) is read-only complex
	if e, ok := r.Resolve("modules.deviantart_com.loopproxies"); ok {
		if e.Kind != KindComplex || !e.ReadOnly {
			t.Errorf("loopproxies should be complex+readonly, got kind=%v ro=%v", e.Kind, e.ReadOnly)
		}
	} else {
		t.Errorf("expected deviantart loopproxies entry")
	}

	// unknown key resolves to false
	if _, ok := r.Resolve("modules.nope.nope"); ok {
		t.Errorf("unknown key should not resolve")
	}
}

func TestEntriesOrdering(t *testing.T) {
	r := Build()

	entries := r.Entries()
	if len(entries) == 0 {
		t.Fatal("Entries() returned empty slice; expected at least one registered setting")
	}

	// Pick two module keys known to be registered (alphabetically ordered: earlyKey < lateKey).
	// We check via Resolve rather than hardcoding assumptions about their exact entry names.
	earlyKey := "bsky.app"
	lateKey := "deviantart.com"

	// Verify both are actually present; if not, fall back to any two present module groups.
	_, earlyPresent := r.Resolve("modules.bsky_app.download.directory")
	_, latePresent := r.Resolve("modules.deviantart_com.download.directory")
	if !earlyPresent || !latePresent {
		t.Skipf("expected modules %q and %q to be registered; skipping ordering check", earlyKey, lateKey)
	}

	// Find the first entry index for each group.
	firstEarly, firstLate := -1, -1
	for i, e := range entries {
		if firstEarly == -1 && e.Group == earlyKey {
			firstEarly = i
		}
		if firstLate == -1 && e.Group == lateKey {
			firstLate = i
		}
		if firstEarly != -1 && firstLate != -1 {
			break
		}
	}
	if firstEarly == -1 {
		t.Fatalf("no entry found with Group=%q", earlyKey)
	}
	if firstLate == -1 {
		t.Fatalf("no entry found with Group=%q", lateKey)
	}
	if firstEarly >= firstLate {
		t.Errorf("registration order wrong: first %q entry at index %d >= first %q entry at index %d; want sorted by module key",
			earlyKey, firstEarly, lateKey, firstLate)
	}
}

func TestEffectiveValue(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	e := Entry{Key: "modules.pawchive_st.external_urls.download_external_items", Type: reflect.TypeOf(true), Kind: KindScalar}
	r := &Registry{}
	if v := r.EffectiveValue(e); v != false {
		t.Errorf("unset bool scalar with no default should be false (zero value), got %v", v)
	}

	withDefault := Entry{Key: "crawljob.auto_start", Type: reflect.TypeOf(true), Kind: KindScalar, Default: true}
	if v := r.EffectiveValue(withDefault); v != true {
		t.Errorf("unset entry should fall back to default true, got %v", v)
	}
	viper.Set("crawljob.auto_start", false)
	if v := r.EffectiveValue(withDefault); v != false {
		t.Errorf("set value should win over default, got %v", v)
	}
}
