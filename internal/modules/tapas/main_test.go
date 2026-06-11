package tapas

import "testing"

// matchesAnySchema reports whether any of the module's registered URI schemas
// match the passed value.
func matchesAnySchema(value string) bool {
	for _, schema := range NewBareModule().URISchemas {
		if schema.MatchString(value) {
			return true
		}
	}

	return false
}

// TestURISchemaMatchesBareDomain locks in the contract that the module can be
// selected by its bare domain. The `-u`/`--url` run flag passes a domain (e.g.
// "tapas.io") which the module factory resolves against these schemas; the
// schemas previously required a /series/ or /episode/ path, so `-u tapas.io`
// failed with "no module is registered which can parse based on the url".
func TestURISchemaMatchesBareDomain(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"bare domain selects module", "tapas.io"},
		{"full series url", "https://tapas.io/series/310047/info"},
		{"full episode url", "https://tapas.io/episode/123456"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !matchesAnySchema(c.value) {
				t.Fatalf("expected a URI schema to match %q, none did", c.value)
			}
		})
	}
}
