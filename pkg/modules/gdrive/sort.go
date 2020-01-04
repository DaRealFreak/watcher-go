package gdrive

import (
	"google.golang.org/api/drive/v3"
	"time"
)

// ByModifiedTime implements the Sort interface of the sort package
type ByModifiedTime []*drive.File

// Len returns the length of all items
func (a ByModifiedTime) Len() int {
	return len(a)
}

// Less compares the modified timestamp and returns if the ith item is less than the jth item
func (a ByModifiedTime) Less(i int, j int) bool {
	iTime, _ := time.Parse(time.RFC3339, a[i].ModifiedTime)
	jTime, _ := time.Parse(time.RFC3339, a[j].ModifiedTime)

	return iTime.Before(jTime)
}

// Swap swaps the passed indexes to apply the sorting
func (a ByModifiedTime) Swap(i int, j int) {
	a[i], a[j] = a[j], a[i]
}
