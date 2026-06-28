package settings

import (
	"reflect"
	"testing"
)

type sampleProxy struct { // simulates a named struct field (like http.ProxySettings)
	Host string `mapstructure:"host"`
}

type sampleSchema struct {
	Loop        bool   `mapstructure:"loop"`
	RateLimit   *int   `mapstructure:"rate_limit"`
	Group       struct { // anonymous inline grouping -> recurse
		Format    string   `mapstructure:"format"`
		Blacklist []string `mapstructure:"blacklisted_tags"`
	} `mapstructure:"search"`
	Proxy       sampleProxy   `mapstructure:"proxy"`        // named struct -> leaf
	LoopProxies []sampleProxy `mapstructure:"loopproxies"`  // []struct -> leaf
	Untagged    string        // no mapstructure tag -> skipped
}

func findField(fields []fieldInfo, path string) (fieldInfo, bool) {
	for _, f := range fields {
		if f.Path == path {
			return f, true
		}
	}
	return fieldInfo{}, false
}

func TestWalkSchema(t *testing.T) {
	fields := walkSchema(sampleSchema{})

	if _, ok := findField(fields, "loop"); !ok {
		t.Errorf("expected leaf 'loop'")
	}
	// pointer unwrapped to int
	if f, ok := findField(fields, "rate_limit"); !ok || f.Type.Kind() != reflect.Int {
		t.Errorf("rate_limit should be a leaf of kind int, got %+v ok=%v", f, ok)
	}
	// anonymous grouping recursed with dotted prefix
	if _, ok := findField(fields, "search.format"); !ok {
		t.Errorf("expected recursed leaf 'search.format'")
	}
	if f, ok := findField(fields, "search.blacklisted_tags"); !ok || f.Type != reflect.TypeOf([]string{}) {
		t.Errorf("blacklisted_tags should be []string leaf, got %+v ok=%v", f, ok)
	}
	// named struct is a leaf (NOT recursed into proxy.host)
	if _, ok := findField(fields, "proxy.host"); ok {
		t.Errorf("named struct field must not be recursed (proxy.host should not exist)")
	}
	if f, ok := findField(fields, "proxy"); !ok || f.Type.Kind() != reflect.Struct {
		t.Errorf("proxy should be a struct leaf, got %+v ok=%v", f, ok)
	}
	// []struct is a leaf
	if f, ok := findField(fields, "loopproxies"); !ok || f.Type.Kind() != reflect.Slice {
		t.Errorf("loopproxies should be a slice leaf, got %+v ok=%v", f, ok)
	}
	// untagged field skipped
	if _, ok := findField(fields, "Untagged"); ok {
		t.Errorf("untagged field must be skipped")
	}
}

func TestParseValue(t *testing.T) {
	if v, err := ParseValue("true", reflect.TypeOf(true)); err != nil || v != true {
		t.Errorf("bool parse: v=%v err=%v", v, err)
	}
	if v, err := ParseValue("hello", reflect.TypeOf("")); err != nil || v != "hello" {
		t.Errorf("string parse: v=%v err=%v", v, err)
	}
	if v, err := ParseValue("42", reflect.TypeOf(0)); err != nil || v.(int64) != 42 {
		t.Errorf("int parse: v=%v err=%v", v, err)
	}
	if v, err := ParseValue("a,b,c", reflect.TypeOf([]string{})); err != nil || len(v.([]string)) != 3 {
		t.Errorf("[]string parse: v=%v err=%v", v, err)
	}
	if _, err := ParseValue("notabool", reflect.TypeOf(true)); err == nil {
		t.Errorf("expected error for bad bool")
	}
	if _, err := ParseValue("x", reflect.TypeOf(sampleProxy{})); err == nil {
		t.Errorf("expected error for struct type")
	}
}
