package deviantart

import (
	"sync"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockProgressDbIO records the sequence of current_item values persisted via
// UpdateTrackedItem. All other DatabaseInterface methods are left unimplemented (the
// embedded nil interface panics if any of them is unexpectedly called).
type mockProgressDbIO struct {
	models.DatabaseInterface
	mu      sync.Mutex
	updates []string
}

func (d *mockProgressDbIO) UpdateTrackedItem(trackedItem *models.TrackedItem, currentItem string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updates = append(d.updates, currentItem)
	trackedItem.CurrentItem = currentItem
}

// TestLastContiguousCompleted locks in the "never skip an unfinished item" rule that
// keeps progress saving safe while items complete out of order across proxies.
func TestLastContiguousCompleted(t *testing.T) {
	cases := []struct {
		name string
		in   []bool
		want int
	}{
		{"empty", nil, -1},
		{"first pending", []bool{false}, -1},
		{"gap at start", []bool{false, true}, -1},
		{"single done", []bool{true}, 0},
		{"stops at first gap", []bool{true, true, false, true}, 1},
		{"all done", []bool{true, true, true}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, lastContiguousCompleted(c.in))
		})
	}
}

// TestMarkCompletedAndPersist is the regression test for the multi-proxy progress bug:
// current_item must be persisted incrementally as items complete (not only once at the
// end of the whole gallery), and must only advance across a contiguous run of completed
// items so an out-of-order completion never skips an item still in flight.
func TestMarkCompletedAndPersist(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)
	db := &mockProgressDbIO{}
	m.SetDbIO(db)

	queue := []downloadQueueItemNAPI{
		{itemID: "100"},
		{itemID: "200"},
		{itemID: "300"},
	}
	completed := make([]bool, len(queue))
	trackedItem := &models.TrackedItem{}

	// the second item finishes first (out of order): nothing may be persisted yet
	// because the first item is still in flight.
	m.markCompletedAndPersist(trackedItem, queue, completed, 1)
	assert.Empty(t, db.updates, "must not advance current_item past an unfinished earlier item")
	assert.Empty(t, trackedItem.CurrentItem)

	// the first item finishes: the contiguous prefix now covers items 0 and 1, so
	// current_item advances to the newer of the two (index 1 -> "200").
	m.markCompletedAndPersist(trackedItem, queue, completed, 0)
	assert.Equal(t, []string{"200"}, db.updates, "progress must be persisted incrementally, not only at the end")
	assert.Equal(t, "200", trackedItem.CurrentItem)

	// the last item finishes: current_item advances to "300".
	m.markCompletedAndPersist(trackedItem, queue, completed, 2)
	assert.Equal(t, []string{"200", "300"}, db.updates)
	assert.Equal(t, "300", trackedItem.CurrentItem)
}
