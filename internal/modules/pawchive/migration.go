package pawchive

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// canonicalDomain is the domain the site is currently reachable on. legacyDomains lists
// every previous (defunct) domain we still recognize, so tracked items and user input
// referencing an old domain keep resolving to the same tracked row after a domain move.
const canonicalDomain = "pawchive.pw"

// nolint: gochecknoglobals
var legacyDomains = []string{"pawchive.st"}

// domainMigrationResult reports the outcome of an attempted domain migration.
type domainMigrationResult int

const (
	// domainMigrationNone means the item already uses the canonical domain.
	domainMigrationNone domainMigrationResult = iota
	// domainMigrationRewritten means the item URI was rewritten in place to the
	// canonical domain; the caller can keep processing the item.
	domainMigrationRewritten
	// domainMigrationSuperseded means the canonical URI was already tracked by another
	// item, so the legacy duplicate was removed; the caller should stop processing it.
	domainMigrationSuperseded
)

// normalizeDomains replaces every known legacy domain in uri with the canonical domain.
// The plain string replacement intentionally covers subdomains (file./img.) as well.
func normalizeDomains(uri string) string {
	for _, legacy := range legacyDomains {
		uri = strings.ReplaceAll(uri, legacy, canonicalDomain)
	}

	return uri
}

// AddItem normalizes legacy domains to the canonical domain before the URI is looked up
// or persisted, and adopts an already tracked legacy row for the canonical URI, so adding
// or running the same page on a different domain falls back to the existing item (keeping
// its download progress) instead of creating a second row.
func (m *pawchive) AddItem(uri string) (string, error) {
	canonicalURI := normalizeDomains(uri)
	if m.DbIO != nil {
		m.adoptLegacyTrackedItem(canonicalURI)
	}

	return canonicalURI, nil
}

// adoptLegacyTrackedItem rewrites a tracked item still using a legacy domain of the passed
// canonical URI to the canonical URI itself. If a canonical row already exists nothing is
// changed (leftover legacy duplicates are merged during Parse).
func (m *pawchive) adoptLegacyTrackedItem(canonicalURI string) {
	items := m.DbIO.GetTrackedItems(m, true)
	for _, existing := range items {
		if existing.URI == canonicalURI {
			return
		}
	}

	for _, existing := range items {
		if normalizeDomains(existing.URI) == canonicalURI {
			slog.Info(
				fmt.Sprintf("adopting legacy domain item %q as %q", existing.URI, canonicalURI),
				"module", m.Key,
			)
			m.DbIO.ChangeTrackedItemUri(existing, canonicalURI)

			return
		}
	}
}

// migrateLegacyDomain rewrites a tracked item using a legacy domain to the canonical
// domain. tracked_items.uri has no UNIQUE constraint, so if another item already tracks
// the canonical URI the legacy duplicate is deleted instead, transferring its download
// progress if the canonical item has none yet.
func (m *pawchive) migrateLegacyDomain(item *models.TrackedItem) domainMigrationResult {
	canonicalURI := normalizeDomains(item.URI)
	if canonicalURI == item.URI {
		return domainMigrationNone
	}

	for _, existing := range m.DbIO.GetTrackedItems(m, true) {
		if existing.URI == canonicalURI && existing.ID != item.ID {
			if existing.CurrentItem == "" && item.CurrentItem != "" {
				m.DbIO.UpdateTrackedItem(existing, item.CurrentItem)
			}

			slog.Info(
				fmt.Sprintf("canonical uri %q already tracked, removing legacy item %q", canonicalURI, item.URI),
				"module", m.Key,
			)
			m.DbIO.DeleteTrackedItem(item)

			return domainMigrationSuperseded
		}
	}

	slog.Info(
		fmt.Sprintf("migrating legacy domain uri %q -> %q", item.URI, canonicalURI),
		"module", m.Key,
	)
	m.DbIO.ChangeTrackedItemUri(item, canonicalURI)

	return domainMigrationRewritten
}
