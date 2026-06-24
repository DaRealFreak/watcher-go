package api

import "testing"

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
