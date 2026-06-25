package api

import (
	"encoding/json"
	"testing"
)

func TestCustomTimeUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"no fractional seconds", `"2026-06-15T17:25:00"`, false},
		{"fractional seconds", `"2026-06-15T17:29:24.117000"`, false},
		{"null", `null`, false},
		{"garbage", `"not-a-time"`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ct CustomTime
			err := ct.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tc.input, err)
			}
		})
	}
}

// TestPostUnmarshalArchiveState locks in that the post detail/list responses
// expose the "not archived yet" state: un-archived posts carry has_full=false /
// preview_state="pending" (the website renders a "haven't archived this post
// yet" CTA), archived posts carry has_full=true.
func TestPostUnmarshalArchiveState(t *testing.T) {
	notArchived := `{"id":"45474459","title":"Isaka","has_full":false,"preview_state":"pending",` +
		`"file":{"name":"Isaka 0.jpg","path":"/b3/9d/abc.jpg"}}`
	var p Post
	if err := json.Unmarshal([]byte(notArchived), &p); err != nil {
		t.Fatalf("unmarshal un-archived post: %v", err)
	}
	if p.HasFull {
		t.Errorf("HasFull = true, want false for un-archived post")
	}
	if p.PreviewState != "pending" {
		t.Errorf("PreviewState = %q, want %q", p.PreviewState, "pending")
	}

	archived := `{"id":"160866428","has_full":true,"preview_state":"scraped"}`
	var p2 Post
	if err := json.Unmarshal([]byte(archived), &p2); err != nil {
		t.Fatalf("unmarshal archived post: %v", err)
	}
	if !p2.HasFull {
		t.Errorf("HasFull = false, want true for archived post")
	}
}
