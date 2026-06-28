package settings

import "testing"

func TestAddToList(t *testing.T) {
	out, added := AddToList([]string{"a"}, "b")
	if !added || len(out) != 2 || out[1] != "b" {
		t.Errorf("add new: out=%v added=%v", out, added)
	}
	out, added = AddToList([]string{"a", "b"}, " b ") // trimmed, already present
	if added || len(out) != 2 {
		t.Errorf("add duplicate (trimmed) should be no-op: out=%v added=%v", out, added)
	}
}

func TestRemoveFromList(t *testing.T) {
	out, removed := RemoveFromList([]string{"a", "b", "c"}, "b")
	if !removed || len(out) != 2 || out[0] != "a" || out[1] != "c" {
		t.Errorf("remove: out=%v removed=%v", out, removed)
	}
	out, removed = RemoveFromList([]string{"a"}, "z")
	if removed || len(out) != 1 {
		t.Errorf("remove absent should be no-op: out=%v removed=%v", out, removed)
	}
}
