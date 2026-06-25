package api

import (
	"fmt"
	"time"
)

// CustomTime parses pawchive timestamps, which appear with fractional seconds
// (comments, e.g. "...:24.117000") and without (posts, e.g. "...:25:00").
type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	s := string(b[1 : len(b)-1]) // strip surrounding quotes
	layouts := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
	}
	var err error
	for _, layout := range layouts {
		ct.Time, err = time.Parse(layout, s)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not parse time: %s, error: %w", s, err)
}

// Profile is the /profile response. pawchive omits kemono's post_count and adds
// fields we don't consume (ever_imported, kemono_favorited, import_size_cap_gb).
type Profile struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Service string     `json:"service"`
	Indexed CustomTime `json:"indexed"`
	Updated CustomTime `json:"updated"`
}

// Post is the flat shape returned by BOTH the /posts list and the /post/{id}
// detail endpoint (pawchive does not wrap detail in {post, attachments, ...}).
type Post struct {
	ID          string       `json:"id"`
	User        string       `json:"user"`
	Service     string       `json:"service"`
	Title       string       `json:"title"`
	Content     string       `json:"content"`
	Embed       Embed        `json:"embed"`
	File        File         `json:"file"`
	Attachments []Attachment `json:"attachments"`
	Published   CustomTime   `json:"published"`
	Added       CustomTime   `json:"added"`
	Edited      CustomTime   `json:"edited"`
	// HasFull is false while the post's full-resolution files have not been
	// archived to the file host yet. pawchive (a kemono successor) then renders a
	// "haven't archived this post yet" CTA and only serves a downscaled thumbnail;
	// fetching the full file 404s. PreviewState is "pending" in that state and
	// "scraped" once imported.
	HasFull      bool   `json:"has_full"`
	PreviewState string `json:"preview_state"`
}

type Embed struct {
	Url         string  `json:"url"`
	Subject     *string `json:"subject"`
	Description *string `json:"description"`
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Attachment carries only name+path; pawchive has no per-file CDN server hint.
type Attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Comment struct {
	ID            string     `json:"id"`
	ParentID      *string    `json:"parent_id"`
	Commenter     string     `json:"commenter"`
	CommenterName string     `json:"commenter_name"`
	Content       string     `json:"content"`
	Published     CustomTime `json:"published"`
	Revisions     []Revision `json:"revisions"`
}

type Revision struct {
	ID      int        `json:"id"`
	Content string     `json:"content"`
	Added   CustomTime `json:"added"`
}
