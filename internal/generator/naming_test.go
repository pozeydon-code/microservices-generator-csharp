package generator

import "testing"

func TestPluralizeEntityNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "y suffix becomes ies", input: "Category", expected: "Categories"},
		{name: "policy becomes policies", input: "Policy", expected: "Policies"},
		{name: "s suffix adds es", input: "Status", expected: "Statuses"},
		{name: "supported irregular", input: "Person", expected: "People"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := pluralize(tt.input); actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
