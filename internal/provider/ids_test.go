package provider

import "testing"

func TestResourceID(t *testing.T) {
	tests := []struct {
		kind, project, dataset, object, want string
	}{
		{"routines", "proj", "ds", "fn", "projects/proj/datasets/ds/routines/fn"},
		{"routines", "", "ds", "fn", "projects/any/datasets/ds/routines/fn"},
		{"routines", "", "", "fn", "projects/any/datasets/any/routines/fn"},
		{"tables", "proj", "ds", "v", "projects/proj/datasets/ds/tables/v"},
		{"tables", "", "ds", "v", "projects/any/datasets/ds/tables/v"},
		{"tables", "", "", "v", "projects/any/datasets/any/tables/v"},
	}
	for _, tt := range tests {
		got := resourceID(tt.kind, tt.project, tt.dataset, tt.object)
		if got != tt.want {
			t.Errorf("resourceID(%q,%q,%q,%q) = %q, want %q",
				tt.kind, tt.project, tt.dataset, tt.object, got, tt.want)
		}
	}
}
