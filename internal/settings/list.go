package settings

import "strings"

// AddToList appends v (trimmed) to list if not already present. Returns the
// (possibly unchanged) list and whether it was added.
func AddToList(list []string, v string) ([]string, bool) {
	v = strings.TrimSpace(v)
	for _, x := range list {
		if x == v {
			return list, false
		}
	}
	return append(list, v), true
}

// RemoveFromList removes v (trimmed) from list. Returns a new slice and whether
// anything was removed.
func RemoveFromList(list []string, v string) ([]string, bool) {
	v = strings.TrimSpace(v)
	out := make([]string, 0, len(list))
	removed := false
	for _, x := range list {
		if x == v {
			removed = true
			continue
		}
		out = append(out, x)
	}
	return out, removed
}
