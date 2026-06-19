package sankakucomplex

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// maxAliasHops bounds how far the alias chain is followed to avoid infinite loops on
// pathological data.
const maxAliasHops = 5

// tagMigrationResult reports the outcome of an attempted tag migration.
type tagMigrationResult int

const (
	// tagMigrationNone means no tag in the search was aliased; nothing changed.
	tagMigrationNone tagMigrationResult = iota
	// tagMigrationRewritten means the item URI was rewritten in place to the canonical
	// search; the caller should re-parse the item.
	tagMigrationRewritten
	// tagMigrationSuperseded means the canonical search was already tracked by another
	// item, so the stale item was deleted; the caller should stop.
	tagMigrationSuperseded
)

// isQualifierTag reports whether a tag is a search qualifier (e.g. "pool:123",
// "order:date", "rating:safe") rather than a plain tag. Qualifiers must never be sent
// to the tag-and-wiki endpoint or migrated.
func isQualifierTag(tag string) bool {
	return strings.Contains(tag, ":")
}

// splitSearchTags splits a search string into its individual tags on whitespace.
func splitSearchTags(tags string) []string {
	return strings.Fields(tags)
}

// rebuildSearchTags resolves every plain tag in tokens via resolve, leaving qualifier
// tags untouched, and returns the rebuilt space-joined search string plus whether any
// tag changed.
func rebuildSearchTags(tokens []string, resolve func(string) (string, bool)) (rebuilt string, changed bool) {
	resolved := make([]string, 0, len(tokens))

	for _, token := range tokens {
		if isQualifierTag(token) {
			resolved = append(resolved, token)
			continue
		}

		canonical, aliased := resolve(token)
		if aliased {
			changed = true
			resolved = append(resolved, canonical)
		} else {
			resolved = append(resolved, token)
		}
	}

	return strings.Join(resolved, " "), changed
}

// followAliasChain resolves start to its final canonical tag, following alias hops via
// lookup up to maxAliasHops and guarding against cycles. It returns the final tag and
// whether it changed from start. A lookup error aborts without migrating.
func followAliasChain(start string, lookup func(string) (string, bool, error)) (string, bool, error) {
	current := start
	visited := map[string]bool{start: true}

	for hop := 0; hop < maxAliasHops; hop++ {
		next, aliased, err := lookup(current)
		if err != nil {
			return start, false, err
		}

		if !aliased || next == "" || visited[next] {
			break
		}

		current = next
		visited[next] = true
	}

	return current, current != start, nil
}

// migrateTagsInURI returns a copy of uri with its tag search replaced by newTags. It
// handles the query form (?tags=..., /books?tags=...) and the /tags/{tag} overview path
// form.
func migrateTagsInURI(uri, newTags string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	// path form: /tags/{tag}
	if matches := tagOverviewPattern.FindStringSubmatch(parsed.Path); len(matches) >= 2 {
		parsed.Path = strings.Replace(parsed.Path, "/tags/"+matches[1], "/tags/"+url.PathEscape(newTags), 1)
		return parsed.String(), nil
	}

	// query form: ?tags=...
	q := parsed.Query()
	q.Set("tags", newTags)
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
}

// resolveTagAlias resolves a single tag to its canonical form, following the alias
// chain. Qualifier and empty tags are returned unchanged. A lookup failure is returned
// as a non-fatal error alongside the original tag.
func (m *sankakuComplex) resolveTagAlias(tag string) (canonical string, aliased bool, err error) {
	if strings.TrimSpace(tag) == "" || isQualifierTag(tag) {
		return tag, false, nil
	}

	return followAliasChain(tag, func(current string) (string, bool, error) {
		resp, lookupErr := m.api.GetTagAndWiki(current)
		if lookupErr != nil {
			return "", false, lookupErr
		}

		next, isAlias := resp.AliasTarget()

		return next, isAlias, nil
	})
}

// resolveTagAliasLogged adapts resolveTagAlias to the resolver signature expected by
// rebuildSearchTags, swallowing (and logging) lookup errors as "not aliased".
func (m *sankakuComplex) resolveTagAliasLogged(tag string) (string, bool) {
	canonical, aliased, err := m.resolveTagAlias(tag)
	if err != nil {
		slog.Debug(fmt.Sprintf("tag-and-wiki lookup failed for %q: %s", tag, err.Error()), "module", m.Key)

		return tag, false
	}

	return canonical, aliased
}

// canonicalAlreadyTracked reports whether a different tracked item already uses newURI.
// tracked_items.uri has no UNIQUE constraint, so this guards against duplicate rows.
func (m *sankakuComplex) canonicalAlreadyTracked(item *models.TrackedItem, newURI string) bool {
	for _, existing := range m.DbIO.GetTrackedItems(m, true) {
		if existing.URI == newURI && existing.ID != item.ID {
			return true
		}
	}

	return false
}

// applyMigratedURI normalizes newURI through AddItem, applies the duplicate guard, and
// persists the result, returning the migration outcome.
func (m *sankakuComplex) applyMigratedURI(item *models.TrackedItem, newURI string) tagMigrationResult {
	// AddItem normalizes the URI (strips tab/order:popularity). Normalization is
	// best-effort: if it ever fails, fall through with the original newURI rather
	// than blocking the migration.
	if normalized, addErr := m.AddItem(newURI); addErr == nil {
		newURI = normalized
	}

	if newURI == item.URI {
		return tagMigrationNone
	}

	if m.canonicalAlreadyTracked(item, newURI) {
		slog.Info(fmt.Sprintf("canonical uri %q already tracked, removing stale item %q", newURI, item.URI), "module", m.Key)
		m.DbIO.DeleteTrackedItem(item)

		return tagMigrationSuperseded
	}

	slog.Info(fmt.Sprintf("migrating aliased uri %q -> %q", item.URI, newURI), "module", m.Key)
	m.DbIO.ChangeTrackedItemUri(item, newURI)

	return tagMigrationRewritten
}

// migrateAliasedSearch checks whether the tracked item's tag search contains aliased
// tags and, if so, rewrites and persists the item URI to the canonical search.
func (m *sankakuComplex) migrateAliasedSearch(item *models.TrackedItem) (tagMigrationResult, error) {
	currentTags, tagErr := m.extractItemTag(item)
	if tagErr != nil {
		// not a tag search (e.g. a single book) - nothing to migrate
		return tagMigrationNone, nil
	}

	rebuilt, changed := rebuildSearchTags(splitSearchTags(currentTags), m.resolveTagAliasLogged)
	if !changed {
		return tagMigrationNone, nil
	}

	newURI, uriErr := migrateTagsInURI(item.URI, rebuilt)
	if uriErr != nil {
		return tagMigrationNone, uriErr
	}

	return m.applyMigratedURI(item, newURI), nil
}
