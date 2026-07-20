package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

func TestModelViewIncludesGenerationPlanSummary(t *testing.T) {
	plan := application.GenerationPlan{
		OutputDir:     "/tmp/generated",
		OutputAction:  "replace",
		ForceRequired: true,
		ForceUsed:     true,
		FileCount:     6,
		Files: []application.PlannedFile{
			{Path: "README.md", Action: "replace"},
			{Path: "src/ProductService/Product.cs", Action: "replace"},
			{Path: "src/ProductService/ProductService.Api/ProductEndpoints.cs", Action: "create"},
			{Path: "src/ProductService/ProductService.Domain/Product.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs", Action: "create"},
		},
	}

	view := NewModel(plan).View()

	assertContains(t, view, "Microgen")
	assertContains(t, view, "Output directory: /tmp/generated")
	assertContains(t, view, "Output action: replace")
	assertContains(t, view, "Force: required=yes, used=yes")
	assertContains(t, view, "Files planned: 6")
	assertContains(t, view, "- replace README.md")
	assertContains(t, view, "Showing first 5 of 6 planned files.")
	assertContains(t, view, "Press q, esc, or ctrl+c to quit.")
	if strings.Contains(view, "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs") {
		t.Fatalf("expected file preview to be truncated, got view %q", view)
	}
}

func TestModelUpdateQuitsOnExitKeys(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{name: "q", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "esc", msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "ctrl+c", msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := NewModel(application.GenerationPlan{}).Update(tt.msg)

			if cmd == nil {
				t.Fatal("expected quit command")
			}
		})
	}
}

func TestModelUpdateIgnoresOtherKeys(t *testing.T) {
	_, cmd := NewModel(application.GenerationPlan{}).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if cmd != nil {
		t.Fatal("expected no command")
	}
}

func assertContains(t *testing.T, value, expected string) {
	t.Helper()
	if !strings.Contains(value, expected) {
		t.Fatalf("expected %q to contain %q", value, expected)
	}
}
