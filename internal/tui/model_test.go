package tui

import (
	"errors"
	"fmt"
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
			SolutionFormat:      "sln",
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

	view := NewModel(plan, application.GenerateRequest{}, nil, nil, nil).View()

	assertContains(t, view, "Microgen")
	assertContains(t, view, "Product: CommercePlatform")
	assertContains(t, view, "Description: Product management.")
	assertContains(t, view, "Target framework: net8.0")
	assertContains(t, view, "Solution format: .sln")
	assertContains(t, view, "Services: 2, entities: 3, value objects: 3")
	assertContains(t, view, "Service names: ProductService, OrderService")
	assertContains(t, view, "Output directory: /tmp/generated")
	assertContains(t, view, "Output action: replace")
	assertContains(t, view, "Force: required=yes, used=yes")
	assertContains(t, view, "Files planned: 6")
	assertContains(t, view, "Impact: create=4, replace=2 (mixed actions)")
	assertContains(t, view, "Files 1-5 of 6 (filter: all)")
	assertContains(t, view, "Selected: 1/6 replace README.md")
	assertContains(t, view, "> [1/6] replace README.md")
	assertContains(t, view, "  [5/6] create tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs")
	assertContains(t, view, "Press r to refresh the plan or g to generate files. Generation writes files to the output directory.")
	assertContains(t, view, readyHelp)
	assertContains(t, view, "Press q, esc, or ctrl+c to quit.")
	if strings.Contains(view, "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs") {
		t.Fatalf("expected file preview to be truncated, got view %q", view)
	}
}

func TestModelViewShowsPlannedFileRangeAndCursor(t *testing.T) {
	view := NewModel(plannedFilesPlan(6), application.GenerateRequest{}, nil, nil, nil).View()

	assertContains(t, view, "Files 1-5 of 6 (filter: all)")
	assertContains(t, view, "Selected: 1/6 create file-01.txt")
	assertContains(t, view, "> [1/6] create file-01.txt")
	assertContains(t, view, "  [5/6] create file-05.txt")
	assertNotContains(t, view, "file-06.txt")
}

func TestModelUpdateMovesPlannedFileCursorAndWindow(t *testing.T) {
	model := NewModel(plannedFilesPlan(7), application.GenerateRequest{}, nil, nil, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	assertContains(t, model.View(), "Files 1-5 of 7 (filter: all)")
	assertContains(t, model.View(), "> [2/7] create file-02.txt")

	for range 4 {
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		model = updated.(Model)
		if cmd != nil {
			t.Fatal("expected no command")
		}
	}
	view := model.View()
	assertContains(t, view, "Files 2-6 of 7 (filter: all)")
	assertContains(t, view, "> [6/7] create file-06.txt")
	assertNotContains(t, view, "file-01.txt")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 2-6 of 7 (filter: all)")
	assertContains(t, view, "> [5/7] create file-05.txt")
}

func TestModelUpdateClampsPlannedFileNavigationBounds(t *testing.T) {
	model := NewModel(plannedFilesPlan(3), application.GenerateRequest{}, nil, nil, nil)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Files 1-3 of 3 (filter: all)")
	assertContains(t, view, "> [1/3] create file-01.txt")

	for range 5 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	view = model.View()
	assertContains(t, view, "Files 1-3 of 3 (filter: all)")
	assertContains(t, view, "> [3/3] create file-03.txt")
}

func TestModelUpdateSupportsPlannedFileHomeEndAndPageKeys(t *testing.T) {
	model := NewModel(plannedFilesPlan(12), application.GenerateRequest{}, nil, nil, nil)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Files 2-6 of 12 (filter: all)")
	assertContains(t, view, "> [6/12] create file-06.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-5 of 12 (filter: all)")
	assertContains(t, view, "> [1/12] create file-01.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 8-12 of 12 (filter: all)")
	assertContains(t, view, "> [12/12] create file-12.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-5 of 12 (filter: all)")
	assertContains(t, view, "> [1/12] create file-01.txt")
}

func TestModelUpdateWindowSizeChangesVisibleFileRange(t *testing.T) {
	model := NewModel(plannedFilesPlan(20), application.GenerateRequest{}, nil, nil, nil)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 1-6 of 20 (filter: all)")
	assertContains(t, view, "  [6/20] create file-06.txt")
	assertNotContains(t, view, "file-07.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 1-12 of 20 (filter: all)")
	assertContains(t, view, "  [12/20] create file-12.txt")
	assertNotContains(t, view, "file-13.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 1-3 of 20 (filter: all)")
	assertContains(t, view, "  [3/20] create file-03.txt")
	assertNotContains(t, view, "file-04.txt")
}

func TestModelUpdateClampsNavigationAfterResize(t *testing.T) {
	model := NewModel(plannedFilesPlan(20), application.GenerateRequest{}, nil, nil, nil)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	assertContains(t, model.View(), "Files 9-20 of 20 (filter: all)")
	assertContains(t, model.View(), "> [20/20] create file-20.txt")

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 18-20 of 20 (filter: all)")
	assertContains(t, view, "> [20/20] create file-20.txt")
	assertNotContains(t, view, "file-17.txt")
}

func TestModelViewShowsImpactSummaryInDeterministicActionOrder(t *testing.T) {
	plan := application.GenerationPlan{
		FileCount: 5,
		Files: []application.PlannedFile{
			{Path: "replace-1.txt", Action: "replace"},
			{Path: "create-1.txt", Action: "create"},
			{Path: "unchanged-1.txt", Action: "unchanged"},
			{Path: "create-2.txt", Action: "create"},
			{Path: "replace-2.txt", Action: "replace"},
		},
	}

	view := NewModel(plan, application.GenerateRequest{}, nil, nil, nil).View()

	assertContains(t, view, "Files planned: 5")
	assertContains(t, view, "Impact: create=2, replace=2, unchanged=1 (mixed actions)")
}

func TestModelUpdateCyclesActionFilterAndNavigatesFilteredFiles(t *testing.T) {
	plan := application.GenerationPlan{
		FileCount: 5,
		Files: []application.PlannedFile{
			{Path: "replace-1.txt", Action: "replace"},
			{Path: "create-1.txt", Action: "create"},
			{Path: "replace-2.txt", Action: "replace"},
			{Path: "create-2.txt", Action: "create"},
			{Path: "unchanged-1.txt", Action: "unchanged"},
		},
	}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 1-2 of 2 (filter: create)")
	assertContains(t, view, "Selected: 1/2 create create-1.txt")
	assertContains(t, view, "> [1/2] create create-1.txt")
	assertNotContains(t, view, "replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Selected: 2/2 create create-2.txt")
	assertContains(t, view, "> [2/2] create create-2.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-2 of 2 (filter: replace)")
	assertContains(t, view, "Selected: 1/2 replace replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-1 of 1 (filter: unchanged)")
	assertContains(t, view, "Selected: 1/1 unchanged unchanged-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	assertContains(t, model.View(), "Files 1-5 of 5 (filter: all)")
}

func TestModelUpdateIgnoresActionFilterWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := NewModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
			updatedModel := updated.(Model)

			if cmd != nil {
				t.Fatal("expected no command")
			}
			if updatedModel.actionFilter != "" {
				t.Fatalf("expected action filter to stay unset while busy, got %q", updatedModel.actionFilter)
			}
		})
	}
}

func TestModelUpdateIgnoresPlannedFileNavigationWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := NewModel(plannedFilesPlan(6), application.GenerateRequest{}, nil, nil, nil)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
			updatedModel := updated.(Model)

			if cmd != nil {
				t.Fatal("expected no command")
			}
			if updatedModel.fileCursor != 0 || updatedModel.fileOffset != 0 {
				t.Fatalf("expected navigation to be ignored while busy, got cursor=%d offset=%d", updatedModel.fileCursor, updatedModel.fileOffset)
			}
		})
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
			_, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(tt.msg)

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
			model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
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

func TestModelUpdateAllowsExitKeysWhileRefreshing(t *testing.T) {
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
			model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
			model.status = statusRefreshing

			_, cmd := model.Update(tt.msg)

			if cmd == nil {
				t.Fatal("expected quit command while refreshing")
			}
		})
	}
}

func TestModelViewShowsRefreshWaitHelpOnly(t *testing.T) {
	model := NewModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
	model.status = statusRefreshing

	view := model.View()

	assertContains(t, view, "Refreshing plan...")
	assertContains(t, view, "Please wait while the read-only plan refresh finishes. Press q, esc, or ctrl+c to quit.")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, generatedHelp)
	assertNotContains(t, view, "Press r to refresh the plan")
	assertNotContains(t, view, "g to generate")
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
			model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
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
	_, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if cmd != nil {
		t.Fatal("expected no command")
	}
}

func TestModelUpdateStartsGenerationOnConfirmedKey(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	model := NewModel(application.GenerationPlan{}, request, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		return application.GenerateResult{OutputDir: request.OutputDir, Plan: application.GenerationPlan{FileCount: 2}}, nil
	}, nil)

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
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "Press q, esc, or ctrl+c to quit.")
}

func TestModelUpdateStartsRefreshOnRefreshKey(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	model := NewModel(application.GenerationPlan{}, request, func(actual application.GenerateRequest) (application.GenerationPlan, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		return plannedFilesPlan(2), nil
	}, nil, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	updatedModel := updated.(Model)

	if updatedModel.status != statusRefreshing {
		t.Fatalf("expected refreshing status, got %v", updatedModel.status)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	msg := cmd()
	finished, ok := msg.(planFinishedMsg)
	if !ok {
		t.Fatalf("expected planFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.plan.FileCount != 2 {
		t.Fatalf("expected successful refresh message, got %#v", finished)
	}
	view := updatedModel.View()
	assertContains(t, view, "Refreshing plan...")
	assertContains(t, view, "Please wait while the read-only plan refresh finishes. Press q, esc, or ctrl+c to quit.")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "g to generate")
}

func TestModelUpdateRecordsRefreshSuccess(t *testing.T) {
	model := NewModel(plannedFilesPlan(10), application.GenerateRequest{}, nil, nil, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)

	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{SolutionName: "Refreshed", TargetFramework: "net9.0", ServiceCount: 1}
	plan.OutputDir = "/tmp/refreshed"
	plan.OutputAction = "replace"
	updated, cmd := model.Update(planFinishedMsg{plan: plan})
	updatedModel := updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if updatedModel.status != statusReady || updatedModel.plan.OutputDir != "/tmp/refreshed" || updatedModel.plan.Config.SolutionName != "Refreshed" {
		t.Fatalf("expected refreshed ready state, got %#v", updatedModel)
	}
	view := updatedModel.View()
	assertContains(t, view, "Product: Refreshed")
	assertContains(t, view, "Target framework: net9.0")
	assertContains(t, view, "Output directory: /tmp/refreshed")
	assertContains(t, view, "Files 1-2 of 2 (filter: all)")
	assertContains(t, view, "> [2/2] create file-02.txt")
}

func TestModelUpdateRecordsRefreshFailureAndAllowsRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return application.GenerationPlan{}, nil
	}, nil, nil)

	failed, cmd := model.Update(planFinishedMsg{err: refreshErr})
	failedModel := failed.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if failedModel.status != statusFailed || failedModel.err != refreshErr || failedModel.errContext != "Refresh" {
		t.Fatalf("expected refresh failed state, got %#v", failedModel)
	}
	view := failedModel.View()
	assertContains(t, view, "Refresh failed: plan failed")
	assertContains(t, view, "Press r to refresh the plan or g to retry generation.")

	retrying, retryCmd := failedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if retrying.(Model).status != statusRefreshing {
		t.Fatalf("expected retry to enter refreshing, got %#v", retrying)
	}
	if retryCmd == nil {
		t.Fatal("expected retry command")
	}
	retryCmd()
	if retries != 1 {
		t.Fatalf("expected one retry, got %d", retries)
	}
}

func TestModelUpdateIgnoresGenerationKeyWhileGeneratingOrGenerated(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusGenerated} {
		model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
			t.Fatal("generation should not run")
			return application.GenerateResult{}, nil
		}, nil)
		model.status = status

		_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

		if cmd != nil {
			t.Fatalf("expected no command for status %v", status)
		}
	}
}

func TestModelUpdateIgnoresRefreshKeyWhileRefreshingOrGenerating(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating} {
		model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
			t.Fatal("refresh should not run")
			return application.GenerationPlan{}, nil
		}, nil, nil)
		model.status = status

		_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

		if cmd != nil {
			t.Fatalf("expected no command for status %v", status)
		}
	}
}

func TestModelUpdateEditsSolutionSettingsAndSaves(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", SolutionDescription: "Old description", TargetFramework: "net8.0"}
	updatedPlan := plannedFilesPlan(3)
	updatedPlan.Config = application.ConfigSummary{SolutionName: "CatalogPlatform", SolutionDescription: "New description", TargetFramework: "net9.0"}
	var capturedSettings application.SolutionSettings
	model := NewModel(plan, request, nil, nil, func(actual application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		capturedSettings = settings
		return application.UpdateSolutionSettingsResult{Saved: true, Plan: updatedPlan}, nil
	}, []string{"net10.0", "net9.0", "net8.0"})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command when entering edit mode")
	}
	if model.status != statusEditing || model.edit.focused != editFieldName {
		t.Fatalf("expected editing name field, got %#v", model)
	}
	assertContains(t, model.View(), "Editing solution settings")
	assertContains(t, model.View(), "Service, entity, field, and value-object editing is not available yet.")

	for range len([]rune("CommercePlatform")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("CatalogPlatform")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if model.edit.focused != editFieldDescription {
		t.Fatalf("expected description field, got %v", model.edit.focused)
	}
	for range len([]rune("Old description")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("New description")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.edit.focused != editFieldTargetFramework {
		t.Fatalf("expected target framework field, got %v", model.edit.focused)
	}
	for range len([]rune("net8.0")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	model = updated.(Model)
	if model.edit.targetFramework.string() != "9" {
		t.Fatalf("expected manual target framework entry, got %q", model.edit.targetFramework.string())
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving {
		t.Fatalf("expected saving state, got %v", model.status)
	}
	if cmd == nil {
		t.Fatal("expected save command")
	}
	assertContains(t, model.View(), "Saving settings...")
	msg := cmd()
	finished, ok := msg.(settingsFinishedMsg)
	if !ok {
		t.Fatalf("expected settingsFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 3 {
		t.Fatalf("expected successful settings message, got %#v", finished)
	}
	expectedSettings := application.SolutionSettings{SolutionName: "CatalogPlatform", SolutionDescription: "New description", TargetFramework: "9"}
	if capturedSettings != expectedSettings {
		t.Fatalf("expected settings %#v, got %#v", expectedSettings, capturedSettings)
	}
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command after save success")
	}
	if model.status != statusReady || model.plan.Config.SolutionName != "CatalogPlatform" || model.plan.FileCount != 3 {
		t.Fatalf("expected ready state with refreshed plan, got %#v", model)
	}
	assertContains(t, model.View(), "Settings saved. Plan refreshed.")
}

func TestModelViewShowsTargetFrameworkSuggestionsAndCyclesThem(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", TargetFramework: "net8.0"}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0", "net8.0"})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)

	view := model.View()
	assertContains(t, view, "Suggestions (newest first): net10.0, net9.0, net8.0")
	assertContains(t, view, "Type a major or TFM such as 6, 7, 10, or net10.0.")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	model = updated.(Model)
	if model.edit.targetFramework.string() != "net10.0" {
		t.Fatalf("expected ctrl+n to cycle to first suggestion, got %q", model.edit.targetFramework.string())
	}
}

func TestModelUpdateSupportsEditNavigationAndCancel(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", SolutionDescription: "Description", TargetFramework: "net8.0"}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updated.(Model)
	if model.edit.focused != editFieldName {
		t.Fatalf("expected shift+tab to return to name field, got %v", model.edit.focused)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model = updated.(Model)
	if model.edit.name.string() != "CommercePlatfoXm" {
		t.Fatalf("expected left/backspace/rune editing, got %q", model.edit.name.string())
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command on cancel")
	}
	if model.status != statusReady || model.plan.Config.SolutionName != "CommercePlatform" {
		t.Fatalf("expected cancel to keep original plan, got %#v", model)
	}
}

func TestModelUpdateCancelRestoresPreviousStatus(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.status = statusGenerated
	model.result = application.GenerateResult{OutputDir: "/tmp/generated", Plan: application.GenerationPlan{FileCount: 1}}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command on cancel")
	}
	if model.status != statusGenerated || model.result.OutputDir != "/tmp/generated" {
		t.Fatalf("expected cancel to restore generated state, got %#v", model)
	}
}

func TestModelUpdateSaveFailureKeepsEditorOpen(t *testing.T) {
	saveErr := errors.New("invalid config")
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", TargetFramework: "net8.0"}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		return application.UpdateSolutionSettingsResult{}, saveErr
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusEditing || model.err != saveErr {
		t.Fatalf("expected failed save to keep editor open, got %#v", model)
	}
	assertContains(t, model.View(), "Save failed: invalid config")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || updated.(Model).status != statusReady {
		t.Fatalf("expected cancel after save failure, got status=%v cmd=%v", updated.(Model).status, cmd)
	}
}

func TestModelUpdateSaveSuccessWithRefreshFailureAllowsRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", SolutionDescription: "Old description", TargetFramework: "net8.0"}
	savedConfig := application.ConfigSummary{SolutionName: "CatalogPlatform", SolutionDescription: "New description", TargetFramework: "net9.0", ServiceCount: 1, EntityCount: 1, ValueObjectCount: 1, ServiceNames: []string{"CatalogService"}}
	refreshedPlan := plannedFilesPlan(2)
	refreshedPlan.Config = application.ConfigSummary{SolutionName: "CatalogPlatform", SolutionDescription: "New description", TargetFramework: "net9.0", ServiceCount: 2, EntityCount: 3, ValueObjectCount: 1, ServiceNames: []string{"CatalogService", "OrderService"}}
	refreshedPlan.OutputDir = "/tmp/refreshed"
	refreshedPlan.OutputAction = "replace"
	model := NewModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return refreshedPlan, nil
	}, nil, func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		return application.UpdateSolutionSettingsResult{Saved: true, Config: savedConfig, PlanError: refreshErr}, nil
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}

	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusFailed || model.err != refreshErr || model.errContext != "Refresh after save" {
		t.Fatalf("expected refresh-after-save failure state, got %#v", model)
	}
	view := model.View()
	assertContains(t, view, "Product: CatalogPlatform")
	assertContains(t, view, "Description: New description")
	assertContains(t, view, "Target framework: net9.0")
	assertContains(t, view, "Settings saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "Refresh after save failed: plan failed")
	assertContains(t, view, "Only refresh retry is available until the plan refresh succeeds.")
	assertContains(t, view, "Press r to retry the plan refresh.")
	assertContains(t, view, "Press q, esc, or ctrl+c to quit.")
	assertNotContains(t, view, "Save failed")
	assertNotContains(t, view, "Esc cancels")
	assertNotContains(t, view, "g to retry generation")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "Press e to edit solution settings")
	updated, generateCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if generateCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected generation retry to be blocked after save refresh failure, got status=%v cmd=%v", updated.(Model).status, generateCmd)
	}
	updated, editCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if editCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected edit to be blocked after save refresh failure, got status=%v cmd=%v", updated.(Model).status, editCmd)
	}

	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if model.status != statusRefreshing || retryCmd == nil {
		t.Fatalf("expected refresh retry command, got status=%v cmd=%v", updated.(Model).status, retryCmd)
	}
	msg := retryCmd()
	if retries != 1 {
		t.Fatalf("expected one retry, got %d", retries)
	}
	updated, cmd = model.Update(msg)
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command after refresh retry finishes")
	}
	if model.status != statusReady || model.plan.OutputDir != "/tmp/refreshed" || model.plan.FileCount != 2 || model.plan.Config.ServiceCount != 2 {
		t.Fatalf("expected refresh retry to replace stale plan fully, got %#v", model)
	}
}

func TestModelUpdateBlocksQuitAndActionsWhileSavingOrEditing(t *testing.T) {
	for _, msg := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}} {
		model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
		model.status = statusSaving
		updated, cmd := model.Update(msg)
		if cmd != nil || updated.(Model).status != statusSaving {
			t.Fatalf("expected quit to be blocked while saving for %q", msg.String())
		}
	}
	model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		t.Fatal("refresh should not run while editing")
		return application.GenerationPlan{}, nil
	}, func(application.GenerateRequest) (application.GenerateResult, error) {
		t.Fatal("generation should not run while editing")
		return application.GenerateResult{}, nil
	}, nil)
	model.status = statusEditing
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil || updated.(Model).edit.name.string() != "r" {
		t.Fatalf("expected refresh key to edit text while editing, got cmd=%v model=%#v", cmd, updated)
	}
	updated, cmd = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd != nil || updated.(Model).edit.name.string() != "rg" {
		t.Fatalf("expected generation key to edit text while editing, got cmd=%v model=%#v", cmd, updated)
	}
}

func TestModelUpdateRecordsGenerationSuccess(t *testing.T) {
	result := application.GenerateResult{
		OutputDir: "/tmp/generated",
		Warning:   "existing warning",
		Plan:      application.GenerationPlan{OutputDir: "/tmp/generated", FileCount: 3},
	}

	updated, cmd := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(generationFinishedMsg{result: result})
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
	assertContains(t, view, generatedHelp)
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "g to generate")
}

func TestModelUpdateRecordsGenerationFailureAndAllowsRetry(t *testing.T) {
	generationErr := errors.New("write failed")
	retries := 0
	model := NewModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		retries++
		return application.GenerateResult{}, nil
	}, nil)

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
	assertContains(t, view, "Press r to refresh the plan or g to retry generation.")
	assertContains(t, view, readyHelp)

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

func plannedFilesPlan(count int) application.GenerationPlan {
	files := make([]application.PlannedFile, count)
	for index := range files {
		files[index] = application.PlannedFile{Path: fmt.Sprintf("file-%02d.txt", index+1), Action: "create"}
	}
	return application.GenerationPlan{FileCount: count, Files: files}
}

func assertNotContains(t *testing.T, value, unexpected string) {
	t.Helper()
	if strings.Contains(value, unexpected) {
		t.Fatalf("expected %q not to contain %q", value, unexpected)
	}
}
