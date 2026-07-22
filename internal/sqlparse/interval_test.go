package sqlparse

import "testing"

func TestNormalizeMaxStaleness(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"INTERVAL 90 MINUTE", "0-0 0 1:30:0"},
		{"INTERVAL 10 MINUTE", "0-0 0 0:10:0"},
		{"INTERVAL 4 HOUR", "0-0 0 4:0:0"},
		{`INTERVAL "4:0:0" HOUR TO SECOND`, "0-0 0 4:0:0"},
		{"0-0 0 1:30:0", "0-0 0 1:30:0"},
		{"INTERVAL 2 DAY", "0-0 2 0:0:0"},
		{"INTERVAL 3600 SECOND", "0-0 0 1:0:0"},
		{"INTERVAL 1 YEAR", "1-0 0 0:0:0"},
		{"INTERVAL 3 MONTH", "0-3 0 0:0:0"},
		{"  interval 90 minute  ", "0-0 0 1:30:0"},
		{`INTERVAL '4:0:0' HOUR TO SECOND`, "0-0 0 4:0:0"},
		{"not an interval", "not an interval"},
	}
	for _, tt := range tests {
		got := normalizeMaxStaleness(tt.in)
		if got != tt.want {
			t.Errorf("normalizeMaxStaleness(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
