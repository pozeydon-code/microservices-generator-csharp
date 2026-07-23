package tui

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
			Services: []application.ServiceSummary{
				{Name: "ProductService", EntityNames: []string{"Product"}},
				{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}},
			},
		},
		OutputDir:      "/tmp/generated",
		OutputAction:   "replace",
		ForceRequired:  true,
		ForceUsed:      true,
		FileCount:      6,
		ExtraFileCount: 1,
		DeletedFiles:   []string{"src/ProductService/OldEndpoint.cs"},
		Files: []application.PlannedFile{
			{Path: "README.md", Action: "replace"},
			{Path: "src/ProductService/Product.cs", Action: "replace"},
			{Path: "src/ProductService/ProductService.Api/ProductEndpoints.cs", Action: "create"},
			{Path: "src/ProductService/ProductService.Domain/Product.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs", Action: "create"},
		},
	}

	model := NewModel(plan, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil)
	view := model.View()

	assertContains(t, view, "Microgen - READY")
	assertContains(t, view, "Step-based generator dashboard")
	assertContains(t, view, "Primary: g Generate")
	assertContains(t, view, "Wizard")
	assertContains(t, view, "> 1/5 Source")
	assertContains(t, view, "  2/5 Project")
	assertContains(t, view, "  3/5 Services")
	assertContains(t, view, "  4/5 Preview")
	assertContains(t, view, "  5/5 Generate")
	assertContains(t, view, "Progress 1/5")
	assertContains(t, view, "Source")
	assertContains(t, view, "Source microgen.json (existing JSON)")
	assertContains(t, view, "Output /tmp/generated")
	assertContains(t, view, "Mode replace")
	assertContains(t, view, "Steps: tab/] next, shift+tab/[ previous.")

	model.currentStep = stepProject
	view = model.View()
	assertContains(t, view, "Solution CommercePlatform")
	assertContains(t, view, "Description Product management.")
	assertContains(t, view, "Target net8.0")
	assertContains(t, view, "Format .sln")
	assertContains(t, view, "e Edit solution name, description, or target framework.")

	model.currentStep = stepServices
	view = model.View()
	assertContains(t, view, "Summary 2 services, 3 entities, 3 value objects")
	assertContains(t, view, "> ProductService: Product")
	assertContains(t, view, "  OrderService: Order, OrderLine")
	assertContains(t, view, "up/down choose service, enter edit entities, e edit services.")
	assertContains(t, view, "Entity fields can be edited from the entity editor; value objects are upcoming. New entities get Id Guid.")

	model.currentStep = stepPreview
	view = model.View()
	assertContains(t, view, "Output Preview")
	assertContains(t, view, "Directory /tmp/generated")
	assertContains(t, view, "Write mode replace")
	assertContains(t, view, "Force required=yes, used=yes")
	assertContains(t, view, "Files 6 planned")
	assertContains(t, view, "Impact create=4, replace=2 (mixed actions)")
	assertContains(t, view, "DANGER replacement removes 1 previous generated file(s)")
	assertContains(t, view, "src/ProductService/OldEndpoint.cs")
	assertContains(t, view, "Planned Files")
	assertContains(t, view, "Files 1-5 of 6 (filter: all)")
	assertContains(t, view, "Selected: 1/6 [REPLACE] README.md")
	assertContains(t, view, "> [1/6] [REPLACE] README.md")
	assertContains(t, view, "  [5/6] [CREATE] tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs")
	assertContains(t, view, readyHelp)
	assertContains(t, view, "Exit: q/esc/ctrl+c")

	model.currentStep = stepGenerate
	view = model.View()
	assertContains(t, view, "Generate 6 planned file(s) into /tmp/generated.")
	assertContains(t, view, "Review the Preview step before confirming writes.")
	if strings.Contains(view, "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs") {
		t.Fatalf("expected file preview to be truncated, got view %q", view)
	}
}

func TestModelViewShowsPrimaryActionOnce(t *testing.T) {
	view := stripANSI(NewModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil).View())

	if count := strings.Count(view, "Primary:"); count != 1 {
		t.Fatalf("expected one primary action, got %d in %q", count, view)
	}
}

func TestModelViewShowsBootstrappedConfigSource(t *testing.T) {
	view := NewModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "starter.json", ConfigBootstrapped: true}, nil, nil, nil).View()

	assertContains(t, view, "Source starter.json (starter config bootstrapped this run)")
	assertContains(t, view, "Created starter config. Edit project, service, entity, and basic field settings incrementally.")
}

func TestModelDefaultsToSourceStep(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

	if model.currentStep != stepSource {
		t.Fatalf("expected source step by default, got %v", model.currentStep)
	}
	assertContains(t, model.View(), "> 1/5 Source")
}

func TestModelUpdateNavigatesSteps(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepProject {
		t.Fatalf("expected tab to move to project step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepServices {
		t.Fatalf("expected ] to move to services step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepProject {
		t.Fatalf("expected shift+tab to move back to project step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepSource {
		t.Fatalf("expected [ to move back to source step, got step=%v cmd=%v", model.currentStep, cmd)
	}
}

func TestModelUpdateIgnoresStepNavigationWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
			updatedModel := updated.(Model)

			if cmd != nil || updatedModel.currentStep != stepSource {
				t.Fatalf("expected busy step navigation to be ignored, got step=%v cmd=%v", updatedModel.currentStep, cmd)
			}
		})
	}
}

func TestModelViewTruncatesDeletedFilePreview(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.ExtraFileCount = 4
	plan.DeletedFiles = []string{"old-1.txt", "old-2.txt", "old-3.txt", "old-4.txt"}

	model := NewModel(plan, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil)
	model.currentStep = stepPreview
	view := model.View()

	assertContains(t, view, "DANGER replacement removes 4 previous generated file(s)")
	assertContains(t, view, "old-1.txt, old-2.txt, old-3.txt, and 1 more")
}

func TestModelViewShowsPlannedFileRangeAndCursor(t *testing.T) {
	view := modelOnStep(plannedFilesPlan(6), stepPreview).View()

	assertContains(t, view, "Files 1-5 of 6 (filter: all)")
	assertContains(t, view, "Selected: 1/6 [CREATE] file-01.txt")
	assertContains(t, view, "> [1/6] [CREATE] file-01.txt")
	assertContains(t, view, "  [5/6] [CREATE] file-05.txt")
	assertNotContains(t, view, "file-06.txt")
}

func TestModelUpdateMovesPlannedFileCursorAndWindow(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(7), stepPreview)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	assertContains(t, model.View(), "Files 1-5 of 7 (filter: all)")
	assertContains(t, model.View(), "> [2/7] [CREATE] file-02.txt")

	for range 4 {
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		model = updated.(Model)
		if cmd != nil {
			t.Fatal("expected no command")
		}
	}
	view := model.View()
	assertContains(t, view, "Files 2-6 of 7 (filter: all)")
	assertContains(t, view, "> [6/7] [CREATE] file-06.txt")
	assertNotContains(t, view, "file-01.txt")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 2-6 of 7 (filter: all)")
	assertContains(t, view, "> [5/7] [CREATE] file-05.txt")
}

func TestModelUpdateClampsPlannedFileNavigationBounds(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(3), stepPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Files 1-3 of 3 (filter: all)")
	assertContains(t, view, "> [1/3] [CREATE] file-01.txt")

	for range 5 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	view = model.View()
	assertContains(t, view, "Files 1-3 of 3 (filter: all)")
	assertContains(t, view, "> [3/3] [CREATE] file-03.txt")
}

func TestModelUpdateSupportsPlannedFileHomeEndAndPageKeys(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(12), stepPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Files 2-6 of 12 (filter: all)")
	assertContains(t, view, "> [6/12] [CREATE] file-06.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-5 of 12 (filter: all)")
	assertContains(t, view, "> [1/12] [CREATE] file-01.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 8-12 of 12 (filter: all)")
	assertContains(t, view, "> [12/12] [CREATE] file-12.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-5 of 12 (filter: all)")
	assertContains(t, view, "> [1/12] [CREATE] file-01.txt")
}

func TestModelUpdateWindowSizeChangesVisibleFileRange(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(20), stepPreview)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 1-6 of 20 (filter: all)")
	assertContains(t, view, "  [6/20] [CREATE] file-06.txt")
	assertNotContains(t, view, "file-07.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 1-12 of 20 (filter: all)")
	assertContains(t, view, "  [12/20] [CREATE] file-12.txt")
	assertNotContains(t, view, "file-13.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Files 1-3 of 20 (filter: all)")
	assertContains(t, view, "  [3/20] [CREATE] file-03.txt")
	assertNotContains(t, view, "file-04.txt")
}

func TestModelUpdateClampsNavigationAfterResize(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(20), stepPreview)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	assertContains(t, model.View(), "Files 9-20 of 20 (filter: all)")
	assertContains(t, model.View(), "> [20/20] [CREATE] file-20.txt")

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 18-20 of 20 (filter: all)")
	assertContains(t, view, "> [20/20] [CREATE] file-20.txt")
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

	view := modelOnStep(plan, stepPreview).View()

	assertContains(t, view, "Files 5 planned")
	assertContains(t, view, "Impact create=2, replace=2, unchanged=1 (mixed actions)")
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
	model := modelOnStep(plan, stepPreview)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Files 1-2 of 2 (filter: create)")
	assertContains(t, view, "Filter Press a to cycle filters back to all.")
	assertContains(t, view, "Selected: 1/2 [CREATE] create-1.txt")
	assertContains(t, view, "> [1/2] [CREATE] create-1.txt")
	assertNotContains(t, view, "replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Selected: 2/2 [CREATE] create-2.txt")
	assertContains(t, view, "> [2/2] [CREATE] create-2.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-2 of 2 (filter: replace)")
	assertContains(t, view, "Selected: 1/2 [REPLACE] replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Files 1-1 of 1 (filter: unchanged)")
	assertContains(t, view, "Selected: 1/1 [UNCHANGED] unchanged-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	assertContains(t, model.View(), "Files 1-5 of 5 (filter: all)")
	assertNotContains(t, model.View(), "Filter Press a to cycle filters back to all.")
}

func TestModelViewReassuresWhenAllPlannedFilesAreUnchanged(t *testing.T) {
	plan := application.GenerationPlan{
		FileCount: 2,
		Files: []application.PlannedFile{
			{Path: "unchanged-1.txt", Action: "unchanged"},
			{Path: "unchanged-2.txt", Action: "unchanged"},
		},
	}

	view := modelOnStep(plan, stepPreview).View()

	assertContains(t, view, "Impact unchanged only")
	assertContains(t, view, "No generated file content changes detected.")
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

	assertContains(t, view, "Microgen - REFRESHING")
	assertContains(t, view, "Primary: Refreshing plan")
	assertContains(t, view, "Refreshing plan. Please wait; editing, filtering, and generation are paused.")
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
	assertContains(t, view, "Microgen - GENERATING")
	assertContains(t, view, "Primary: Generating files")
	assertContains(t, view, "Generating files. Please wait; exit is available after generation finishes.")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "Exit: q/esc/ctrl+c")
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
	assertContains(t, view, "Microgen - REFRESHING")
	assertContains(t, view, "Primary: Refreshing plan")
	assertContains(t, view, "Refreshing plan. Please wait; editing, filtering, and generation are paused.")
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
	assertContains(t, view, "Product Refreshed")
	assertContains(t, view, "Target net9.0")
	assertContains(t, view, "Output /tmp/refreshed")
	updatedModel.currentStep = stepPreview
	view = updatedModel.View()
	assertContains(t, view, "Directory /tmp/refreshed")
	assertContains(t, view, "Files 1-2 of 2 (filter: all)")
	assertContains(t, view, "> [2/2] [CREATE] file-02.txt")
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
	assertContains(t, view, "Microgen - FAILED")
	assertContains(t, view, "Primary: g Retry generation")
	assertContains(t, view, "FAILED Refresh failed: plan failed")
	assertContains(t, view, "g Retry generation, or r refresh the plan first.")

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
	assertContains(t, model.View(), "Use the Services step for service, entity, and basic field editing. Value-object editing is upcoming.")

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
	assertContains(t, model.View(), "Microgen - SAVING")
	assertContains(t, model.View(), "Primary: Saving settings")
	assertContains(t, model.View(), "Saving settings...")
	assertNotContains(t, model.View(), readyHelp)
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

func TestModelUpdateEditModeTabNavigatesFieldsNotSteps(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", TargetFramework: "net8.0"}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if model.currentStep != stepProject {
		t.Fatalf("expected edit mode to stay on project step, got %v", model.currentStep)
	}
	if model.edit.focused != editFieldDescription {
		t.Fatalf("expected tab to move editor focus, got %v", model.edit.focused)
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
	assertContains(t, view, "Microgen - FAILED")
	assertContains(t, view, "Primary: r Retry refresh")
	assertContains(t, view, "Solution CatalogPlatform")
	assertContains(t, view, "Description New description")
	assertContains(t, view, "Target net9.0")
	assertContains(t, view, "Settings saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	assertContains(t, view, "r Retry plan refresh. Other actions stay locked until refresh succeeds.")
	assertContains(t, view, "Keys: r retry refresh | q/esc/ctrl+c quit")
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

func TestModelUpdateSaveRefreshRetryFailureKeepsStalePlanLocked(t *testing.T) {
	initialErr := errors.New("initial plan failed")
	retryErr := errors.New("retry plan failed")
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{SolutionName: "CommercePlatform", TargetFramework: "net8.0"}
	savedConfig := application.ConfigSummary{SolutionName: "CatalogPlatform", TargetFramework: "net9.0"}
	model := NewModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return application.GenerationPlan{}, retryErr
	}, nil, nil)

	updated, cmd := model.Update(settingsFinishedMsg{result: application.UpdateSolutionSettingsResult{Saved: true, Config: savedConfig, PlanError: initialErr}})
	model = updated.(Model)
	if cmd != nil || !model.postSaveRefreshFailed() {
		t.Fatalf("expected initial refresh-after-save failure lock, cmd=%v model=%#v", cmd, model)
	}

	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if model.status != statusRefreshing || retryCmd == nil {
		t.Fatalf("expected retry refresh command, got status=%v cmd=%v", model.status, retryCmd)
	}
	updated, cmd = model.Update(retryCmd())
	model = updated.(Model)
	if cmd != nil || !model.postSaveRefreshFailed() || model.err != retryErr {
		t.Fatalf("expected failed retry to keep stale-plan lock, cmd=%v model=%#v", cmd, model)
	}

	for _, tt := range []struct {
		name string
		msg  tea.KeyMsg
	}{
		{name: "generate", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}},
		{name: "edit", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}},
		{name: "navigate", msg: tea.KeyMsg{Type: tea.KeyTab}},
		{name: "entities", msg: tea.KeyMsg{Type: tea.KeyEnter}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			updated, cmd := model.Update(tt.msg)
			updatedModel := updated.(Model)
			if cmd != nil || !updatedModel.postSaveRefreshFailed() {
				t.Fatalf("expected %s to stay blocked after retry failure, cmd=%v model=%#v", tt.name, cmd, updatedModel)
			}
		})
	}
}

func TestModelUpdateEditsServicesAndSaves(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{ServiceCount: 2, EntityCount: 2, ServiceNames: []string{"ProductService", "OrderService"}}
	updatedPlan := plannedFilesPlan(4)
	updatedPlan.Config = application.ConfigSummary{ServiceCount: 2, EntityCount: 2, ServiceNames: []string{"CatalogService", "Service3Service"}}
	var capturedSettings application.ServiceSettings
	model := NewModel(plan, request, nil, nil, nil)
	model.currentStep = stepServices
	model.updateServices = func(actual application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		capturedSettings = settings
		return application.UpdateServiceSettingsResult{Saved: true, Plan: updatedPlan}, nil
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeServices {
		t.Fatalf("expected services edit mode, got status=%v mode=%v cmd=%v", model.status, model.edit.mode, cmd)
	}
	assertContains(t, model.View(), "Editing services")
	assertContains(t, model.View(), "Keys: up/down select, a add, r rename, d delete, enter save, esc cancel.")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("ProductService")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("CatalogService")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.servicesEdit.renaming || model.servicesEdit.services[0].string() != "CatalogService" {
		t.Fatalf("expected local rename confirmation, got %#v", model.servicesEdit)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if model.servicesEdit.selected != 2 || model.servicesEdit.services[2].string() != "Service3Service" {
		t.Fatalf("expected added placeholder service selected, got %#v", model.servicesEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if got := serviceEditNames(model.servicesEdit.services); !reflect.DeepEqual(got, []string{"CatalogService", "Service3Service"}) {
		t.Fatalf("expected order service deleted, got %#v", got)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	assertContains(t, model.View(), "Saving services...")
	msg := cmd()
	finished, ok := msg.(servicesFinishedMsg)
	if !ok {
		t.Fatalf("expected servicesFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 4 {
		t.Fatalf("expected successful services message, got %#v", finished)
	}
	expectedServices := []application.ServiceNameSetting{{OriginalName: "ProductService", Name: "CatalogService"}, {Name: "Service3Service"}}
	if !reflect.DeepEqual(capturedSettings.Services, expectedServices) {
		t.Fatalf("expected captured service settings, got %#v", capturedSettings.Services)
	}
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.plan.FileCount != 4 || model.plan.Config.ServiceNames[0] != "CatalogService" {
		t.Fatalf("expected ready state with refreshed services plan, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Services saved. Plan refreshed. Use enter on the Services step to edit entities and fields.")
}

func TestModelUpdateServicesEditCancelKeepsPlan(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ServiceNames: []string{"ProductService"}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.status != statusReady || !reflect.DeepEqual(model.plan.Config.ServiceNames, []string{"ProductService"}) {
		t.Fatalf("expected cancel to keep original services, got cmd=%v model=%#v", cmd, model)
	}
}

func TestModelUpdateServicesEditDeleteKeepsOneServiceLocally(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ServiceNames: []string{"ProductService"}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)

	if cmd != nil || len(model.servicesEdit.services) != 1 || model.servicesEdit.services[0].string() != "ProductService" {
		t.Fatalf("expected last service deletion to be ignored locally, got cmd=%v services=%#v", cmd, model.servicesEdit.services)
	}
}

func TestModelUpdateServicesSaveFailureKeepsEditorOpen(t *testing.T) {
	saveErr := errors.New("invalid config")
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ServiceNames: []string{"ProductService"}}
	model := modelOnStep(plan, stepServices)
	model.updateServices = func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		return application.UpdateServiceSettingsResult{}, saveErr
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusEditing || model.edit.mode != editModeServices || model.err != saveErr {
		t.Fatalf("expected failed services save to keep editor open, got %#v", model)
	}
	assertContains(t, model.View(), "Save failed: invalid config")
}

func TestModelUpdateServicesSaveSuccessWithRefreshFailureAllowsRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ServiceNames: []string{"ProductService"}}
	savedConfig := application.ConfigSummary{ServiceCount: 2, EntityCount: 2, ServiceNames: []string{"CatalogService", "BillingService"}}
	refreshedPlan := plannedFilesPlan(2)
	refreshedPlan.Config = savedConfig
	model := modelOnStep(plan, stepServices)
	model.planFunc = func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return refreshedPlan, nil
	}
	model.updateServices = func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		return application.UpdateServiceSettingsResult{Saved: true, Config: savedConfig, PlanError: refreshErr}, nil
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusFailed || model.err != refreshErr || model.errContext != "Refresh after save" || model.plan.Config.ServiceCount != 2 {
		t.Fatalf("expected services refresh-after-save failure state, got %#v", model)
	}
	view := model.View()
	assertContains(t, view, "Services saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	updated, editCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if editCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected edit to be blocked after services save refresh failure, got status=%v cmd=%v", updated.(Model).status, editCmd)
	}
	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if model.status != statusRefreshing || retryCmd == nil {
		t.Fatalf("expected refresh retry command, got status=%v cmd=%v", model.status, retryCmd)
	}
	updated, _ = model.Update(retryCmd())
	model = updated.(Model)
	if retries != 1 || model.status != statusReady || model.plan.FileCount != 2 {
		t.Fatalf("expected refresh retry to restore ready plan, retries=%d model=%#v", retries, model)
	}
}

func TestModelUpdateSelectsServiceAndEditsEntitiesAndSaves(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{
		ServiceCount: 2,
		EntityCount:  3,
		ServiceNames: []string{"ProductService", "OrderService"},
		Services: []application.ServiceSummary{
			{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}}},
			{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}, Entities: []application.EntitySummary{{Name: "Order", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}}}, {Name: "OrderLine", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Quantity", Type: "int"}}}}},
		},
	}
	updatedPlan := plannedFilesPlan(4)
	updatedPlan.Config = application.ConfigSummary{
		ServiceCount: 2,
		EntityCount:  2,
		ServiceNames: []string{"ProductService", "OrderService"},
		Services: []application.ServiceSummary{
			{Name: "ProductService", EntityNames: []string{"Product"}},
			{Name: "OrderService", EntityNames: []string{"Purchase", "Entity3"}},
		},
	}
	var capturedSettings application.EntitySettings
	model := modelOnStep(plan, stepServices)
	model.updateEntities = func(actual application.GenerateRequest, settings application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		capturedSettings = settings
		return application.UpdateEntitySettingsResult{Saved: true, Plan: updatedPlan}, nil
	}
	model.request = request

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.selectedService != 1 {
		t.Fatalf("expected selected service cursor to move to OrderService, got selected=%d cmd=%v", model.selectedService, cmd)
	}
	assertContains(t, model.View(), "> OrderService: Order, OrderLine")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeEntities || model.entitiesEdit.serviceName != "OrderService" {
		t.Fatalf("expected entity edit mode for OrderService, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Editing entities for OrderService")
	assertContains(t, model.View(), "Press f to edit fields for the selected saved entity. New entities get Id Guid.")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("Order")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Purchase")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.entitiesEdit.renaming || model.entitiesEdit.entities[0].string() != "Purchase" {
		t.Fatalf("expected local entity rename confirmation, got %#v", model.entitiesEdit)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if model.entitiesEdit.selected != 2 || model.entitiesEdit.entities[2].string() != "Entity3" {
		t.Fatalf("expected added placeholder entity selected, got %#v", model.entitiesEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if got := entityEditNames(model.entitiesEdit.entities); !reflect.DeepEqual(got, []string{"Purchase", "Entity3"}) {
		t.Fatalf("expected OrderLine entity deleted, got %#v", got)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	assertContains(t, model.View(), "Saving entities...")
	msg := cmd()
	finished, ok := msg.(entitiesFinishedMsg)
	if !ok {
		t.Fatalf("expected entitiesFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 4 {
		t.Fatalf("expected successful entities message, got %#v", finished)
	}
	expectedEntities := []application.EntityNameSetting{{OriginalName: "Order", Name: "Purchase"}, {Name: "Entity3"}}
	if capturedSettings.ServiceName != "OrderService" || !reflect.DeepEqual(capturedSettings.Entities, expectedEntities) {
		t.Fatalf("expected captured entity settings for selected service, got %#v", capturedSettings)
	}
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.plan.FileCount != 4 || model.plan.Config.Services[1].EntityNames[0] != "Purchase" {
		t.Fatalf("expected ready state with refreshed entity plan, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Entities saved. Plan refreshed. Press f in the entity editor to edit fields.")
}

func TestModelUpdateEntitiesRenameInsertsShortcutRunes(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}}}}
	model := modelOnStep(plan, stepServices)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeEntities {
		t.Fatalf("expected entity editor, got cmd=%v model=%#v", cmd, model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("Product")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if len(model.entitiesEdit.entities) != 1 || model.entitiesEdit.entities[0].string() != "a" {
		t.Fatalf("expected a to edit entity name during rename, got %#v", model.entitiesEdit)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if len(model.entitiesEdit.entities) != 2 || model.entitiesEdit.entities[1].string() != "Entity2" {
		t.Fatalf("expected a to add an entity after rename confirmation, got %#v", model.entitiesEdit)
	}
}

func TestModelUpdateEntitiesEditCancelKeepsPlan(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.status != statusReady || !reflect.DeepEqual(model.plan.Config.Services[0].EntityNames, []string{"Product"}) {
		t.Fatalf("expected cancel to keep original entities, got cmd=%v model=%#v", cmd, model)
	}
}

func TestModelUpdateEntitiesEditDeleteKeepsOneEntityLocally(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)

	if cmd != nil || len(model.entitiesEdit.entities) != 1 || model.entitiesEdit.entities[0].string() != "Product" {
		t.Fatalf("expected last entity deletion to be ignored locally, got cmd=%v entities=%#v", cmd, model.entitiesEdit.entities)
	}
}

func TestModelUpdateEntitiesSaveFailureKeepsEditorOpen(t *testing.T) {
	saveErr := errors.New("invalid config")
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}}}}
	model := modelOnStep(plan, stepServices)
	model.updateEntities = func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		return application.UpdateEntitySettingsResult{}, saveErr
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusEditing || model.edit.mode != editModeEntities || model.err != saveErr {
		t.Fatalf("expected failed entity save to keep editor open, got %#v", model)
	}
	assertContains(t, model.View(), "Save failed: invalid config")
}

func TestModelUpdateEntitiesSaveSuccessWithRefreshFailureBlocksUntilRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}}}}
	savedConfig := application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Catalog"}}}}
	refreshedPlan := plannedFilesPlan(2)
	refreshedPlan.Config = savedConfig
	model := modelOnStep(plan, stepServices)
	model.planFunc = func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return refreshedPlan, nil
	}
	model.updateEntities = func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		return application.UpdateEntitySettingsResult{Saved: true, Config: savedConfig, PlanError: refreshErr}, nil
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusFailed || model.err != refreshErr || model.errContext != "Refresh after save" || model.plan.Config.Services[0].EntityNames[0] != "Catalog" {
		t.Fatalf("expected entities refresh-after-save failure state, got %#v", model)
	}
	view := model.View()
	assertContains(t, view, "Entities saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	updated, editCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if editCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected entity edit to be blocked after save refresh failure, got status=%v cmd=%v", updated.(Model).status, editCmd)
	}
	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if model.status != statusRefreshing || retryCmd == nil {
		t.Fatalf("expected refresh retry command, got status=%v cmd=%v", model.status, retryCmd)
	}
	updated, _ = model.Update(retryCmd())
	model = updated.(Model)
	if retries != 1 || model.status != statusReady || model.plan.FileCount != 2 {
		t.Fatalf("expected refresh retry to restore ready plan, retries=%d model=%#v", retries, model)
	}
}

func TestModelUpdateOpensFieldsEditorAndSavesFieldChanges(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{
		ServiceCount: 1,
		EntityCount:  1,
		Services:     []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}}}},
	}
	updatedPlan := plannedFilesPlan(4)
	updatedPlan.Config = application.ConfigSummary{
		ServiceCount: 1,
		EntityCount:  1,
		Services:     []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Title", Type: "string"}, {Name: "Name", Type: "decimal"}}}}}},
	}
	var capturedSettings application.FieldSettings
	model := modelOnStep(plan, stepServices)
	model.request = request
	model.updateFields = func(actual application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		capturedSettings = settings
		return application.UpdateFieldSettingsResult{Saved: true, Plan: updatedPlan}, nil
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.edit.mode != editModeEntities {
		t.Fatalf("expected entity editor, got cmd=%v model=%#v", cmd, model)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	if cmd != nil || model.edit.mode != editModeFields || model.fieldsEdit.entityName != "Product" {
		t.Fatalf("expected fields editor, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Editing fields for ProductService/Product")
	assertContains(t, model.View(), "Keys: up/down select, a add string field, r rename, t edit type, d delete, enter save, esc back.")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("Name")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Title")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.fieldsEdit.editingName || model.fieldsEdit.fields[1].name.string() != "Title" {
		t.Fatalf("expected local field rename confirmation, got %#v", model.fieldsEdit)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if model.fieldsEdit.selected != 2 || model.fieldsEdit.fields[2].name.string() != "Name" || model.fieldsEdit.fields[2].typeName.string() != "string" {
		t.Fatalf("expected added string field selected, got %#v", model.fieldsEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)
	for range len([]rune("string")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("decimal")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if len(model.fieldsEdit.fields) != 2 || model.fieldsEdit.fields[0].name.string() != "Title" {
		t.Fatalf("expected Id field deleted locally, got %#v", model.fieldsEdit)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	assertContains(t, model.View(), "Saving fields...")
	msg := cmd()
	finished, ok := msg.(fieldsFinishedMsg)
	if !ok {
		t.Fatalf("expected fieldsFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 4 {
		t.Fatalf("expected successful fields message, got %#v", finished)
	}
	wantFields := []application.FieldSetting{{OriginalName: "Name", Name: "Title", Type: "string"}, {Name: "Name", Type: "decimal"}}
	if capturedSettings.ServiceName != "ProductService" || capturedSettings.EntityName != "Product" || !reflect.DeepEqual(capturedSettings.Fields, wantFields) {
		t.Fatalf("expected captured field settings, got %#v", capturedSettings)
	}
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.plan.FileCount != 4 {
		t.Fatalf("expected ready state with refreshed fields plan, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Fields saved. Plan refreshed. Value-object editing is upcoming.")
}

func TestModelUpdateFieldsEditCancelReturnsToEntitiesEditor(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeEntities {
		t.Fatalf("expected esc to return to entities editor, got cmd=%v model=%#v", cmd, model)
	}
}

func TestModelUpdateFieldsRenameAndTypeInsertShortcutRunes(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("Id")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if len(model.fieldsEdit.fields) != 1 || model.fieldsEdit.fields[0].name.string() != "a" {
		t.Fatalf("expected a to edit field name during rename, got %#v", model.fieldsEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)
	for range len([]rune("Guid")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if len(model.fieldsEdit.fields) != 1 || model.fieldsEdit.fields[0].typeName.string() != "d" {
		t.Fatalf("expected d to edit field type during type edit, got %#v", model.fieldsEdit)
	}
}

func TestModelUpdateFieldsSaveSuccessWithRefreshFailureBlocksUntilRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}}}}}
	savedConfig := application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}}}}}
	refreshedPlan := plannedFilesPlan(2)
	refreshedPlan.Config = savedConfig
	model := modelOnStep(plan, stepServices)
	model.planFunc = func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return refreshedPlan, nil
	}
	model.updateFields = func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		return application.UpdateFieldSettingsResult{Saved: true, Config: savedConfig, PlanError: refreshErr}, nil
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusFailed || model.err != refreshErr || model.errContext != "Refresh after save" || len(model.plan.Config.Services[0].Entities[0].Fields) != 2 {
		t.Fatalf("expected fields refresh-after-save failure state, got %#v", model)
	}
	view := model.View()
	assertContains(t, view, "Fields saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	updated, editCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if editCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected edit to be blocked after fields save refresh failure, got status=%v cmd=%v", updated.(Model).status, editCmd)
	}
	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if model.status != statusRefreshing || retryCmd == nil {
		t.Fatalf("expected refresh retry command, got status=%v cmd=%v", model.status, retryCmd)
	}
	updated, _ = model.Update(retryCmd())
	model = updated.(Model)
	if retries != 1 || model.status != statusReady || model.plan.FileCount != 2 {
		t.Fatalf("expected refresh retry to restore ready plan, retries=%d model=%#v", retries, model)
	}
}

func TestModelUpdateBlocksEntitiesEditWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := modelOnStep(plannedFilesPlan(1), stepServices)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			updatedModel := updated.(Model)

			if cmd != nil || updatedModel.status != status || updatedModel.edit.mode == editModeEntities {
				t.Fatalf("expected entities edit to be ignored while busy, got status=%v mode=%v cmd=%v", updatedModel.status, updatedModel.edit.mode, cmd)
			}
		})
	}
}

func TestModelUpdateBlocksServicesEditWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := modelOnStep(plannedFilesPlan(1), stepServices)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			updatedModel := updated.(Model)

			if cmd != nil || updatedModel.status != status || updatedModel.edit.mode == editModeServices {
				t.Fatalf("expected services edit to be ignored while busy, got status=%v mode=%v cmd=%v", updatedModel.status, updatedModel.edit.mode, cmd)
			}
		})
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
	assertContains(t, view, "Microgen - GENERATED")
	assertContains(t, view, "Primary: r Refresh")
	assertContains(t, view, "Generated 3 files in /tmp/generated.")
	assertContains(t, view, "WARNING existing warning")
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
	assertContains(t, view, "Microgen - FAILED")
	assertContains(t, view, "Primary: g Retry generation")
	assertContains(t, view, "FAILED Generation failed: write failed")
	assertContains(t, view, "g Retry generation, or r refresh the plan first.")
	assertContains(t, view, "Generate: g generate, r refresh.")

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
	value = stripANSI(value)
	if !strings.Contains(value, expected) {
		t.Fatalf("expected %q to contain %q", value, expected)
	}
}

func stripANSI(value string) string {
	return ansiRegexp.ReplaceAllString(value, "")
}

func plannedFilesPlan(count int) application.GenerationPlan {
	files := make([]application.PlannedFile, count)
	for index := range files {
		files[index] = application.PlannedFile{Path: fmt.Sprintf("file-%02d.txt", index+1), Action: "create"}
	}
	return application.GenerationPlan{FileCount: count, Files: files}
}

func serviceEditNames(services []textField) []string {
	names := make([]string, len(services))
	for index, service := range services {
		names[index] = service.string()
	}
	return names
}

func entityEditNames(entities []textField) []string {
	names := make([]string, len(entities))
	for index, entity := range entities {
		names[index] = entity.string()
	}
	return names
}

func modelOnStep(plan application.GenerationPlan, step tuiStep) Model {
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.currentStep = step
	return model
}

func assertNotContains(t *testing.T, value, unexpected string) {
	t.Helper()
	value = stripANSI(value)
	if strings.Contains(value, unexpected) {
		t.Fatalf("expected %q not to contain %q", value, unexpected)
	}
}
