package pawchive

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMigrationDbIO serves a fixed set of tracked items and records URI rewrites,
// progress updates and deletions. All other DatabaseInterface methods are left
// unimplemented (the embedded nil interface panics if any of them is called).
type mockMigrationDbIO struct {
	models.DatabaseInterface
	items      []*models.TrackedItem
	uriChanges map[int]string
	updates    map[int]string
	deleted    []int
}

func newMockMigrationDbIO(items ...*models.TrackedItem) *mockMigrationDbIO {
	return &mockMigrationDbIO{
		items:      items,
		uriChanges: map[int]string{},
		updates:    map[int]string{},
	}
}

func (d *mockMigrationDbIO) GetTrackedItems(_ models.ModuleInterface, _ bool) []*models.TrackedItem {
	return d.items
}

func (d *mockMigrationDbIO) ChangeTrackedItemUri(trackedItem *models.TrackedItem, uri string) {
	d.uriChanges[trackedItem.ID] = uri
	trackedItem.URI = uri
}

func (d *mockMigrationDbIO) UpdateTrackedItem(trackedItem *models.TrackedItem, currentItem string) {
	d.updates[trackedItem.ID] = currentItem
	trackedItem.CurrentItem = currentItem
}

func (d *mockMigrationDbIO) DeleteTrackedItem(trackedItem *models.TrackedItem) {
	d.deleted = append(d.deleted, trackedItem.ID)
}

func newTestModule(db *mockMigrationDbIO) *pawchive {
	m := NewBareModule().ModuleInterface.(*pawchive)
	m.SetDbIO(db)

	return m
}

func TestNormalizeDomains(t *testing.T) {
	cases := map[string]string{
		"https://pawchive.st/patreon/user/4829343":          "https://pawchive.pw/patreon/user/4829343",
		"https://pawchive.st/patreon/user/48/post/161":      "https://pawchive.pw/patreon/user/48/post/161",
		"https://file.pawchive.st/data/aa/bb/x.png?f=x.png": "https://file.pawchive.pw/data/aa/bb/x.png?f=x.png",
		"https://pawchive.pw/patreon/user/4829343":          "https://pawchive.pw/patreon/user/4829343",
		"https://cdn.example.com/abs.png":                   "https://cdn.example.com/abs.png",
	}
	for uri, want := range cases {
		assert.Equal(t, want, normalizeDomains(uri))
	}
}

// TestAddItemNormalizesLegacyDomain locks in that adding an item via a defunct domain
// persists the canonical domain, so both domains resolve to the same tracked row.
func TestAddItemNormalizesLegacyDomain(t *testing.T) {
	m := newTestModule(newMockMigrationDbIO())

	got, err := m.AddItem("https://pawchive.st/patreon/user/4829343")
	require.NoError(t, err)
	assert.Equal(t, "https://pawchive.pw/patreon/user/4829343", got)
}

// TestAddItemAdoptsLegacyRow is the regression test for the duplicate-row bug: adding or
// running the canonical (or legacy) URI while an old-domain row exists must rewrite that
// row instead of letting the caller create a second one.
func TestAddItemAdoptsLegacyRow(t *testing.T) {
	legacyItem := &models.TrackedItem{
		ID:          1,
		URI:         "https://pawchive.st/patreon/user/4829343",
		CurrentItem: "161164023",
	}
	db := newMockMigrationDbIO(legacyItem)
	m := newTestModule(db)

	got, err := m.AddItem("https://pawchive.pw/patreon/user/4829343")
	require.NoError(t, err)
	assert.Equal(t, "https://pawchive.pw/patreon/user/4829343", got)
	// the legacy row was rewritten in place, keeping its download progress
	assert.Equal(t, "https://pawchive.pw/patreon/user/4829343", db.uriChanges[1])
	assert.Equal(t, "161164023", legacyItem.CurrentItem)
	assert.Empty(t, db.deleted)
}

// TestAddItemPrefersExistingCanonicalRow ensures a legacy row is left untouched when the
// canonical URI is already tracked (the duplicate is merged during Parse instead).
func TestAddItemPrefersExistingCanonicalRow(t *testing.T) {
	canonicalItem := &models.TrackedItem{ID: 1, URI: "https://pawchive.pw/patreon/user/4829343"}
	legacyItem := &models.TrackedItem{ID: 2, URI: "https://pawchive.st/patreon/user/4829343"}
	db := newMockMigrationDbIO(canonicalItem, legacyItem)
	m := newTestModule(db)

	got, err := m.AddItem("https://pawchive.st/patreon/user/4829343")
	require.NoError(t, err)
	assert.Equal(t, "https://pawchive.pw/patreon/user/4829343", got)
	assert.Empty(t, db.uriChanges)
	assert.Empty(t, db.deleted)
}

// TestAddItemIgnoresUnrelatedItems ensures adopting only touches rows that normalize to
// the exact canonical URI being added.
func TestAddItemIgnoresUnrelatedItems(t *testing.T) {
	otherItem := &models.TrackedItem{ID: 1, URI: "https://pawchive.st/patreon/user/1111111"}
	db := newMockMigrationDbIO(otherItem)
	m := newTestModule(db)

	_, err := m.AddItem("https://pawchive.pw/patreon/user/4829343")
	require.NoError(t, err)
	assert.Empty(t, db.uriChanges)
}

func TestMigrateLegacyDomain(t *testing.T) {
	t.Run("canonical item is not touched", func(t *testing.T) {
		item := &models.TrackedItem{ID: 1, URI: "https://pawchive.pw/patreon/user/4829343"}
		db := newMockMigrationDbIO(item)
		m := newTestModule(db)

		assert.Equal(t, domainMigrationNone, m.migrateLegacyDomain(item))
		assert.Empty(t, db.uriChanges)
	})

	t.Run("legacy item is rewritten in place", func(t *testing.T) {
		item := &models.TrackedItem{ID: 1, URI: "https://pawchive.st/patreon/user/4829343"}
		db := newMockMigrationDbIO(item)
		m := newTestModule(db)

		assert.Equal(t, domainMigrationRewritten, m.migrateLegacyDomain(item))
		assert.Equal(t, "https://pawchive.pw/patreon/user/4829343", item.URI)
	})

	t.Run("legacy duplicate is deleted, progress transferred", func(t *testing.T) {
		canonicalItem := &models.TrackedItem{ID: 1, URI: "https://pawchive.pw/patreon/user/4829343"}
		legacyItem := &models.TrackedItem{
			ID:          2,
			URI:         "https://pawchive.st/patreon/user/4829343",
			CurrentItem: "161164023",
		}
		db := newMockMigrationDbIO(canonicalItem, legacyItem)
		m := newTestModule(db)

		assert.Equal(t, domainMigrationSuperseded, m.migrateLegacyDomain(legacyItem))
		assert.Equal(t, []int{2}, db.deleted)
		assert.Equal(t, "161164023", canonicalItem.CurrentItem)
	})

	t.Run("progress of the canonical item is never overwritten", func(t *testing.T) {
		canonicalItem := &models.TrackedItem{
			ID:          1,
			URI:         "https://pawchive.pw/patreon/user/4829343",
			CurrentItem: "170000000",
		}
		legacyItem := &models.TrackedItem{
			ID:          2,
			URI:         "https://pawchive.st/patreon/user/4829343",
			CurrentItem: "161164023",
		}
		db := newMockMigrationDbIO(canonicalItem, legacyItem)
		m := newTestModule(db)

		assert.Equal(t, domainMigrationSuperseded, m.migrateLegacyDomain(legacyItem))
		assert.Equal(t, "170000000", canonicalItem.CurrentItem)
		assert.Empty(t, db.updates)
	})
}

// TestURISchemasMatchBothDomains ensures the module claims URIs of the canonical and all
// legacy domains, so routing keeps working for previously tracked items.
func TestURISchemasMatchBothDomains(t *testing.T) {
	module := NewBareModule()

	for _, uri := range []string{
		"https://pawchive.pw/patreon/user/4829343",
		"https://pawchive.st/patreon/user/4829343",
	} {
		matched := false
		for _, schema := range module.URISchemas {
			if schema.MatchString(uri) {
				matched = true
				break
			}
		}
		assert.True(t, matched, "uri %q should be matched by the module", uri)
	}
}
