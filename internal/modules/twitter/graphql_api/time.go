package graphql_api

import (
	"fmt"
	"strings"
	"time"
)

// TwitterTime is a normal time.Time wrapper to unmarshal with RubyDate format
type TwitterTime struct {
	time.Time
}

func (tt *TwitterTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		tt.Time = time.Time{}
		return
	}
	tt.Time, err = time.Parse(time.RubyDate, s)
	return
}

func (tt *TwitterTime) MarshalJSON() ([]byte, error) {
	if tt.UnixNano() == (time.Time{}).UnixNano() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", tt.Format(time.RubyDate))), nil
}
