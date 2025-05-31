package patreon

import (
	"fmt"
	"strings"
	"time"
)

// Time is a normal time.Time wrapper to unmarshal with RFC3339 format
type Time struct {
	time.Time
}

func (pt *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		pt.Time = time.Time{}
		return
	}
	pt.Time, err = time.Parse(time.RFC3339, s)
	return
}

func (pt *Time) MarshalJSON() ([]byte, error) {
	if pt.UnixNano() == (time.Time{}).UnixNano() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", pt.Format(time.RFC3339))), nil
}
