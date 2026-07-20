package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

func TestModelViewIncludesGenerationPlanSummary(t *testing.T) {
	plan := application.GenerationPlan{
		Config: application.ConfigSummary{
			SolutionName:        "CommercePlatform",
			SolutionDescription: "Product management.",
			TargetFramework:     "net8.0",
			ServiceCount:        2,
			EntityCount:         3,
			ValueObjectCount:    3,
			ServiceNames:        []string{"ProductService", "OrderService"},
		},
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

	view := NewModel(plan, application.GenerateRequest{}, nil).View()

	assertContains(t, view, "Microgen")
	assertContains(t, view, "Product: CommercePlatform")
	assertContains(t, view, "Description: Product management.")
	assertContains(t, view, "Target framework: net8.0")
	assertContains(t, view, "Services: 2, entities: 3, value objects: 3")
	assertContains(t, view, "Service names: ProductService, OrderService")
	assertContains(t, view, "Output directory: /tmp/generated")
	assertContains(t, view, "Output action: replace")
	assertContains(t, view, "Force: required=yes, used=yes")
	assertContains(t, view, "Files planned: 6")
	assertContains(t, view, "- replace README.md")
	assertContains(t, view, "Showing first 5 of 6 planned files.")
	assertContains(t, view, "Press g to generate files. This writes files to the output directory.")
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
			_, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil).Update(tt.msg)

			if cmd == nil {
				t.Fatal("expected quit command")
			}
		})
	}
}

func TestModelUpdateIgnoresExitKeysWhileGenerating(t *testing.T) {
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
			model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil)
			model.status = statusGenerating

			updated, cmd := model.Update(tt.msg)
			updatedModel := updated.(Model)

			if cmd != nil {
				t.Fatal("expected no quit command while generating")
			}
			if updatedModel.status != statusGenerating {
				t.Fatalf("expected generating status to be preserved, got %v", updatedModel.status)
			}
		})
	}
}

func TestModelUpdateAllowsExitKeysAfterGenerationFinishes(t *testing.T) {
	tests := []struct {
		name      string
		finishMsg generationFinishedMsg
		msg       tea.KeyMsg
	}{
		{name: "success q", finishMsg: generationFinishedMsg{}, msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "success esc", finishMsg: generationFinishedMsg{}, msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "success ctrl+c", finishMsg: generationFinishedMsg{}, msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
		{name: "failure q", finishMsg: generationFinishedMsg{err: errors.New("write failed")}, msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "failure esc", finishMsg: generationFinishedMsg{err: errors.New("write failed")}, msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "failure ctrl+c", finishMsg: generationFinishedMsg{err: errors.New("write failed")}, msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil)
			model.status = statusGenerating
			finished, finishCmd := model.Update(tt.finishMsg)

			if finishCmd != nil {
				t.Fatal("expected no command when generation finishes")
			}

			_, cmd := finished.(Model).Update(tt.msg)

			if cmd == nil {
				t.Fatal("expected quit command after generation finishes")
			}
		})
	}
}

func TestModelUpdateIgnoresOtherKeys(t *testing.T) {
	_, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if cmd != nil {
		t.Fatal("expected no command")
	}
}

func TestModelUpdateStartsGenerationOnConfirmedKey(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	model := NewModel(application.GenerationPlan{}, request, func(actual application.GenerateRequest) (application.GenerateResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		return application.GenerateResult{OutputDir: request.OutputDir, Plan: application.GenerationPlan{FileCount: 2}}, nil
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	updatedModel := updated.(Model)

	if updatedModel.status != statusGenerating {
		t.Fatalf("expected generating status, got %v", updatedModel.status)
	}
	if cmd == nil {
		t.Fatal("expected generation command")
	}
	msg := cmd()
	finished, ok := msg.(generationFinishedMsg)
	if !ok {
		t.Fatalf("expected generationFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 2 || finished.result.OutputDir != request.OutputDir {
		t.Fatalf("expected successful generation message, got %#v", finished)
	}

	view := updatedModel.View()
	assertContains(t, view, "Generating files...")
	assertContains(t, view, "Generation is in progress. Exit will be available after it finishes.")
	assertNotContains(t, view, "Press q, esc, or ctrl+c to quit.")
}

func TestModelUpdateIgnoresGenerationKeyWhileGeneratingOrGenerated(t *testing.T) {
	for _, status := range []modelStatus{statusGenerating, statusGenerated} {
		model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerateResult, error) {
			t.Fatal("generation should not run")
			return application.GenerateResult{}, nil
		})
		model.status = status

		_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

		if cmd != nil {
			t.Fatalf("expected no command for status %v", status)
		}
	}
}

func TestModelUpdateRecordsGenerationSuccess(t *testing.T) {
	result := application.GenerateResult{
		OutputDir: "/tmp/generated",
		Warning:   "existing warning",
		Plan:      application.GenerationPlan{OutputDir: "/tmp/generated", FileCount: 3},
	}

	updated, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil).Update(generationFinishedMsg{result: result})
	updatedModel := updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if updatedModel.status != statusGenerated || updatedModel.result.OutputDir != result.OutputDir || updatedModel.plan.FileCount != 3 {
		t.Fatalf("expected generated state, got %#v", updatedModel)
	}
	view := updatedModel.View()
	assertContains(t, view, "Generated 3 files in /tmp/generated.")
	assertContains(t, view, "Warning: existing warning")
}

func TestModelUpdateRecordsGenerationFailureAndAllowsRetry(t *testing.T) {
	generationErr := errors.New("write failed")
	retries := 0
	model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerateResult, error) {
		retries++
		return application.GenerateResult{}, nil
	})

	failed, cmd := model.Update(generationFinishedMsg{err: generationErr})
	failedModel := failed.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if failedModel.status != statusFailed || failedModel.err != generationErr {
		t.Fatalf("expected failed state, got %#v", failedModel)
	}
	view := failedModel.View()
	assertContains(t, view, "Generation failed: write failed")
	assertContains(t, view, "Press g to retry generation. This writes files to the output directory.")

	retrying, retryCmd := failedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if retrying.(Model).status != statusGenerating {
		t.Fatalf("expected retry to enter generating, got %#v", retrying)
	}
	if retryCmd == nil {
		t.Fatal("expected retry command")
	}
	retryCmd()
	if retries != 1 {
		t.Fatalf("expected one retry, got %d", retries)
	}
}

func assertContains(t *testing.T, value, expected string) {
	t.Helper()
	if !strings.Contains(value, expected) {
		t.Fatalf("expected %q to contain %q", value, expected)
	}
}

func assertNotContains(t *testing.T, value, unexpected string) {
	t.Helper()
	if strings.Contains(value, unexpected) {
		t.Fatalf("expected %q not to contain %q", value, unexpected)
	}
}
