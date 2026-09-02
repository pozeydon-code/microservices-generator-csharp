package tui

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
	"github.com/rivo/tview"
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
				{Name: "ProductService", EntityNames: []string{"Product"}, ValueObjectNames: []string{"ProductName"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}}},
				{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}, ValueObjectNames: []string{"OrderNumber", "Money"}},
			},
		},
		Readiness: application.ReadinessSummary{
			ProjectPresent:      true,
			ServiceCount:        2,
			EntityCount:         3,
			FieldCount:          2,
			ValueObjectCount:    3,
			OutputForceRequired: true,
			Hints:               []string{"Review output replacement; --force is required to write."},
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
			{Path: "src/ProductService/ProductService.WebApi/Controllers/Products/ProductController.cs", Action: "create"},
			{Path: "src/ProductService/ProductService.Domain/Product.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.WebApi.Tests/Features/Products/ProductControllerTests.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs", Action: "create"},
		},
	}

	model := workspaceModel(plan, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil)
	view := model.View()

	assertContains(t, view, "Microgen READY")
	assertContains(t, view, "Routes")
	assertContains(t, view, "g Generate")
	assertContains(t, view, "Routes")
	assertContains(t, view, "1 Overview")
	assertContains(t, view, "2 Project")
	assertContains(t, view, "3 Services")
	assertContains(t, view, "4 Entities")
	assertContains(t, view, "5 Value Objects")
	assertContains(t, view, "6 Preview")
	assertContains(t, view, "7 Generate")
	assertContains(t, view, "8 Result")
	assertNotContains(t, view, "Wizard")
	assertNotContains(t, view, "Progress 1/5")
	assertContains(t, view, "Source")
	assertContains(t, view, "Source microgen.json (existing JSON)")
	assertContains(t, view, "Output /tmp/generated")
	assertContains(t, view, "Mode replace")
	assertContains(t, view, "up/down route | enter open | ? help | ctrl+p routes")

	model.currentStep = stepProject
	view = model.View()
	assertContains(t, view, "Solution CommercePlatform")
	assertContains(t, view, "Description Product management.")
	assertContains(t, view, "Target net8.0")
	assertContains(t, view, "Format .sln")
	assertContains(t, view, "e Edit solution name, description, or target framework.")

	model.currentStep = stepServices
	view = model.View()
	assertContains(t, view, "Services")
	assertContains(t, view, "Selected service: ProductService")
	assertContains(t, view, "Context: [Services]  Entities  Value Objects")
	assertContains(t, view, "Service detail")
	assertContains(t, view, "Entities: 1")
	assertContains(t, view, "Fields: 2")
	assertContains(t, view, "Value objects: 1")
	assertContains(t, view, "References: 0")
	assertNotContains(t, view, "Editing entities")

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
	assertContains(t, view, "Rows 1-5/6 filter=all")
	assertContains(t, view, "Focus 1/6 [REPLACE] README.md")
	assertContains(t, view, "#   Action     Path")
	assertContains(t, view, ">  1 REPLACE    README.md")
	assertContains(t, view, "   5 CREATE")
	assertContains(t, view, "tests/ProductService/ProductService.WebApi.Tests/Features")
	assertContains(t, view, "files up/down/pg/home/end | a filter | r refresh | g generate")
	assertContains(t, view, "esc back | q quit")

	model.currentStep = stepGenerate
	view = model.View()
	assertContains(t, view, "Readiness project=yes, services=2, entities=3, fields=2, value objects=3, force required=yes")
	assertContains(t, view, "Next Review output replacement; --force is required to write.")
	assertContains(t, view, "Generate 6 planned file(s) into /tmp/generated.")
	assertContains(t, view, "Review the Preview step before confirming writes.")
	if strings.Contains(view, "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs") {
		t.Fatalf("expected file preview to be truncated, got view %q", view)
	}
}

func TestModelGenerateStepShowsStarterReadinessGuidance(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.OutputDir = "/tmp/generated"
	plan.Readiness = application.ReadinessSummary{
		ProjectPresent:   true,
		ServiceCount:     1,
		EntityCount:      1,
		FieldCount:       2,
		ValueObjectCount: 0,
		Hints: []string{
			"Rename the starter project.",
			"Rename the starter service.",
			"Rename the starter entity and add domain fields.",
			"Review the output preview before generating.",
		},
	}

	view := modelOnStep(plan, stepGenerate).View()

	assertContains(t, view, "Readiness project=yes, services=1, entities=1, fields=2, value objects=0, force required=no")
	assertContains(t, view, "Next Rename the starter project.")
	assertContains(t, view, "Next Rename the starter service.")
	assertContains(t, view, "Next Rename the starter entity and add domain fields.")
}

func TestModelGenerateStepShowsConfiguredMultiServiceReadiness(t *testing.T) {
	plan := plannedFilesPlan(4)
	plan.OutputDir = "/tmp/generated"
	plan.Readiness = application.ReadinessSummary{ProjectPresent: true, ServiceCount: 3, EntityCount: 5, FieldCount: 12, ValueObjectCount: 4, Hints: []string{"Review the output preview before generating."}}

	view := modelOnStep(plan, stepGenerate).View()

	assertContains(t, view, "Readiness project=yes, services=3, entities=5, fields=12, value objects=4, force required=no")
	assertContains(t, view, "Next Review the output preview before generating.")
}

func TestModelGenerateStepShowsForceRequiredReadinessWarning(t *testing.T) {
	plan := plannedFilesPlan(4)
	plan.OutputDir = "/tmp/generated"
	plan.ForceRequired = true
	plan.Readiness = application.ReadinessSummary{ProjectPresent: true, ServiceCount: 1, EntityCount: 1, FieldCount: 2, OutputForceRequired: true, Hints: []string{"Review output replacement; --force is required to write."}}

	view := modelOnStep(plan, stepGenerate).View()

	assertContains(t, view, "Readiness project=yes, services=1, entities=1, fields=2, value objects=0, force required=yes")
	assertContains(t, view, "Next Review output replacement; --force is required to write.")
}

func TestModelGenerateStepShowsPostGenerateImpactSummary(t *testing.T) {
	plan := plannedFilesPlan(4)
	plan.OutputDir = "/tmp/generated"
	plan.Files = []application.PlannedFile{
		{Path: "README.md", Action: "replace"},
		{Path: "src/ProductService/Product.cs", Action: "unchanged"},
		{Path: "src/ProductService/ProductEndpoint.cs", Action: "create"},
		{Path: "tests/ProductService/ProductTests.cs", Action: "create"},
	}
	plan.DeletedFiles = []string{"src/ProductService/OldEndpoint.cs", "tests/ProductService/OldEndpointTests.cs"}
	model := modelOnStep(plan, stepGenerate)
	model.status = statusGenerated
	model.result = application.GenerateResult{OutputDir: "/tmp/generated", Plan: plan}

	view := model.View()

	assertContains(t, view, "Generated 4 files written to /tmp/generated.")
	assertContains(t, view, "Impact created=2, replaced=1, unchanged=1")
	assertContains(t, view, "Cleanup deleted 2 previous generated file(s)")
	assertContains(t, view, "Next cd /tmp/generated && dotnet build")
	assertNotContains(t, view, "src/ProductService/OldEndpoint.cs")
}

func TestModelViewShowsPrimaryActionWithoutRedundantLabel(t *testing.T) {
	view := stripANSI(workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil).View())

	assertContains(t, view, "g Generate")
	assertNotContains(t, view, "Primary action")
}

func TestModelViewRendersProfessionalWorkspaceHeader(t *testing.T) {
	plan := plannedFilesPlan(2)
	plan.Config.SolutionName = "CommercePlatform"
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)

	view := stripANSI(model.View())

	assertContains(t, view, "Microgen READY")
	assertContains(t, view, "Project CommercePlatform")
	assertContains(t, view, "g Generate")
	assertNotContains(t, view, "Primary action")
	assertNotContains(t, view, "Current project:")
}

func TestModelViewRendersModernRouteNavigation(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.openScreen(screenPreview)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	if cmd != nil {
		t.Fatal("expected no command from window size")
	}
	model = updated.(Model)

	view := stripANSI(model.View())

	assertContains(t, view, "Routes")
	for _, route := range []string{"1 Overview", "2 Project", "3 Services", "4 Entities", "5 Value Objects", "6 Preview", "7 Generate", "8 Result"} {
		assertContains(t, view, route)
	}
	assertContains(t, view, "> 6 Preview")
	assertContains(t, view, "enter open | ctrl+p routes")
	assertContains(t, view, "route")
}

func TestModelViewFooterKeepsEssentialContextShortcuts(t *testing.T) {
	tests := []struct {
		name string
		set  func(*Model)
		want []string
	}{
		{
			name: "overview keeps compact global help",
			want: []string{"up/down route | enter open | ? help | ctrl+p routes", "esc back | q quit", "overview r refresh g generate"},
		},
		{
			name: "project keeps edit safety action",
			set:  func(model *Model) { model.openScreen(screenProject) },
			want: []string{"up/down route | enter open | ? help | ctrl+p routes", "project e edit r refresh"},
		},
		{
			name: "failed generation keeps retry cue",
			set: func(model *Model) {
				model.openScreen(screenResult)
				model.status = statusFailed
				model.err = errors.New("boom")
			},
			want: []string{"result g retry esc generate r refresh", "esc back | q quit"},
		},
		{
			name: "stale refresh lock keeps only safe actions",
			set: func(model *Model) {
				model.status = statusFailed
				model.errContext = "Refresh after save"
			},
			want: []string{"locked | r retry refresh | q quit"},
		},
		{
			name: "force preview keeps file navigation and force cue",
			set: func(model *Model) {
				model.openScreen(screenPreview)
				model.plan.ForceRequired = true
				model.plan.Readiness.OutputForceRequired = true
			},
			want: []string{"files up/down/pg/home/end | a filter | r refresh | g generate", "force required"},
		},
		{
			name: "busy refresh keeps wait message",
			set:  func(model *Model) { model.status = statusRefreshing },
			want: []string{"refreshing plan | controls paused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
			if tt.set != nil {
				tt.set(&model)
			}
			view := stripANSI(model.View())
			for _, want := range tt.want {
				assertContains(t, view, want)
			}
			assertNotContains(t, view, "Navigate: up/down select route, enter open, h/l switch, ? help.")
		})
	}
}

func TestModelViewShowsBootstrappedConfigSource(t *testing.T) {
	view := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "starter.json", ConfigBootstrapped: true}, nil, nil, nil).View()

	assertContains(t, view, "Source starter.json (starter config bootstrapped this run)")
	assertContains(t, view, "Created starter config. Edit project, service, entity, and basic field settings incrementally.")
}

func TestNewModelDefaultsToRouteUI(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

	if model.mode != modeWorkspace || model.screen != screenOverview || model.selectedScreen != screenOverview {
		t.Fatalf("expected route UI by default, got mode=%v screen=%v selected=%v", model.mode, model.screen, model.selectedScreen)
	}
	view := stripANSI(model.View())
	assertContains(t, view, "Routes")
	assertContains(t, view, "Route Overview/Overview")
	assertContains(t, view, "up/down route | enter open | ? help | ctrl+p routes")
	assertNotContains(t, view, "Breadcrumb: Wizard / Menu")
	assertNotContains(t, view, "What would you like to configure?")
}

func TestWizardSelectionAndRouting(t *testing.T) {
	tests := []struct {
		name          string
		moves         int
		wantMode      tuiMode
		wantWizard    wizardScreen
		wantWorkspace workspaceScreen
	}{
		{name: "project", moves: wizardConfigureProject, wantMode: modeWizard, wantWizard: wizardProject},
		{name: "services", moves: wizardConfigureServices, wantMode: modeWizard, wantWizard: wizardServices},
		{name: "review", moves: wizardReviewChanges, wantMode: modeWizard, wantWizard: wizardReview},
		{name: "generate", moves: wizardGenerateSolution, wantMode: modeWorkspace, wantWorkspace: screenGenerate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := wizardModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
			for range tt.moves {
				updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
				model = updated.(Model)
				if cmd != nil {
					t.Fatal("expected no command while selecting wizard option")
				}
			}
			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			model = updated.(Model)
			if cmd != nil || model.mode != tt.wantMode || model.wizardScreen != tt.wantWizard || (tt.wantMode == modeWorkspace && model.screen != tt.wantWorkspace) {
				t.Fatalf("expected %s route, got mode=%v wizard=%v screen=%v cmd=%v", tt.name, model.mode, model.wizardScreen, model.screen, cmd)
			}
			assertNotContains(t, stripANSI(model.View()), "Navigation")
		})
	}
}

func TestWizardEscReturnsToMenuAndQuitKeysExit(t *testing.T) {
	model := wizardModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardMenu {
		t.Fatalf("expected esc to return to wizard menu, got mode=%v screen=%v cmd=%v", model.mode, model.wizardScreen, cmd)
	}

	for _, msg := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyCtrlC}} {
		_, cmd = model.Update(msg)
		if cmd == nil {
			t.Fatalf("expected quit command for %q", msg.String())
		}
	}
}

func TestWizardProjectStepUsesExistingEditorAndContinuesToServices(t *testing.T) {
	plan := wizardPlan()
	var captured application.SolutionSettings
	model := wizardModel(plan, application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		captured = settings
		return application.UpdateSolutionSettingsResult{Saved: true, Plan: plan}, nil
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected project save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardServices || model.status != statusReady {
		t.Fatalf("expected successful project save to continue to services, got mode=%v screen=%v status=%v cmd=%v", model.mode, model.wizardScreen, model.status, cmd)
	}
	if captured.SolutionName != "CommercePlatformX" {
		t.Fatalf("expected existing project callback to receive edited name, got %#v", captured)
	}
}

func TestWizardProjectSaveFailureKeepsEditorActive(t *testing.T) {
	model := wizardModel(wizardPlan(), application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, _ application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		return application.UpdateSolutionSettingsResult{}, errors.New("config write failed")
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusEditing || model.wizardScreen != wizardProject || model.err == nil {
		t.Fatalf("expected failed save to keep project editor, got status=%v screen=%v err=%v", model.status, model.wizardScreen, model.err)
	}
	assertContains(t, model.View(), "Save failed: config write failed")
}

func TestWizardProjectStaleRefreshLocksUntilRetry(t *testing.T) {
	plan := wizardPlan()
	model := wizardModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return plan, nil
	}, nil, func(_ application.GenerateRequest, _ application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		return application.UpdateSolutionSettingsResult{Saved: true, Config: plan.Config, PlanError: errors.New("generation plan failed")}, nil
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if !model.postSaveRefreshFailed() || model.wizardScreen != wizardProject {
		t.Fatalf("expected stale project plan lock, got status=%v screen=%v context=%q", model.status, model.wizardScreen, model.errContext)
	}
	assertContains(t, model.View(), "Press r to retry the refresh before continuing.")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if cmd == nil || model.status != statusRefreshing {
		t.Fatalf("expected refresh retry command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.wizardScreen != wizardServices || model.postSaveRefreshFailed() {
		t.Fatalf("expected retry to unlock and continue, got status=%v screen=%v stale=%v", model.status, model.wizardScreen, model.postSaveRefreshFailed())
	}
}

func TestWizardProjectViewShowsTargetFrameworkAndSuggestions(t *testing.T) {
	model := wizardModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0", "net8.0"})
	model.enterWizardProject()

	view := stripANSI(model.View())
	assertContains(t, view, "Target framework: net8.0")
	assertContains(t, view, "Available target frameworks")
	assertContains(t, view, "net8.0 (current)")
	assertContains(t, view, "Edit solution name and description")
	assertContains(t, view, "Continue to services")
}

func TestProjectSettingsEditorShowsGatewayState(t *testing.T) {
	tests := []struct {
		name           string
		gatewayEnabled bool
		want           string
	}{
		{name: "default disabled", gatewayEnabled: false, want: "Gateway generation: disabled"},
		{name: "initialized enabled", gatewayEnabled: true, want: "Gateway generation: enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := wizardPlan()
			plan.Config.GatewayEnabled = tt.gatewayEnabled
			model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
			model.startEditing()

			view := stripANSI(model.View())
			assertContains(t, view, tt.want)
			if model.edit.gatewayEnabled != tt.gatewayEnabled {
				t.Fatalf("expected gateway draft %v, got %v", tt.gatewayEnabled, model.edit.gatewayEnabled)
			}
		})
	}
}

func TestProjectSettingsEditorTogglesAndSavesGatewayState(t *testing.T) {
	plan := wizardPlan()
	plan.Config.GatewayEnabled = false
	refreshedPlan := plan
	refreshedPlan.Config.GatewayEnabled = true
	var captured application.SolutionSettings
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		captured = settings
		return application.UpdateSolutionSettingsResult{Saved: true, Plan: refreshedPlan}, nil
	})
	model.startEditing()
	model.edit.focused = editFieldGatewayEnabled

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if cmd != nil || !model.edit.gatewayEnabled {
		t.Fatalf("expected space to toggle gateway without command, gateway=%v cmd=%v", model.edit.gatewayEnabled, cmd)
	}
	assertContains(t, model.View(), "Gateway generation: enabled")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected enter to save without toggling, gateway=%v status=%v cmd=%v", model.edit.gatewayEnabled, model.status, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || !model.plan.Config.GatewayEnabled {
		t.Fatalf("expected saved refresh to preserve enabled gateway, status=%v gateway=%v cmd=%v", model.status, model.plan.Config.GatewayEnabled, cmd)
	}
	if captured.GatewayEnabled == nil || *captured.GatewayEnabled != true {
		t.Fatalf("expected explicit enabled gateway setting, got %#v", captured.GatewayEnabled)
	}
	model.startEditing()
	if !model.edit.gatewayEnabled {
		t.Fatalf("expected refreshed gateway state to initialize the next edit draft as enabled")
	}
}

func TestProjectSettingsEditorSavesDisabledGatewayState(t *testing.T) {
	plan := wizardPlan()
	plan.Config.GatewayEnabled = true
	refreshedPlan := plan
	refreshedPlan.Config.GatewayEnabled = false
	var captured application.SolutionSettings
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		captured = settings
		return application.UpdateSolutionSettingsResult{Saved: true, Plan: refreshedPlan}, nil
	})
	model.startEditing()
	model.edit.focused = editFieldGatewayEnabled

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	if cmd != nil || model.edit.gatewayEnabled {
		t.Fatalf("expected space to disable gateway without command, gateway=%v cmd=%v", model.edit.gatewayEnabled, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.edit.gatewayEnabled {
		t.Fatalf("expected enter to save without re-toggling gateway, gateway=%v cmd=%v", model.edit.gatewayEnabled, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.plan.Config.GatewayEnabled {
		t.Fatalf("expected saved refresh to preserve disabled gateway, status=%v gateway=%v cmd=%v", model.status, model.plan.Config.GatewayEnabled, cmd)
	}
	if captured.GatewayEnabled == nil || *captured.GatewayEnabled != false {
		t.Fatalf("expected explicit disabled gateway setting, got %#v", captured.GatewayEnabled)
	}
}

func TestWizardProjectSuggestionChangesDraftWithoutSaving(t *testing.T) {
	called := false
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, _ application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		called = true
		return application.UpdateSolutionSettingsResult{}, nil
	}, []string{"net10.0", "net9.0", "net8.0"})
	model.enterWizardProject()
	model.wizardProjectSelection = 0

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || called || model.status != statusReady {
		t.Fatalf("expected suggestion selection to stay local, got status=%v called=%v cmd=%v", model.status, called, cmd)
	}
	if model.edit.targetFramework.string() != "net10.0" || model.plan.Config.TargetFramework != "net8.0" {
		t.Fatalf("expected draft target framework without plan save, draft=%q plan=%q", model.edit.targetFramework.string(), model.plan.Config.TargetFramework)
	}
	model.wizardProjectSelection = model.wizardProjectEditOption()
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.targetFramework.string() != "net10.0" {
		t.Fatalf("expected existing editor to preserve draft framework, got status=%v target=%q cmd=%v", model.status, model.edit.targetFramework.string(), cmd)
	}
}

func TestWizardProjectContinueSavesSelectedTargetFramework(t *testing.T) {
	plan := wizardPlan()
	plan.Config.TargetFramework = "net10.0"
	var captured application.SolutionSettings
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		captured = settings
		return application.UpdateSolutionSettingsResult{Saved: true, Plan: plan}, nil
	}, []string{"net10.0", "net9.0", "net8.0"})
	model.enterWizardProject()
	model.wizardProjectSelection = 0
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	model.wizardProjectSelection = model.wizardProjectContinueOption()

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected explicit project continue save, got status=%v cmd=%v", model.status, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardServices || captured.TargetFramework != "net10.0" {
		t.Fatalf("expected selected framework save and Services route, got screen=%v captured=%#v cmd=%v", model.wizardScreen, captured, cmd)
	}
}

func TestWizardProjectPreservesCurrentFrameworkOutsideSuggestions(t *testing.T) {
	plan := wizardPlan()
	plan.Config.TargetFramework = "net7.0"
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0"})
	model.enterWizardProject()

	view := stripANSI(model.View())
	assertContains(t, view, "Target framework: net7.0")
	assertContains(t, view, "Current framework: net7.0")
	assertNotContains(t, view, "net7.0 (current)")
	if model.wizardProjectSelection != model.wizardProjectEditOption() {
		t.Fatalf("expected selection to avoid replacing custom current framework, got %d", model.wizardProjectSelection)
	}
}

func TestWizardProjectUnsupportedCurrentFrameworkKeepsValidationErrorActionable(t *testing.T) {
	plan := wizardPlan()
	plan.Config.TargetFramework = "net0.0"
	validationErr := errors.New("generation.targetFramework must be netN.0")
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		if settings.TargetFramework != "net0.0" {
			t.Fatalf("expected unsupported current framework to reach existing validation, got %q", settings.TargetFramework)
		}
		return application.UpdateSolutionSettingsResult{}, validationErr
	}, []string{"net10.0", "net9.0"})
	model.enterWizardProject()
	model.wizardProjectSelection = model.wizardProjectContinueOption()
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusEditing || model.err != validationErr || model.edit.targetFramework.string() != "net0.0" {
		t.Fatalf("expected validation error to keep project editor active, got status=%v err=%v target=%q", model.status, model.err, model.edit.targetFramework.string())
	}
	assertContains(t, model.View(), "generation.targetFramework must be netN.0")
}

func TestWizardProjectEscReturnsToMenu(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0", "net8.0"})
	model.enterWizardProject()
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardMenu {
		t.Fatalf("expected Project esc to return to menu, got mode=%v screen=%v cmd=%v", model.mode, model.wizardScreen, cmd)
	}
}

func TestWizardServicesSelectsServiceAndPreparesContext(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardServices()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.selectedService != 1 || model.wizardScreen != wizardValueObjects {
		t.Fatalf("expected selected service to route to value objects, got service=%d screen=%v cmd=%v", model.selectedService, model.wizardScreen, cmd)
	}
	assertContains(t, model.View(), "OrderService")
}

func TestWizardEntitiesSelectionRoutesToFieldsAndBack(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardValueObjects()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardEntities {
		t.Fatalf("expected value-object skip to route to entities, got screen=%v", model.wizardScreen)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardFields || model.selectedEntity != 0 {
		t.Fatalf("expected entity selection to route to fields, got screen=%v entity=%d cmd=%v", model.wizardScreen, model.selectedEntity, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardEntities {
		t.Fatalf("expected fields esc to return to entities, got screen=%v cmd=%v", model.wizardScreen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardValueObjects {
		t.Fatalf("expected entities esc to return to value objects, got screen=%v cmd=%v", model.wizardScreen, cmd)
	}
}

func TestWizardEntityAndFieldListsExposeAddEditAdvancedEntries(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardEntities()
	view := stripANSI(model.View())
	assertContains(t, view, "Product (2 fields)")
	assertContains(t, view, "Add entity")
	assertContains(t, view, "Edit entities")
	assertContains(t, view, "Open route editor")

	for range model.wizardEntityAddOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeEntities || !model.entitiesEdit.renaming {
		t.Fatalf("expected add entity editor, got status=%v mode=%v renaming=%v cmd=%v", model.status, model.edit.mode, model.entitiesEdit.renaming, cmd)
	}

	model = NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardFields()
	view = stripANSI(model.View())
	assertContains(t, view, "Id: Guid")
	assertContains(t, view, "Add field")
	assertContains(t, view, "Edit fields")
	assertContains(t, view, "Open route editor")
	for range model.wizardFieldAddOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeFields || len(model.fieldsEdit.fields) != 3 {
		t.Fatalf("expected add field editor, got status=%v mode=%v fields=%d cmd=%v", model.status, model.edit.mode, len(model.fieldsEdit.fields), cmd)
	}

	model = NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardFields()
	for range model.wizardFieldEditOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeFields || model.fieldsEdit.selected != 0 {
		t.Fatalf("expected edit field editor, got status=%v mode=%v selected=%d cmd=%v", model.status, model.edit.mode, model.fieldsEdit.selected, cmd)
	}
}

func TestWizardGuidedViewsShowEntityAndFieldDetailsWithoutWorkspaceRail(t *testing.T) {
	plan := wizardPlan()
	plan.Config.Services[0].Entities[0].Fields[1] = application.FieldSummary{Name: "Name", Type: "ProductName"}
	plan.Config.Services[0].ValueObjects[0].RulesLabel = "required, minLength=3"
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardEntities()
	view := stripANSI(model.View())
	assertContains(t, view, "Breadcrumb: Project > Services > Value Objects > Entities")
	assertContains(t, view, "Selected service: ProductService")
	assertContains(t, view, "Field count: 2")
	assertContains(t, view, "Name: ProductName")
	assertContains(t, view, "Referenced value objects: ProductName")
	assertNotContains(t, view, "Navigation")

	model.enterWizardFields()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	view = stripANSI(model.View())
	assertContains(t, view, "Breadcrumb: Project > Services > Value Objects > Entities > Fields")
	assertContains(t, view, "ProductService/Product")
	assertContains(t, view, "Name: Name")
	assertContains(t, view, "Value object: ProductName")
	assertContains(t, view, "Rules: required, minLength=3")
	assertNotContains(t, view, "Navigation")
}

func TestWizardEntitySaveSuccessUsesExistingCallback(t *testing.T) {
	plan := wizardPlan()
	var captured application.EntitySettings
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.updateEntities = func(_ application.GenerateRequest, settings application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		captured = settings
		return application.UpdateEntitySettingsResult{Saved: true, Plan: plan}, nil
	}
	model.enterWizardEntities()
	for range model.wizardEntityEditOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected entity save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.wizardScreen != wizardEntities || len(captured.Entities) != 1 {
		t.Fatalf("expected entity save to return to guided list, got status=%v screen=%v settings=%#v", model.status, model.wizardScreen, captured)
	}
}

func TestWizardFieldSaveFailureKeepsExistingEditor(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.updateFields = func(_ application.GenerateRequest, _ application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		return application.UpdateFieldSettingsResult{}, errors.New("field write failed")
	}
	model.enterWizardFields()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusEditing || model.edit.mode != editModeFields || model.err == nil {
		t.Fatalf("expected failed field save to keep editor, got status=%v mode=%v err=%v", model.status, model.edit.mode, model.err)
	}
	assertContains(t, model.View(), "field write failed")
}

func TestWizardFieldSaveSuccessUsesExistingCallback(t *testing.T) {
	plan := wizardPlan()
	var captured application.FieldSettings
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.updateFields = func(_ application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		captured = settings
		return application.UpdateFieldSettingsResult{Saved: true, Plan: plan}, nil
	}
	model.enterWizardFields()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected field save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.wizardScreen != wizardReview || captured.ServiceName != "ProductService" || captured.EntityName != "Product" {
		t.Fatalf("expected field save to continue to review, got status=%v screen=%v settings=%#v", model.status, model.wizardScreen, captured)
	}
}

func TestRelationshipEditorSavesBoundedRelationshipSettings(t *testing.T) {
	plan := wizardPlanWithRelationships()
	var captured application.RelationshipSettings
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.updateRelationships = func(_ application.GenerateRequest, settings application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		captured = settings
		return application.UpdateRelationshipSettingsResult{Saved: true, Plan: plan}, nil
	}
	model.startRelationshipsEditing()
	model.relationshipsEdit.focused = 4
	model.relationshipsEdit.relationships[0].foreignKeyName = newTextField("CategoryId")
	model.relationshipsEdit.relationships[0].required = false

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected relationship save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusReady || len(captured.Relationships) != 1 {
		t.Fatalf("expected relationship save to complete, status=%v settings=%#v", model.status, captured)
	}
	relationship := captured.Relationships[0]
	if relationship.Multiplicity != "one-to-many" || relationship.PrincipalEntity != "Category" || relationship.DependentEntity != "Product" || relationship.ForeignKeyName != "CategoryId" || relationship.Required == nil || *relationship.Required {
		t.Fatalf("expected bounded relationship settings, got %#v", relationship)
	}
}

func TestRelationshipEditorChangesEndpointsThroughBoundedKeyPathBeforeSaving(t *testing.T) {
	plan := wizardPlanWithRelationships()
	plan.Config.Services[0].Entities = append(plan.Config.Services[0].Entities, application.EntitySummary{Name: "Supplier"})
	var captured application.RelationshipSettings
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.updateRelationships = func(_ application.GenerateRequest, settings application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		captured = settings
		return application.UpdateRelationshipSettingsResult{Saved: true, Plan: plan}, nil
	}
	model.startRelationshipsEditing()

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyTab},               // multiplicity
		{Type: tea.KeyTab},               // principal endpoint
		{Type: tea.KeySpace},             // Category -> Product
		{Type: tea.KeyTab},               // dependent endpoint
		{Type: tea.KeySpace},             // Product -> Supplier
		{Type: tea.KeyEnter, Alt: false}, // save
	} {
		updated, cmd := model.Update(key)
		model = updated.(Model)
		if cmd != nil {
			updated, _ = model.Update(cmd())
			model = updated.(Model)
		}
	}

	if model.status != statusReady || len(captured.Relationships) != 1 {
		t.Fatalf("expected endpoint edit save to complete, status=%v settings=%#v", model.status, captured)
	}
	relationship := captured.Relationships[0]
	if relationship.PrincipalEntity != "Product" || relationship.DependentEntity != "Supplier" {
		t.Fatalf("expected key path to save changed endpoints, got principal=%q dependent=%q", relationship.PrincipalEntity, relationship.DependentEntity)
	}
}

func TestRelationshipEditorRejectsUnsupportedMultiplicityBeforeSaving(t *testing.T) {
	model := NewModel(wizardPlanWithRelationships(), application.GenerateRequest{}, nil, nil, nil)
	model.updateRelationships = func(application.GenerateRequest, application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		return application.UpdateRelationshipSettingsResult{}, errors.New("relationship multiplicity \"many-to-many\" is not editable")
	}
	model.startRelationshipsEditing()
	model.relationshipsEdit.relationships[0].multiplicity = "many-to-many"

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusEditing || model.edit.mode != editModeRelationships || model.err == nil {
		t.Fatalf("expected rejected relationship save to keep editor, status=%v mode=%v err=%v", model.status, model.edit.mode, model.err)
	}
	assertContains(t, stripANSI(model.View()), "many-to-many")
}

func TestTViewRelationshipStateMapsRowsToApplicationSettings(t *testing.T) {
	state := tviewRelationshipsStateFromService(wizardPlanWithRelationships().Config.Services[0])
	if len(state.rows) != 1 || state.rows[0].principalEntity != "Category" || state.rows[0].dependentEntity != "Product" {
		t.Fatalf("expected relationship state from service summary, got %#v", state)
	}
	state.rows[0].required = false
	settings := tviewRelationshipSettingsFromState(state)
	if len(settings.Relationships) != 1 || settings.Relationships[0].Required == nil || *settings.Relationships[0].Required || settings.Relationships[0].PrincipalNavigation != "Products" {
		t.Fatalf("expected tview relationship settings mapping, got %#v", settings)
	}
}

func TestTViewEditKeyOpensRelationshipManagerFromRelationshipsRoute(t *testing.T) {
	ui := newTViewUI(wizardPlanWithRelationships(), application.GenerateRequest{}, nil, nil, noopUpdateSettings, nil, nil, nil, nil, func(application.GenerateRequest, application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		return application.UpdateRelationshipSettingsResult{}, nil
	})
	ui.open(tviewScreenRelationships)

	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	if !ui.editOpen {
		t.Fatalf("expected relationship edit key to open native tview editing")
	}
	if !ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected relationship edit to open as a modal page")
	}
	if _, ok := ui.app.GetFocus().(*tview.Table); !ok {
		t.Fatalf("expected relationship edit focus to be inside a table manager, got %T", ui.app.GetFocus())
	}
}

func TestTViewRelationshipManagerSavesEditedEndpoints(t *testing.T) {
	plan := wizardPlanWithRelationships()
	plan.Config.Services[0].Entities = append(plan.Config.Services[0].Entities, application.EntitySummary{Name: "Supplier"})
	var captured application.RelationshipSettings
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil, noopUpdateSettings, nil, nil, nil, nil, func(_ application.GenerateRequest, settings application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		captured = settings
		return application.UpdateRelationshipSettingsResult{Saved: true, Plan: plan}, nil
	})
	state := tviewRelationshipsStateFromService(plan.Config.Services[0])
	state.rows[0].principalEntity = "Product"
	state.rows[0].dependentEntity = "Supplier"

	ui.saveRelationshipsEdit(state)

	if len(captured.Relationships) != 1 {
		t.Fatalf("expected one saved relationship, got %#v", captured)
	}
	relationship := captured.Relationships[0]
	if relationship.PrincipalEntity != "Product" || relationship.DependentEntity != "Supplier" {
		t.Fatalf("expected tview relationship save to include edited endpoints, got principal=%q dependent=%q", relationship.PrincipalEntity, relationship.DependentEntity)
	}
	if !strings.Contains(ui.message, "Relationships saved") {
		t.Fatalf("expected relationship save confirmation, got %q", ui.message)
	}
}

func TestWizardFieldStaleRefreshLocksUntilRetry(t *testing.T) {
	plan := wizardPlan()
	model := NewModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return plan, nil
	}, nil, nil)
	model.updateFields = func(_ application.GenerateRequest, _ application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		return application.UpdateFieldSettingsResult{Saved: true, Config: plan.Config, PlanError: errors.New("field plan failed")}, nil
	}
	model.enterWizardFields()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if !model.postSaveRefreshFailed() || model.wizardScreen != wizardFields {
		t.Fatalf("expected stale field lock, got status=%v screen=%v context=%q", model.status, model.wizardScreen, model.errContext)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if cmd == nil || model.status != statusRefreshing {
		t.Fatalf("expected stale retry command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.postSaveRefreshFailed() || model.wizardScreen != wizardFields {
		t.Fatalf("expected retry to unlock field step, got status=%v stale=%v screen=%v", model.status, model.postSaveRefreshFailed(), model.wizardScreen)
	}
}

func TestWizardEntityStaleRefreshLocksUntilRetry(t *testing.T) {
	plan := wizardPlan()
	model := NewModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return plan, nil
	}, nil, nil)
	model.updateEntities = func(_ application.GenerateRequest, _ application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		return application.UpdateEntitySettingsResult{Saved: true, Config: plan.Config, PlanError: errors.New("entity plan failed")}, nil
	}
	model.enterWizardEntities()
	for range model.wizardEntityEditOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if !model.postSaveRefreshFailed() || model.wizardScreen != wizardEntities {
		t.Fatalf("expected stale entity lock, got status=%v screen=%v context=%q", model.status, model.wizardScreen, model.errContext)
	}
	selection := model.wizardEntitySelection
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.wizardEntitySelection != selection {
		t.Fatalf("expected stale lock to pause selection, got selection=%d cmd=%v", model.wizardEntitySelection, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if cmd == nil || model.status != statusRefreshing {
		t.Fatalf("expected stale retry command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.postSaveRefreshFailed() || model.wizardScreen != wizardEntities {
		t.Fatalf("expected retry to unlock entity step, got status=%v stale=%v screen=%v", model.status, model.postSaveRefreshFailed(), model.wizardScreen)
	}
}

func TestWizardServicesAddServiceUsesExistingEditorCallback(t *testing.T) {
	plan := wizardPlan()
	var captured application.ServiceSettings
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.updateServices = func(_ application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		captured = settings
		return application.UpdateServiceSettingsResult{Saved: true, Plan: plan}, nil
	}
	model.enterWizardServices()
	for range model.wizardServiceAddOption() {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeServices || !model.servicesEdit.renaming {
		t.Fatalf("expected add service editor, got status=%v mode=%v renaming=%v cmd=%v", model.status, model.edit.mode, model.servicesEdit.renaming, cmd)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusSaving {
		t.Fatalf("expected service save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || model.wizardScreen != wizardServices || len(captured.Services) != 3 {
		t.Fatalf("expected service callback and return to services, got status=%v screen=%v settings=%#v", model.status, model.wizardScreen, captured)
	}
}

func TestWizardGuidedViewsUseSingleColumnPromptListAndDetail(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardProject()
	view := stripANSI(model.View())
	assertContains(t, view, "Set up your project")
	assertContains(t, view, "Solution name:")
	assertContains(t, view, "Target framework: net8.0")
	assertContains(t, view, "Continue to services")
	assertNotContains(t, view, "Navigation")

	model = NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardServices()
	view = stripANSI(model.View())
	assertContains(t, view, "Which service should we configure?")
	assertContains(t, view, "Add service")
	assertContains(t, view, "Edit services")
	assertContains(t, view, "Open route editor")
	assertContains(t, view, "Selected service")
	assertContains(t, view, "Entities: 1 | Fields: 2 | Value objects: 1")
	assertNotContains(t, view, "Navigation")
}

func TestWizardAdvancedWorkspaceHasBackPath(t *testing.T) {
	model := wizardModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	for range wizardAdvancedWorkspace {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWorkspace || model.screen != screenOverview || model.guidedWorkspace {
		t.Fatalf("expected workspace, got mode=%v screen=%v guided=%v cmd=%v", model.mode, model.screen, model.guidedWorkspace, cmd)
	}
	view := stripANSI(model.View())
	assertContains(t, view, "Routes")
	assertNotContains(t, view, "Workspace")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard {
		t.Fatalf("expected esc to return from workspace, got mode=%v cmd=%v", model.mode, cmd)
	}
}

func TestWorkspaceViewsDoNotExposeAdvancedModeLanguage(t *testing.T) {
	plan := plannedFilesPlan(2)
	plan.Config.SolutionName = "CommercePlatform"
	plan.OutputDir = "/tmp/generated"
	result := application.GenerateResult{OutputDir: plan.OutputDir, Plan: plan}

	tests := []struct {
		name    string
		prepare func(*Model)
	}{
		{name: "normal workspace", prepare: func(model *Model) { model.openScreen(screenOverview) }},
		{name: "guided workspace", prepare: func(model *Model) { model.enterWizardWorkspace(screenPreview) }},
		{name: "wizard menu", prepare: func(model *Model) { model.enterWizardMenu() }},
		{name: "wizard result", prepare: func(model *Model) {
			model.mode = modeWizard
			model.wizardScreen = wizardResult
			model.status = statusGenerated
			model.result = result
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
			tt.prepare(&model)
			view := stripANSI(model.View())

			assertNotContains(t, view, "Advanced workspace")
			assertNotContains(t, view, "advanced workspace")
			assertNotContains(t, view, "advanced mode")
			assertNotContains(t, view, "Advanced configuration")
			assertNotContains(t, view, "Inspect advanced preview")
			assertNotContains(t, view, "guided setup")
		})
	}
}

func TestWideWorkspaceUsesFullScreenPaneLayout(t *testing.T) {
	model := workspaceModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.layout = layoutWide
	model.windowWidth = 120
	model.openScreen(screenServices)

	view := stripANSI(model.View())

	assertContains(t, view, "Microgen READY")
	assertContains(t, view, "Routes")
	assertContains(t, view, "> 3 Services")
	assertContains(t, view, "up/down route | enter open | ? help | ctrl+p routes")
	assertContains(t, view, "services tab context e edit r refresh")
	assertNotContains(t, view, "Main detail")
	assertNotContains(t, view, "Command / status")
	assertNotContains(t, view, "Active Overview")
	assertNotContains(t, view, "Selected Services")
	assertNotContains(t, view, "+-Sidebar")
	assertNotContains(t, view, "+-Main detail")
	assertNotContains(t, view, "+-Command / status")
}

func TestWorkspaceChromeUsesCompactSegments(t *testing.T) {
	model := workspaceModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.openScreen(screenPreview)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 125, Height: 24})
	if cmd != nil {
		t.Fatal("expected no command from window resize")
	}

	view := stripANSI(updated.(Model).View())
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Fatalf("expected 24 lines, got %d in %q", len(lines), view)
	}
	assertContains(t, lines[0], "Microgen READY")
	assertContains(t, lines[0], "Route Preview/Preview")
	assertContains(t, lines[len(lines)-1], "up/down route | enter open | ? help | ctrl+p routes")
	assertNotContains(t, view, "Primary action")
	assertNotContains(t, view, "Command / status")
	assertNotContains(t, view, "Main detail")
}

func TestWorkspaceViewUsesFixedViewportHeight(t *testing.T) {
	for _, screen := range []workspaceScreen{screenProject, screenPreview} {
		t.Run(screen.label(), func(t *testing.T) {
			model := workspaceModel(longPreviewPlan(), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
			model.openScreen(screen)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: 125, Height: 24})
			if cmd != nil {
				t.Fatal("expected no command from window resize")
			}

			view := stripANSI(updated.(Model).View())
			assertViewportSize(t, view, 125, 24)
		})
	}
}

func TestPreviewLongPathsStayInsideFixedViewport(t *testing.T) {
	model := workspaceModel(longPreviewPlan(), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	model.openScreen(screenPreview)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 125, Height: 24})
	if cmd != nil {
		t.Fatal("expected no command from window resize")
	}

	view := stripANSI(updated.(Model).View())
	assertViewportSize(t, view, 125, 24)
	assertContains(t, view, "...")
	assertNotContains(t, view, "ExtremelyLongServiceNameThatWouldPreviouslyWrapAcrossTerminalRows")
}

func TestSwitchingPreviewToProjectKeepsFixedViewportHeight(t *testing.T) {
	model := workspaceModel(longPreviewPlan(), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 125, Height: 24})
	model = updated.(Model)
	model.openScreen(screenPreview)
	assertViewportSize(t, stripANSI(model.View()), 125, 24)

	model.openScreen(screenProject)
	assertViewportSize(t, stripANSI(model.View()), 125, 24)
}

func TestOverlayViewportLinesPreservesDimensionsAndClamps(t *testing.T) {
	tests := []struct {
		name       string
		base       []string
		modal      string
		width      int
		wantHeight int
		want       []string
	}{
		{
			name:       "centers modal without changing viewport",
			base:       []string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccccccc", "dddddddddd", "eeeeeeeeee"},
			modal:      "XX\nYY",
			width:      10,
			wantHeight: 5,
			want:       []string{"aaaaaaaaaa", "    XX    ", "    YY    ", "dddddddddd", "eeeeeeeeee"},
		},
		{
			name:       "clamps oversized modal inside compact viewport",
			base:       []string{"aaaaaa", "bbbbbb"},
			modal:      "one\ntwo\nthree",
			width:      6,
			wantHeight: 2,
			want:       []string{" one  ", " two  "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overlayViewportLines(tt.base, tt.modal, tt.width)
			if len(got) != tt.wantHeight {
				t.Fatalf("expected %d lines, got %d in %#v", tt.wantHeight, len(got), got)
			}
			for index, line := range got {
				if visible := len([]rune(stripANSI(line))); visible != tt.width {
					t.Fatalf("expected line %d width %d, got %d in %q", index+1, tt.width, visible, line)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("overlayViewportLines() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestActiveModalSelectsVisibleWorkspaceOverlay(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.routeSelectorOpen = true
	model.routeSelectorScreen = screenProject
	if modal := stripANSI(model.activeModal()); !strings.Contains(modal, "Routes") || !strings.Contains(modal, "target Project") {
		t.Fatalf("expected route selector modal with target context, got %q", modal)
	}

	model.helpOpen = true
	if modal := stripANSI(model.activeModal()); !strings.Contains(modal, "Keys") || strings.Contains(modal, "target Project") {
		t.Fatalf("expected help modal to take precedence over route selector, got %q", modal)
	}
}

func TestWorkspaceModalsOverlayFixedViewport(t *testing.T) {
	tests := []struct {
		name      string
		open      func(Model) Model
		wantModal string
	}{
		{
			name: "help overlay",
			open: func(model Model) Model {
				updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
				return updated.(Model)
			},
			wantModal: "Keys",
		},
		{
			name: "route overlay",
			open: func(model Model) Model {
				updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
				return updated.(Model)
			},
			wantModal: "Routes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(longPreviewPlan(), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
			updated, _ := model.Update(tea.WindowSizeMsg{Width: 96, Height: 18})
			model = tt.open(updated.(Model))

			view := stripANSI(model.View())
			assertViewportSize(t, view, 96, 18)
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			footer := strings.TrimSpace(lines[len(lines)-1])
			if !strings.Contains(footer, "up/down route") {
				t.Fatalf("expected footer on final line, got %q in %q", footer, view)
			}
			modalLine := lineIndexContaining(lines, tt.wantModal)
			if modalLine < 0 || modalLine >= len(lines)-1 {
				t.Fatalf("expected %q before footer inside viewport, got index=%d in %q", tt.wantModal, modalLine, view)
			}
			assertNotContains(t, strings.Join(lines[modalLine+1:len(lines)-1], "\n"), "Microgen READY")
		})
	}
}

func TestWorkspaceModalsPreserveCompactHeight(t *testing.T) {
	model := workspaceModel(longPreviewPlan(), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 54, Height: 9})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = updated.(Model)

	view := stripANSI(model.View())
	assertViewportSize(t, view, 54, 9)
	assertContains(t, view, "Routes")
	assertContains(t, strings.Split(strings.TrimRight(view, "\n"), "\n")[8], "up/down route")
}

func TestWorkspaceModalOutputUsesNeutralPaneWording(t *testing.T) {
	tests := []struct {
		name string
		open func(Model) Model
	}{
		{
			name: "help",
			open: func(model Model) Model {
				updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
				return updated.(Model)
			},
		},
		{
			name: "route selector",
			open: func(model Model) Model {
				updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
				return updated.(Model)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
			updated, _ := model.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
			view := strings.ToLower(stripANSI(tt.open(updated.(Model)).View()))
			for _, unwanted := range []string{"source product", "external product", "command / status", "advanced"} {
				assertNotContains(t, view, unwanted)
			}
		})
	}
}

func TestGuidedWorkspaceUsesUnifiedWorkspaceShell(t *testing.T) {
	model := workspaceModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardWorkspace(screenServices)
	view := stripANSI(model.View())

	assertContains(t, view, "Microgen READY")
	assertContains(t, view, "Routes")
	assertContains(t, view, "Route Services/Services")
	assertContains(t, view, "esc back")
	assertNotContains(t, view, "Command / status")
	assertNotContains(t, view, "Breadcrumb: Wizard")
	assertNotContains(t, view, "guided setup")
}

func TestGuidedWorkspaceEscReturnsWithNeutralWordingAndPreservesState(t *testing.T) {
	plan := wizardPlan()
	plan.Config.SolutionName = "CommercePlatform"
	model := workspaceModel(plan, application.GenerateRequest{ConfigPath: "microgen.json", Force: true}, nil, nil, nil)
	model.enterWizardReview()
	model.enterWizardWorkspace(screenPreview)
	model.message = "Review the route preview before generating."

	view := stripANSI(model.View())
	assertContains(t, view, "esc back")
	assertContains(t, view, "Review the route preview before generating.")
	assertContains(t, view, "esc back")
	assertNotContains(t, view, "advanced")
	assertNotContains(t, view, "guided setup")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardReview {
		t.Fatalf("expected esc to return to guided review, got mode=%v screen=%v cmd=%v", model.mode, model.wizardScreen, cmd)
	}
	if model.plan.Config.SolutionName != "CommercePlatform" || model.request.ConfigPath != "microgen.json" || !model.request.Force || model.status != statusReady {
		t.Fatalf("expected guided return to preserve config/request/status, got plan=%#v request=%#v status=%v err=%v", model.plan.Config, model.request, model.status, model.err)
	}
}

func TestUnifiedWorkspaceSafetyStatesRemainVisibleAndProtected(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(*Model)
		msg        tea.KeyMsg
		wantView   []string
		wantStatus modelStatus
		wantScreen workspaceScreen
	}{
		{
			name: "busy generation blocks quit and keeps wait text",
			prepare: func(model *Model) {
				model.status = statusGenerating
				model.openScreen(screenGenerate)
			},
			msg:        tea.KeyMsg{Type: tea.KeyEsc},
			wantView:   []string{"Generating files. Please wait", "generating files | controls paused"},
			wantStatus: statusGenerating,
			wantScreen: screenGenerate,
		},
		{
			name: "stale refresh stays locked until retry",
			prepare: func(model *Model) {
				model.status = statusFailed
				model.errContext = "Refresh after save"
				model.err = errors.New("refresh failed")
				model.openScreen(screenGenerate)
			},
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}},
			wantView:   []string{"locked | r retry refresh", "route selector unavailable until refresh succeeds"},
			wantStatus: statusFailed,
			wantScreen: screenGenerate,
		},
		{
			name: "force required blocks generation",
			prepare: func(model *Model) {
				model.plan.ForceRequired = true
				model.openScreen(screenGenerate)
			},
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}},
			wantView:   []string{"Generation is locked until --force is confirmed", "Confirm --force or change the output directory"},
			wantStatus: statusReady,
			wantScreen: screenGenerate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(wizardPlan(), application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
				t.Fatal("generation must remain protected")
				return application.GenerateResult{}, nil
			}, nil)
			tt.prepare(&model)
			updated, cmd := model.Update(tt.msg)
			model = updated.(Model)
			if cmd != nil || model.status != tt.wantStatus || model.activeScreen() != tt.wantScreen {
				t.Fatalf("expected protected state status=%v screen=%v without command, got status=%v screen=%v cmd=%v", tt.wantStatus, tt.wantScreen, model.status, model.activeScreen(), cmd)
			}
			view := stripANSI(model.View())
			for _, want := range tt.wantView {
				assertContains(t, view, want)
			}
		})
	}
}

func TestGuidedServicesPreserveSelectedContext(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardWorkspace(screenServices)
	model.serviceContext = serviceResourceValueObjects
	model.selectedService = 2
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.screen != screenServices || model.serviceContext != serviceResourceValueObjects || model.selectedService != 2 {
		t.Fatalf("expected Services context to be preserved, got screen=%v context=%v service=%d", model.screen, model.serviceContext, model.selectedService)
	}
}

func TestGuidedGenerationReturnsMinimalResultWizard(t *testing.T) {
	plan := plannedFilesPlan(2)
	plan.OutputDir = "/tmp/generated"
	request := application.GenerateRequest{OutputDir: plan.OutputDir}
	model := wizardModel(plan, request, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
		return application.GenerateResult{OutputDir: actual.OutputDir, Plan: plan}, nil
	}, nil)
	for range wizardGenerateSolution {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if cmd == nil || model.status != statusGenerating {
		t.Fatalf("expected explicit generation command, got status=%v cmd=%v", model.status, cmd)
	}
	finished := cmd().(generationFinishedMsg)
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardResult {
		t.Fatalf("expected wizard result screen, got mode=%v screen=%v cmd=%v", model.mode, model.wizardScreen, cmd)
	}
	view := stripANSI(model.View())
	assertContains(t, view, "Generation complete")
	assertContains(t, view, "2 files written to /tmp/generated")
	assertContains(t, view, "Back to menu")
	assertContains(t, view, "Open route editor")
	assertNotContains(t, view, "Navigation")
}

func TestWizardFieldsContinueToReviewAndEscBacksToFields(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardFields()
	for range model.wizardFieldContinueOption() {
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
		if cmd != nil {
			t.Fatal("expected no command while selecting field continuation")
		}
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardReview {
		t.Fatalf("expected Fields completion to open review, got screen=%v cmd=%v", model.wizardScreen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardFields {
		t.Fatalf("expected review esc to return to Fields, got screen=%v cmd=%v", model.wizardScreen, cmd)
	}
}

func TestWizardValueObjectsSkipConfigureAndEntities(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
		model.enterWizardValueObjects()
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if cmd != nil || model.wizardScreen != wizardEntities {
			t.Fatalf("expected skip to entities, got screen=%v cmd=%v", model.wizardScreen, cmd)
		}
	})

	t.Run("configure and edit", func(t *testing.T) {
		plan := wizardPlan()
		model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
		model.updateValueObjects = func(_ application.GenerateRequest, _ application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			return application.UpdateValueObjectSettingsResult{Saved: true, Plan: plan}, nil
		}
		model.enterWizardValueObjects()
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if !model.wizardValueObjectConfigured || model.wizardValueObjectSelection != 0 {
			t.Fatalf("expected configure selection to open value-object list, got configured=%v selection=%d", model.wizardValueObjectConfigured, model.wizardValueObjectSelection)
		}
		assertContains(t, model.View(), "ProductName: string")
		assertContains(t, model.View(), "Rules: no rules")
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if cmd != nil || model.status != statusEditing || model.edit.mode != editModeValueObjects {
			t.Fatalf("expected selected value object editor, got status=%v mode=%v cmd=%v", model.status, model.edit.mode, cmd)
		}
		assertContains(t, model.View(), "Editing value objects for ProductService")
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if cmd == nil || model.status != statusSaving {
			t.Fatalf("expected value-object save command, got status=%v cmd=%v", model.status, cmd)
		}
		updated, cmd = model.Update(cmd())
		model = updated.(Model)
		if cmd != nil || model.status != statusReady || model.wizardScreen != wizardEntities {
			t.Fatalf("expected value-object save to return to Entities, got status=%v screen=%v cmd=%v", model.status, model.wizardScreen, cmd)
		}
	})

	t.Run("configured list continues", func(t *testing.T) {
		model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
		model.enterWizardValueObjects()
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		for range model.wizardValueObjectReviewOption() {
			updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
			model = updated.(Model)
		}
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if cmd != nil || model.wizardScreen != wizardEntities {
			t.Fatalf("expected configured value-object list to continue to entities, got screen=%v cmd=%v", model.wizardScreen, cmd)
		}
	})
}

func TestWizardValueObjectChoiceShowsConfigureAndSkipWithoutExistingValueObjects(t *testing.T) {
	plan := wizardPlan()
	plan.Config.Services[0].ValueObjects = nil
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardValueObjects()

	view := stripANSI(model.View())
	assertContains(t, view, "Current value objects: 0")
	assertContains(t, view, "Configure value objects")
	assertContains(t, view, "Skip to entities")
}

func TestWizardBackNavigationFollowsValueObjectsBeforeEntities(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardServices()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardValueObjects {
		t.Fatalf("expected services to open value objects, got %v", model.wizardScreen)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.wizardScreen != wizardServices {
		t.Fatalf("expected value objects esc to return to services, got %v", model.wizardScreen)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardEntities {
		t.Fatalf("expected value objects skip to open entities, got %v", model.wizardScreen)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.wizardScreen != wizardValueObjects {
		t.Fatalf("expected entities esc to return to value objects, got %v", model.wizardScreen)
	}
}

func TestWizardFieldsCanReferenceValueObjectDefinedBeforeFields(t *testing.T) {
	plan := wizardPlan()
	plan.Config.Services[0].Entities[0].Fields[1] = application.FieldSummary{Name: "Name", Type: "ProductName"}
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardServices()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardEntities {
		t.Fatalf("expected value-object skip to open entities, got %v", model.wizardScreen)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardFields {
		t.Fatalf("expected entity selection to open fields, got %v", model.wizardScreen)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	view := stripANSI(model.View())
	assertContains(t, view, "Value object: ProductName")
	assertNotContains(t, view, "Configure value objects")

	for range model.wizardFieldContinueOption() - model.wizardFieldSelection {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.wizardScreen != wizardReview {
		t.Fatalf("expected fields completion to open review without another value-object step, got %v", model.wizardScreen)
	}
}

func TestWizardReviewRoutesToExplicitGenerateAndResult(t *testing.T) {
	plan := wizardPlan()
	plan.OutputDir = "/tmp/generated"
	plan.FileCount = 2
	plan.Files = []application.PlannedFile{{Path: "one.cs", Action: "create"}, {Path: "two.cs", Action: "replace"}}
	plan.Readiness = application.ReadinessSummary{ProjectPresent: true, Hints: []string{"Review output before writing."}}
	called := false
	model := NewModel(plan, application.GenerateRequest{OutputDir: plan.OutputDir}, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
		called = true
		return application.GenerateResult{OutputDir: actual.OutputDir, Plan: plan, Warning: "check generated tests"}, nil
	}, nil)
	model.enterWizardReview()
	view := stripANSI(model.View())
	assertContains(t, view, "Review your generation plan")
	assertContains(t, view, "Solution: CommercePlatform")
	assertContains(t, view, "Services: 2 | Entities: 2 | Fields: 3 | Value objects: 1")
	assertContains(t, view, "Changes: created=1 | replaced=1 | unchanged=0 | deleted=0")
	assertNotContains(t, view, "Navigation")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWorkspace || model.screen != screenGenerate || called {
		t.Fatalf("expected Review to open explicit Generate without writing, got mode=%v screen=%v called=%v cmd=%v", model.mode, model.screen, called, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil || model.status != statusGenerating {
		t.Fatalf("expected Generate Enter confirmation to start writing, got status=%v cmd=%v", model.status, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardResult || model.status != statusGenerated {
		t.Fatalf("expected successful generation Result, got mode=%v screen=%v status=%v cmd=%v", model.mode, model.wizardScreen, model.status, cmd)
	}
	view = stripANSI(model.View())
	assertContains(t, view, "Output directory: /tmp/generated")
	assertContains(t, view, "Impact: created=1, replaced=1, unchanged=0")
	assertContains(t, view, "dotnet build && dotnet test")
	assertContains(t, view, "Warning: check generated tests")
	assertNotContains(t, view, "Navigation")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard || model.wizardScreen != wizardMenu {
		t.Fatalf("expected Result esc to return to menu, got mode=%v screen=%v cmd=%v", model.mode, model.wizardScreen, cmd)
	}
}

func TestWizardGenerationFailureResultAndSafetyBlocks(t *testing.T) {
	generationErr := errors.New("write failed")
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		return application.GenerateResult{}, generationErr
	}, nil)
	model.enterWizardReview()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.wizardScreen != wizardResult || model.status != statusFailed {
		t.Fatalf("expected failed generation Result, got screen=%v status=%v cmd=%v", model.wizardScreen, model.status, cmd)
	}
	assertContains(t, model.View(), "FAILED Generation failed: write failed")
	assertContains(t, model.View(), "No generated result was published.")

	for _, test := range []struct {
		name    string
		prepare func(*Model)
		want    string
	}{{
		name: "stale plan",
		prepare: func(model *Model) {
			model.status = statusFailed
			model.errContext = "Refresh after save"
		},
		want: "Readiness is stale. Saved settings need a successful plan refresh before generation.",
	}, {
		name:    "force required",
		prepare: func(model *Model) { model.plan.ForceRequired = true },
		want:    "Generation is locked until --force is confirmed",
	}} {
		t.Run(test.name, func(t *testing.T) {
			blocked := NewModel(wizardPlan(), application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
				t.Fatal("generation should remain blocked")
				return application.GenerateResult{}, nil
			}, nil)
			blocked.enterWizardWorkspace(screenGenerate)
			test.prepare(&blocked)
			updated, command := blocked.Update(tea.KeyMsg{Type: tea.KeyEnter})
			blocked = updated.(Model)
			if command != nil {
				t.Fatal("expected no command for blocked generation")
			}
			assertContains(t, blocked.View(), test.want)
		})
	}
}

func TestWizardValueObjectsAndReviewViewsStaySingleColumn(t *testing.T) {
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil)
	model.enterWizardValueObjects()
	view := stripANSI(model.View())
	assertContains(t, view, "Breadcrumb: Project > Services > Value Objects")
	assertContains(t, view, "Would you like to configure value objects before entities and fields?")
	assertContains(t, view, "Configure value objects")
	assertContains(t, view, "Skip to entities")
	assertContains(t, view, "Open route editor")
	assertNotContains(t, view, "Navigation")

	model.enterWizardReview()
	view = stripANSI(model.View())
	assertContains(t, view, "Review your generation plan")
	assertContains(t, view, "Generate solution")
	assertContains(t, view, "Inspect route preview")
	assertContains(t, view, "Back to fields")
	assertNotContains(t, view, "Navigation")
}

func TestRunUsesTViewApplication(t *testing.T) {
	original := runTViewApplication
	t.Cleanup(func() { runTViewApplication = original })
	called := false
	runTViewApplication = func(app *tview.Application, root tview.Primitive) error {
		called = app != nil && root != nil
		return nil
	}

	if err := Run(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected Run to start a tview application")
	}
}

func TestNewTViewUIWiresNativeEditCallbacks(t *testing.T) {
	updateSettings := func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
		return application.UpdateSolutionSettingsResult{}, nil
	}
	updateServices := func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		return application.UpdateServiceSettingsResult{}, nil
	}
	updateEntities := func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		return application.UpdateEntitySettingsResult{}, nil
	}
	updateFields := func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		return application.UpdateFieldSettingsResult{}, nil
	}
	updateValueObjects := func(application.GenerateRequest, application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		return application.UpdateValueObjectSettingsResult{}, nil
	}
	updateRelationships := func(application.GenerateRequest, application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		return application.UpdateRelationshipSettingsResult{}, nil
	}

	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, updateSettings, updateServices, updateEntities, updateFields, updateValueObjects, updateRelationships, []string{"net10.0"})

	if ui.updateSettings == nil || ui.updateServices == nil || ui.updateEntities == nil || ui.updateFields == nil || ui.updateValueObjects == nil || ui.updateRelationships == nil {
		t.Fatalf("expected all native tview edit callbacks to be wired")
	}
	if len(ui.targetFrameworkSuggestions) != 1 || ui.targetFrameworkSuggestions[0] != "net10.0" {
		t.Fatalf("expected target framework suggestions to remain wired, got %v", ui.targetFrameworkSuggestions)
	}
}

func TestTViewGenerateHonorsForceLock(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{ForceRequired: true}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		t.Fatalf("generate should not run while force is required")
		return application.GenerateResult{}, nil
	})

	ui.generateFiles()

	if ui.screen != tviewScreenGenerate {
		t.Fatalf("expected generate screen, got %d", ui.screen)
	}
	if !strings.Contains(ui.message, "Generation is locked") {
		t.Fatalf("expected force lock message, got %q", ui.message)
	}
}

func TestTViewGenerateStartsAsynchronously(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	updates := make(chan func(), 1)
	var calls atomic.Int32
	plan := application.GenerationPlan{FileCount: 1}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		calls.Add(1)
		close(started)
		<-release
		return application.GenerateResult{Plan: plan, OutputDir: "out"}, nil
	})
	ui.queueUpdateDraw = func(fn func()) { updates <- fn }
	ui.open(tviewScreenGenerate)

	returned := make(chan struct{})
	go func() {
		ui.generateFiles()
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("generateFiles blocked while generation was running")
	}
	if !ui.generating {
		t.Fatal("expected generation to be marked as running")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("generate was not started")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one generation call, got %d", calls.Load())
	}

	close(release)
	select {
	case update := <-updates:
		update()
	case <-time.After(time.Second):
		t.Fatal("generation completion was not queued")
	}
	if ui.generating || ui.screen != tviewScreenResult {
		t.Fatalf("expected generation completion to open Result, generating=%v screen=%d", ui.generating, ui.screen)
	}
}

func TestTViewGenerateKeyDoesNotDoubleStartWhileRunning(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	updates := make(chan func(), 1)
	var calls atomic.Int32
	plan := application.GenerationPlan{FileCount: 1}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		calls.Add(1)
		close(started)
		<-release
		return application.GenerateResult{Plan: plan, OutputDir: "out"}, nil
	})
	ui.queueUpdateDraw = func(fn func()) { updates <- fn }
	ui.open(tviewScreenGenerate)

	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("generate was not started")
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone))
	if calls.Load() != 1 {
		t.Fatalf("expected one generation call while running, got %d", calls.Load())
	}
	if !strings.Contains(ui.message, "already running") {
		t.Fatalf("expected running message, got %q", ui.message)
	}

	close(release)
	select {
	case update := <-updates:
		update()
	case <-time.After(time.Second):
		t.Fatal("generation completion was not queued")
	}
	if calls.Load() != 1 || ui.screen != tviewScreenResult {
		t.Fatalf("expected one completed generation, calls=%d screen=%d", calls.Load(), ui.screen)
	}
}

func TestTViewQuitKeysWaitWhileGenerating(t *testing.T) {
	tests := []struct {
		name  string
		event *tcell.EventKey
	}{
		{name: "q", event: tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)},
		{name: "ctrl+c", event: tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stops atomic.Int32
			ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil)
			ui.stopApp = func() { stops.Add(1) }
			ui.generating = true

			if got := ui.handleKey(tt.event); got != nil {
				t.Fatalf("expected quit key to be handled, got %#v", got)
			}
			if stops.Load() != 0 {
				t.Fatalf("expected app not to stop while generating, got %d stops", stops.Load())
			}
			if !strings.Contains(ui.message, "Generation is running") || !strings.Contains(ui.message, "Wait for completion") {
				t.Fatalf("expected wait message while generating, got %q", ui.message)
			}
		})
	}
}

func TestTViewQuitKeysStopWhenNotGenerating(t *testing.T) {
	tests := []struct {
		name  string
		event *tcell.EventKey
	}{
		{name: "q", event: tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)},
		{name: "ctrl+c", event: tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stops atomic.Int32
			ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil)
			ui.stopApp = func() { stops.Add(1) }

			if got := ui.handleKey(tt.event); got != nil {
				t.Fatalf("expected quit key to be handled, got %#v", got)
			}
			if stops.Load() != 1 {
				t.Fatalf("expected app to stop once, got %d stops", stops.Load())
			}
		})
	}
}

func TestTViewGenerateKeyRoutesBeforeGenerating(t *testing.T) {
	generated := make(chan struct{}, 1)
	completed := make(chan struct{}, 1)
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		generated <- struct{}{}
		return application.GenerateResult{Plan: application.GenerationPlan{FileCount: 1}, OutputDir: "out"}, nil
	})
	ui.queueUpdateDraw = func(fn func()) {
		fn()
		completed <- struct{}{}
	}

	ui.open(tviewScreenProject)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone))
	if len(generated) != 0 || ui.screen != tviewScreenGenerate {
		t.Fatalf("expected first g to route to Generate without generating, generated=%d screen=%d", len(generated), ui.screen)
	}

	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone))
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("expected second g to complete generation")
	}
	if len(generated) != 1 || ui.screen != tviewScreenResult {
		t.Fatalf("expected second g to generate and show Result, generated=%d screen=%d", len(generated), ui.screen)
	}
}

func TestTViewEditKeyOpensNativeProjectForm(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, noopUpdateSettings)
	ui.open(tviewScreenProject)

	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	if !ui.editOpen {
		t.Fatalf("expected edit key to open native tview editing")
	}
	if !ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected project edit to open as a modal page")
	}
	if _, ok := ui.app.GetFocus().(*tview.Table); !ok {
		t.Fatalf("expected project edit focus to be on the manager table, got %T", ui.app.GetFocus())
	}
}

func TestTViewEditModalClosesWithEscape(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, noopUpdateSettings)
	ui.open(tviewScreenProject)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	ui.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))

	if ui.editOpen {
		t.Fatalf("expected escape to close edit modal")
	}
	if ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected edit modal page to be removed")
	}
	if ui.app.GetFocus() != ui.sidebar {
		t.Fatalf("expected focus to return to dashboard, got %T", ui.app.GetFocus())
	}
}

func TestTViewServicesEditOpensModal(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{Config: application.ConfigSummary{ServiceNames: []string{"ProductService"}}}, application.GenerateRequest{}, nil, nil)
	ui.open(tviewScreenServices)

	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	if !ui.editOpen {
		t.Fatalf("expected services edit to stay in tview")
	}
	if !ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected services edit to open as a modal page")
	}
	if _, ok := ui.app.GetFocus().(*tview.Table); !ok {
		t.Fatalf("expected services edit focus to be inside a table manager, got %T", ui.app.GetFocus())
	}
}

func TestTViewEditModalTabsBetweenManagerAndActions(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, noopUpdateSettings)
	ui.open(tviewScreenProject)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	tab := tcell.NewEventKey(tcell.KeyTAB, 0, tcell.ModNone)
	if got := ui.handleKey(tab); got != nil {
		t.Fatalf("expected tab to cycle modal focus, got %v", got)
	}
	if ui.modalFocusIndex != 1 {
		t.Fatalf("expected tab to move focus to modal actions, got index %d and focus %T", ui.modalFocusIndex, ui.app.GetFocus())
	}
	if ui.focus != tviewFocusSidebar {
		t.Fatalf("expected modal tab to leave outer panel focus unchanged, got %d", ui.focus)
	}

	backtab := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModShift)
	if got := ui.handleKey(backtab); got != nil {
		t.Fatalf("expected shift+tab to cycle modal focus, got %v", got)
	}
	if _, ok := ui.app.GetFocus().(*tview.Table); !ok {
		t.Fatalf("expected shift+tab to return focus to manager table, got %T", ui.app.GetFocus())
	}
}

func TestTViewEditInputFieldsUseCompactModalWidth(t *testing.T) {
	form := tview.NewForm()

	addEditInputField(form, "Description", "A longer project description")

	field := form.GetFormItem(0).(*tview.InputField)
	if field.GetFieldWidth() != tviewEditModalInputWidth {
		t.Fatalf("expected compact modal input width %d, got %d", tviewEditModalInputWidth, field.GetFieldWidth())
	}
}

func TestTViewProjectEditSavesAndRefreshesPlan(t *testing.T) {
	updated := application.SolutionSettings{}
	refreshes := 0
	refreshedPlan := application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce", SolutionDescription: "Updated", TargetFramework: "net9.0", SolutionFormat: "slnx", GatewayEnabled: true}, FileCount: 3}
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{SolutionName: " Commerce ", SolutionDescription: " Updated ", TargetFramework: " net9.0 ", SolutionFormat: " slnx ", GatewayEnabled: true}},
		application.GenerateRequest{ConfigPath: "microgen.json"},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			refreshes++
			return refreshedPlan, nil
		},
		nil,
		func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
			updated = settings
			return application.UpdateSolutionSettingsResult{Plan: refreshedPlan}, nil
		},
	)
	ui.open(tviewScreenProject)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))

	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone))

	if updated.SolutionName != "Commerce" || updated.SolutionDescription != "Updated" || updated.TargetFramework != "net9.0" || updated.SolutionFormat != "slnx" {
		t.Fatalf("unexpected project settings: %+v", updated)
	}
	if updated.GatewayEnabled == nil || !*updated.GatewayEnabled {
		t.Fatalf("expected gateway setting to be saved, got %+v", updated.GatewayEnabled)
	}
	if refreshes != 1 {
		t.Fatalf("expected plan refresh after save, got %d", refreshes)
	}
	if ui.plan.Config.SolutionName != "Commerce" || ui.plan.FileCount != 3 {
		t.Fatalf("expected refreshed plan to be shown, got %+v", ui.plan)
	}
	if ui.editOpen {
		t.Fatalf("expected edit form to close after save")
	}
}

func TestTViewServicesEditSavesRenamesAndRefreshesPlan(t *testing.T) {
	var updated application.ServiceSettings
	refreshedPlan := application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce", ServiceNames: []string{"CatalogService", "ShippingService"}, ServiceCount: 2}, FileCount: 4}
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce", ServiceNames: []string{"ProductService", "OrderService"}, ServiceCount: 2}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) { return refreshedPlan, nil },
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
			updated = settings
			return application.UpdateServiceSettingsResult{Plan: refreshedPlan}, nil
		},
	)
	state := &tviewServicesEditState{rows: []tviewManagerRow{
		{original: "ProductService", name: " CatalogService "},
		{original: "OrderService", name: " OrderService ", deleted: true},
		{name: " ShippingService "},
	}}

	ui.saveServicesEdit(state)

	if len(updated.Services) != 2 {
		t.Fatalf("expected two service settings, got %+v", updated.Services)
	}
	if updated.Services[0] != (application.ServiceNameSetting{OriginalName: "ProductService", Name: "CatalogService"}) {
		t.Fatalf("unexpected first service setting: %+v", updated.Services[0])
	}
	if updated.Services[1] != (application.ServiceNameSetting{Name: "ShippingService"}) {
		t.Fatalf("unexpected second service setting: %+v", updated.Services[1])
	}
	if ui.plan.Config.ServiceNames[0] != "CatalogService" || ui.plan.FileCount != 4 {
		t.Fatalf("expected refreshed service plan, got %+v", ui.plan)
	}
}

func TestTViewServicesManagerSavesMultipleNewServices(t *testing.T) {
	var updated application.ServiceSettings
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{ServiceNames: []string{"ProductService"}}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{}, nil
		},
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
			updated = settings
			return application.UpdateServiceSettingsResult{}, nil
		},
	)

	ui.saveServicesEdit(&tviewServicesEditState{rows: []tviewManagerRow{
		{original: "ProductService", name: "ProductService"},
		{name: "OrderService"},
		{name: "ShippingService"},
	}})

	if len(updated.Services) != 3 {
		t.Fatalf("expected all service rows to be submitted before closing, got %+v", updated.Services)
	}
	if updated.Services[1] != (application.ServiceNameSetting{Name: "OrderService"}) || updated.Services[2] != (application.ServiceNameSetting{Name: "ShippingService"}) {
		t.Fatalf("unexpected new service settings: %+v", updated.Services)
	}
}

func TestTViewServicesManagerKeepsActionsInTableAndFooterSticky(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{ServiceNames: []string{"ProductService", "OrderService", "ShippingService", "BillingService", "CatalogService", "IdentityService"}}}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)

	ui.openServicesEdit()

	manager, ok := ui.app.GetFocus().(*tview.Table)
	if !ok {
		t.Fatalf("expected services manager table focus, got %T", ui.app.GetFocus())
	}
	if got := manager.GetCell(0, 1).Text; got != "Actions" {
		t.Fatalf("expected right-side actions column, got %q", got)
	}
	if got := manager.GetCell(1, 1).Text; got != "Delete" {
		t.Fatalf("expected delete action in the row action column, got %q", got)
	}
}

func TestTViewManagerRowsUseReadableSelectedStyles(t *testing.T) {
	manager := tview.NewTable().SetSelectable(true, true).SetFixed(1, 0)
	manager.SetCell(0, 0, tview.NewTableCell("Name"))
	manager.SetCell(0, 1, tview.NewTableCell("Actions"))
	renderManagerRows(manager, []tviewManagerRow{{name: "Product"}}, false, "")

	for _, cell := range []*tview.TableCell{manager.GetCell(1, 0), manager.GetCell(1, 1)} {
		selectedForeground, selectedBackground, _ := cell.SelectedStyle.Decompose()
		if selectedForeground != tcell.ColorWhite || selectedBackground != tcell.ColorTeal {
			t.Fatalf("expected selected row cell to be readable white on teal, got fg=%v bg=%v", selectedForeground, selectedBackground)
		}
	}

	foreground, _, _ := manager.GetCell(1, 1).Style.Decompose()
	if foreground != tcell.ColorYellow {
		t.Fatalf("expected non-selected action text to stay yellow, got %v", foreground)
	}
}

func sendKeyToTViewFocus(ui *tviewUI, event *tcell.EventKey) {
	focus := ui.app.GetFocus()
	if focus == nil || focus.InputHandler() == nil {
		return
	}
	focus.InputHandler()(event, func(primitive tview.Primitive) { ui.app.SetFocus(primitive) })
}

func noopUpdateSettings(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
	return application.UpdateSolutionSettingsResult{}, nil
}

func TestTViewEntityEditSavesSelectedServiceEntitiesAndFieldsNatively(t *testing.T) {
	var entitySettings application.EntitySettings
	var fieldSettings application.FieldSettings
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{
			{Name: "ProductService", Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Name", Type: "string"}}}}},
			{Name: "OrderService", Entities: []application.EntitySummary{{Name: "Order", Fields: []application.FieldSummary{{Name: "Total", Type: "decimal"}}}}},
		}}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{FileCount: 2}, nil
		},
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
			entitySettings = settings
			return application.UpdateEntitySettingsResult{Plan: application.GenerationPlan{FileCount: 1}}, nil
		},
		func(_ application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			fieldSettings = settings
			return application.UpdateFieldSettingsResult{Plan: application.GenerationPlan{FileCount: 2}}, nil
		},
	)
	ui.saveEntitiesEdit(&tviewEntitiesEditState{serviceName: "OrderService", rows: []tviewManagerRow{
		{original: "Order", name: " PurchaseOrder "},
		{name: " Invoice "},
	}})
	ui.saveFieldsEdit(&tviewFieldsEditState{serviceName: "OrderService", entityName: "PurchaseOrder", rows: []tviewManagerRow{
		{original: "Total", name: " Amount ", typeName: " Money "},
		{name: " Currency ", typeName: " string "},
	}})

	if entitySettings.ServiceName != "OrderService" {
		t.Fatalf("unexpected entity settings: %+v", entitySettings)
	}
	if len(entitySettings.Entities) != 2 || entitySettings.Entities[0] != (application.EntityNameSetting{OriginalName: "Order", Name: "PurchaseOrder"}) || entitySettings.Entities[1] != (application.EntityNameSetting{Name: "Invoice"}) {
		t.Fatalf("unexpected entity settings: %+v", entitySettings)
	}
	if fieldSettings.ServiceName != "OrderService" || fieldSettings.EntityName != "PurchaseOrder" {
		t.Fatalf("unexpected field settings: %+v", fieldSettings)
	}
	if len(fieldSettings.Fields) != 2 || fieldSettings.Fields[0] != (application.FieldSetting{OriginalName: "Total", Name: "Amount", Type: "Money"}) || fieldSettings.Fields[1] != (application.FieldSetting{Name: "Currency", Type: "string"}) {
		t.Fatalf("unexpected field settings: %+v", fieldSettings)
	}
	if ui.editOpen {
		t.Fatalf("expected native entity edit form to close after save")
	}
}

func TestTViewEntityManagerDoesNotSaveFieldsInline(t *testing.T) {
	fieldCalls := 0
	entityCalls := 0
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{{Name: "OrderService", Entities: []application.EntitySummary{{Name: "Order", Fields: []application.FieldSummary{{Name: "Total", Type: "decimal"}}}}}}}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{}, nil
		},
		nil,
		nil,
		nil,
		func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
			entityCalls++
			return application.UpdateEntitySettingsResult{}, nil
		},
		func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			fieldCalls++
			return application.UpdateFieldSettingsResult{}, nil
		},
	)

	ui.saveEntitiesEdit(&tviewEntitiesEditState{serviceName: "OrderService", rows: []tviewManagerRow{{original: "Order", name: "PurchaseOrder"}}})

	if entityCalls != 1 || fieldCalls != 0 {
		t.Fatalf("expected entities save to be separate from fields save, entityCalls=%d fieldCalls=%d", entityCalls, fieldCalls)
	}
}

func TestTViewFieldsManagerSavesFieldPayloadSeparately(t *testing.T) {
	var updated application.FieldSettings
	ui := newTViewUI(
		application.GenerationPlan{},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{}, nil
		},
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			updated = settings
			return application.UpdateFieldSettingsResult{}, nil
		},
	)

	ui.saveFieldsEdit(&tviewFieldsEditState{serviceName: "OrderService", entityName: "Order", rows: []tviewManagerRow{
		{original: "Total", name: "Amount", typeName: "Money"},
		{name: "Currency", typeName: "string"},
	}})

	if updated.ServiceName != "OrderService" || updated.EntityName != "Order" {
		t.Fatalf("unexpected field context: %+v", updated)
	}
	if len(updated.Fields) != 2 || updated.Fields[0] != (application.FieldSetting{OriginalName: "Total", Name: "Amount", Type: "Money"}) || updated.Fields[1] != (application.FieldSetting{Name: "Currency", Type: "string"}) {
		t.Fatalf("unexpected field payload: %+v", updated.Fields)
	}
}

func TestTViewEntityEditBlocksGenerationWhenFieldSaveFailsAfterEntitySave(t *testing.T) {
	generated := 0
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{
			{Name: "OrderService", Entities: []application.EntitySummary{{Name: "Order", Fields: []application.FieldSummary{{Name: "Total", Type: "decimal"}}}}},
		}}},
		application.GenerateRequest{},
		nil,
		func(application.GenerateRequest) (application.GenerateResult, error) {
			generated++
			return application.GenerateResult{Plan: application.GenerationPlan{FileCount: 1}, OutputDir: "out"}, nil
		},
		nil,
		nil,
		func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
			return application.UpdateEntitySettingsResult{Plan: application.GenerationPlan{FileCount: 1}}, nil
		},
		func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			return application.UpdateFieldSettingsResult{}, errors.New("field save failed")
		},
	)
	ui.saveEntitiesEdit(&tviewEntitiesEditState{serviceName: "OrderService", rows: []tviewManagerRow{{original: "Order", name: "Order"}}})
	ui.saveFieldsEdit(&tviewFieldsEditState{serviceName: "OrderService", entityName: "Order", rows: []tviewManagerRow{{original: "Total", name: "Total", typeName: "decimal"}}})
	ui.generateFiles()

	if !ui.planStale {
		t.Fatalf("expected partial entity save to mark plan stale")
	}
	if generated != 0 {
		t.Fatalf("expected stale partial save to block generation, got %d", generated)
	}
	if !strings.Contains(ui.message, "blocked until the plan refreshes successfully") {
		t.Fatalf("expected stale generation block message, got %q", ui.message)
	}
}

func TestTViewValueObjectEditSavesSelectedServiceAndAddsNatively(t *testing.T) {
	var updated application.ValueObjectSettings
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{
			{Name: "ProductService", ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string"}}},
			{Name: "OrderService", ValueObjects: []application.ValueObjectSummary{{Name: "OrderNumber", Type: "string"}}},
		}}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{FileCount: 2}, nil
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			updated = settings
			return application.UpdateValueObjectSettingsResult{Plan: application.GenerationPlan{FileCount: 2}}, nil
		},
	)
	ui.saveValueObjectsEdit(&tviewValueObjectsEditState{serviceName: "OrderService", rows: []tviewManagerRow{
		{original: "OrderNumber", name: " PurchaseNumber ", typeName: " string "},
		{name: " Money ", typeName: " decimal "},
		{name: " Code ", typeName: " string "},
	}})

	if updated.ServiceName != "OrderService" {
		t.Fatalf("unexpected value object settings: %+v", updated)
	}
	if len(updated.ValueObjects) != 3 || updated.ValueObjects[0].OriginalName != "OrderNumber" || updated.ValueObjects[0].Name != "PurchaseNumber" || updated.ValueObjects[0].Type != "string" || updated.ValueObjects[1].Name != "Money" || updated.ValueObjects[1].Type != "decimal" || updated.ValueObjects[2].Name != "Code" || updated.ValueObjects[2].Type != "string" {
		t.Fatalf("unexpected value object settings: %+v", updated)
	}
	rules := updated.ValueObjects[2].Validations
	if rules.Required == nil || !*rules.Required || rules.MinLength == nil || *rules.MinLength != 1 || rules.MaxLength == nil || *rules.MaxLength != 100 || rules.ValidExample == nil || *rules.ValidExample != "Sample" {
		t.Fatalf("expected native new value object defaults, got %+v", rules)
	}
	if ui.editOpen {
		t.Fatalf("expected native value object edit form to close after save")
	}
}

func TestTViewValueObjectRulesSavePreservesRowsAndAppliesRules(t *testing.T) {
	var updated application.ValueObjectSettings
	ui := newTViewUI(
		application.GenerationPlan{},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{}, nil
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			updated = settings
			return application.UpdateValueObjectSettingsResult{}, nil
		},
	)
	form := tview.NewForm()
	form.AddCheckbox("Required", true, nil)
	form.AddInputField("Min length", "3", 32, nil, nil)
	form.AddInputField("Max length", "80", 32, nil, nil)
	form.AddInputField("Pattern", "^[A-Z]", 32, nil, nil)
	form.AddInputField("Valid example", "ABC", 32, nil, nil)
	form.AddInputField("Invalid example", "abc", 32, nil, nil)
	form.AddInputField("Minimum", "1", 32, nil, nil)
	form.AddInputField("Maximum", "99", 32, nil, nil)
	form.AddCheckbox("Not empty", true, nil)
	form.AddCheckbox("Not default", false, nil)

	ui.saveValueObjectRulesEdit(&tviewValueObjectRulesEditState{
		serviceName: "OrderService",
		valueObject: application.ValueObjectSummary{Name: "OrderNumber", Type: "string"},
		rows: []tviewManagerRow{
			{original: "OrderNumber", name: "OrderNumber", typeName: "string"},
			{original: "Money", name: "Money", typeName: "decimal", validations: application.ValidationRuleSettings{Minimum: stringPtr("0")}},
		},
	}, form)

	if updated.ServiceName != "OrderService" || len(updated.ValueObjects) != 2 {
		t.Fatalf("unexpected value object settings: %+v", updated)
	}
	rules := updated.ValueObjects[0].Validations
	if rules.Required == nil || !*rules.Required || rules.MinLength == nil || *rules.MinLength != 3 || rules.MaxLength == nil || *rules.MaxLength != 80 || rules.Pattern == nil || *rules.Pattern != "^[A-Z]" || rules.ValidExample == nil || *rules.ValidExample != "ABC" || rules.InvalidExample == nil || *rules.InvalidExample != "abc" {
		t.Fatalf("unexpected applied validation rules: %+v", rules)
	}
	if rules.Minimum != nil || rules.Maximum != nil || rules.NotEmpty != nil || rules.NotDefault != nil {
		t.Fatalf("expected non-applicable string rules to be omitted, got %+v", rules)
	}
	if updated.ValueObjects[1].Name != "Money" || updated.ValueObjects[1].Validations.Minimum == nil || *updated.ValueObjects[1].Validations.Minimum != "0" {
		t.Fatalf("expected other value objects to be preserved, got %+v", updated.ValueObjects[1])
	}
}

func TestTViewValueObjectRulesSaveOmitsNonApplicableFalseRulesThroughApplicationUpdate(t *testing.T) {
	var captured application.ValueObjectSettings
	var saved spec.Config
	cfg := spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation: spec.GenerationOptions{
			TargetFramework: spec.DefaultTargetFramework,
			SolutionFormat:  spec.DefaultSolutionFormat(spec.DefaultTargetFramework),
		},
		Solution: spec.Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []spec.Service{{
			Name:         "ProductService",
			ValueObjects: []spec.ValueObject{{Name: "ProductName", Type: "string"}},
			Entities:     []spec.Entity{{Name: "Product", Fields: []spec.Field{{Name: "Id", Type: "Guid"}}}},
		}},
	}
	service := application.NewService(application.Ports{
		ConfigLoader: tuiConfigLoaderFunc(func(string) (spec.Config, error) { return cfg, nil }),
		ConfigSaver: tuiConfigSaverFunc(func(_ string, cfg spec.Config) error {
			saved = cfg
			return nil
		}),
		Generator: tuiGeneratorFunc(func(spec.Config) ([]application.GeneratedFile, error) { return nil, nil }),
		OutputPlanner: tuiOutputPlannerFunc(func(string, []application.GeneratedFile, bool) (application.OutputPlan, error) {
			return application.OutputPlan{}, nil
		}),
	})
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil, nil, nil, nil, func(request application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		captured = settings
		return service.UpdateValueObjectSettings(request, settings)
	})
	form := tview.NewForm()
	form.AddCheckbox("Required", false, nil)
	form.AddInputField("Min length", "3", 32, nil, nil)
	form.AddInputField("Max length", "80", 32, nil, nil)
	form.AddInputField("Pattern", "", 32, nil, nil)
	form.AddInputField("Valid example", "Sample", 32, nil, nil)
	form.AddInputField("Invalid example", "", 32, nil, nil)
	form.AddInputField("Minimum", "1", 32, nil, nil)
	form.AddInputField("Maximum", "99", 32, nil, nil)
	form.AddCheckbox("Not empty", false, nil)
	form.AddCheckbox("Not default", false, nil)

	ui.saveValueObjectRulesEdit(&tviewValueObjectRulesEditState{
		serviceName: "ProductService",
		valueObject: application.ValueObjectSummary{Name: "ProductName", Type: "string"},
		rows:        []tviewManagerRow{{original: "ProductName", name: "ProductName", typeName: "string"}},
	}, form)

	if ui.err != nil {
		t.Fatalf("expected application update to accept string rules, got %v", ui.err)
	}
	rules := captured.ValueObjects[0].Validations
	if rules.MinLength == nil || *rules.MinLength != 3 || rules.MaxLength == nil || *rules.MaxLength != 80 || rules.ValidExample == nil || *rules.ValidExample != "Sample" {
		t.Fatalf("unexpected captured string rules: %+v", rules)
	}
	if rules.Required != nil || rules.Minimum != nil || rules.Maximum != nil || rules.NotEmpty != nil || rules.NotDefault != nil {
		t.Fatalf("expected non-applicable false rules to be omitted from captured settings, got %+v", rules)
	}
	savedRules := saved.Services[0].ValueObjects[0].Validations
	if savedRules.MinLength == nil || *savedRules.MinLength != 3 || savedRules.MaxLength == nil || *savedRules.MaxLength != 80 || savedRules.ValidExample == nil || *savedRules.ValidExample != "Sample" {
		t.Fatalf("unexpected saved string rules: %+v", savedRules)
	}
	if savedRules.Required != nil || savedRules.Minimum != nil || savedRules.Maximum != nil || savedRules.NotEmpty != nil || savedRules.NotDefault != nil {
		t.Fatalf("expected non-applicable false rules to be omitted from saved config, got %+v", savedRules)
	}
}

func TestTViewValueObjectRulesSavePreservesNewValueObjectWithUpdatedRules(t *testing.T) {
	var updated application.ValueObjectSettings
	ui := newTViewUI(
		application.GenerationPlan{},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			return application.GenerationPlan{}, nil
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			updated = settings
			return application.UpdateValueObjectSettingsResult{}, nil
		},
	)
	form := tview.NewForm()
	form.AddCheckbox("Required", true, nil)
	form.AddInputField("Min length", "2", 32, nil, nil)
	form.AddInputField("Max length", "", 32, nil, nil)
	form.AddInputField("Pattern", "", 32, nil, nil)
	form.AddInputField("Valid example", "OK", 32, nil, nil)
	form.AddInputField("Invalid example", "", 32, nil, nil)
	form.AddInputField("Minimum", "", 32, nil, nil)
	form.AddInputField("Maximum", "", 32, nil, nil)
	form.AddCheckbox("Not empty", true, nil)
	form.AddCheckbox("Not default", false, nil)

	ui.saveValueObjectRulesEdit(&tviewValueObjectRulesEditState{
		serviceName: "OrderService",
		valueObject: application.ValueObjectSummary{Name: "NewValueObject2", Type: "Guid"},
		rows: []tviewManagerRow{
			{original: "OrderNumber", name: "OrderNumber", typeName: "string"},
			{name: "NewValueObject2", typeName: "Guid"},
		},
	}, form)

	if updated.ServiceName != "OrderService" || len(updated.ValueObjects) != 2 {
		t.Fatalf("unexpected value object payload: %+v", updated)
	}
	if updated.ValueObjects[1].OriginalName != "" || updated.ValueObjects[1].Name != "NewValueObject2" || updated.ValueObjects[1].Type != "Guid" {
		t.Fatalf("expected newly added value object to be preserved completely, got %+v", updated.ValueObjects[1])
	}
	rules := updated.ValueObjects[1].Validations
	if rules.NotEmpty == nil || !*rules.NotEmpty {
		t.Fatalf("expected rules to be applied to new value object, got %+v", rules)
	}
	if rules.Required != nil || rules.MinLength != nil || rules.ValidExample != nil || rules.NotDefault != nil {
		t.Fatalf("expected non-applicable Guid rules to be omitted, got %+v", rules)
	}
}

func TestTViewValueObjectRulesModalLetsFormHandleTab(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil, nil, nil, nil, nil)

	ui.showValueObjectRulesManager(&tviewValueObjectRulesEditState{
		serviceName: "ProductService",
		valueObject: application.ValueObjectSummary{Name: "ProductName", Type: "string"},
		validations: application.ValidationRuleSettings{Required: boolPtr(true)},
		rows:        []tviewManagerRow{{original: "ProductName", name: "ProductName", typeName: "string"}},
	})

	if !ui.editOpen {
		t.Fatalf("expected rules modal to open")
	}
	if _, ok := ui.app.GetFocus().(*tview.Checkbox); !ok {
		t.Fatalf("expected rules modal focus to start on the first form control, got %T", ui.app.GetFocus())
	}
	if len(ui.modalFocus) != 0 {
		t.Fatalf("expected rules modal not to register modal focus cycling, got %d entries", len(ui.modalFocus))
	}
	tab := tcell.NewEventKey(tcell.KeyTAB, 0, tcell.ModNone)
	if got := ui.handleKey(tab); got != tab {
		t.Fatalf("expected tab to reach the rules form, got %v", got)
	}
	sendKeyToTViewFocus(ui, tab)
	if _, ok := ui.app.GetFocus().(*tview.InputField); !ok {
		t.Fatalf("expected tab to move focus through rules form controls, got %T", ui.app.GetFocus())
	}
	backtab := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModShift)
	if got := ui.handleKey(backtab); got != backtab {
		t.Fatalf("expected shift+tab to reach the rules form, got %v", got)
	}
}

func TestTViewEditKeyOpensNativeNestedForms(t *testing.T) {
	var entitySettings application.EntitySettings
	var valueObjectSettings application.ValueObjectSettings
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{{Name: "ProductService", Entities: []application.EntitySummary{{Name: "Product"}}, ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string"}}}}}},
		application.GenerateRequest{},
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
			entitySettings = settings
			return application.UpdateEntitySettingsResult{}, nil
		},
		nil,
		func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			valueObjectSettings = settings
			return application.UpdateValueObjectSettingsResult{}, nil
		},
	)

	ui.open(tviewScreenEntities)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))
	if !ui.editOpen {
		t.Fatalf("expected entities edit to stay in tview")
	}
	if !ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected entities edit to open as a modal page")
	}
	manager, ok := ui.app.GetFocus().(*tview.Table)
	if !ok || manager.GetCell(0, 0).Text != "Name" {
		t.Fatalf("expected entities edit focus to start on manager table, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if manager.GetRowCount() != 3 {
		t.Fatalf("expected add key to append an entity row, got %d rows", manager.GetRowCount())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone))
	if len(entitySettings.Entities) != 2 || entitySettings.Entities[1].Name != "NewEntity2" {
		t.Fatalf("expected entity save to include added row, got %+v", entitySettings)
	}
	if !ui.editOpen {
		t.Fatalf("expected entities modal to stay open after ctrl+s save")
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))

	ui.open(tviewScreenValueObjects)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))
	if !ui.editOpen {
		t.Fatalf("expected value objects edit to stay in tview")
	}
	if !ui.root.HasPage(tviewEditModalPage) {
		t.Fatalf("expected value objects edit to open as a modal page")
	}
	manager, ok = ui.app.GetFocus().(*tview.Table)
	if !ok || manager.GetCell(0, 0).Text != "Name" {
		t.Fatalf("expected value objects edit focus to start on manager table, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if manager.GetRowCount() != 3 {
		t.Fatalf("expected add key to append a value-object row, got %d rows", manager.GetRowCount())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone))
	if len(valueObjectSettings.ValueObjects) != 2 || valueObjectSettings.ValueObjects[1].Name != "NewValueObject2" {
		t.Fatalf("expected value-object save to include added row, got %+v", valueObjectSettings)
	}
	if !ui.editOpen {
		t.Fatalf("expected value objects modal to stay open after ctrl+s save")
	}
}

func TestTViewEntityServicePickerSelectsAndCancelsWithoutFocusTrap(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{
		{Name: "ProductService", Entities: []application.EntitySummary{{Name: "Product"}}},
		{Name: "OrderService", Entities: []application.EntitySummary{{Name: "Order"}}},
	}}}, application.GenerateRequest{}, nil, nil)

	ui.openEntitiesEditForService("ProductService")
	manager, ok := ui.app.GetFocus().(*tview.Table)
	if !ok {
		t.Fatalf("expected entities manager focus, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if _, ok := ui.app.GetFocus().(*tview.List); !ok {
		t.Fatalf("expected service picker list focus, got %T", ui.app.GetFocus())
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if ui.app.GetFocus() != manager || ui.root.HasPage(tviewServicePickerPage) {
		t.Fatalf("expected service picker escape to restore manager focus without closing edit")
	}
	if !ui.editOpen {
		t.Fatalf("expected canceling picker to keep entity manager open")
	}

	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	list, ok := ui.app.GetFocus().(*tview.List)
	if !ok {
		t.Fatalf("expected service picker list focus, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone))
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ui.root.HasPage(tviewServicePickerPage) || ui.app.GetFocus() == list {
		t.Fatalf("expected selecting a service to close picker and leave list focus")
	}
	manager, ok = ui.app.GetFocus().(*tview.Table)
	if !ok || manager.GetCell(1, 0).Text != "Order" {
		t.Fatalf("expected service selection to open OrderService entities, got focus=%T first row=%q", ui.app.GetFocus(), manager.GetCell(1, 0).Text)
	}
}

func TestTViewValueObjectServicePickerSelectsAndCancelsWithoutFocusTrap(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{
		{Name: "ProductService", ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string"}}},
		{Name: "OrderService", ValueObjects: []application.ValueObjectSummary{{Name: "OrderNumber", Type: "string"}}},
	}}}, application.GenerateRequest{}, nil, nil)

	ui.openValueObjectsEditForService("ProductService")
	manager, ok := ui.app.GetFocus().(*tview.Table)
	if !ok {
		t.Fatalf("expected value objects manager focus, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if _, ok := ui.app.GetFocus().(*tview.List); !ok {
		t.Fatalf("expected service picker list focus, got %T", ui.app.GetFocus())
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if ui.app.GetFocus() != manager || ui.root.HasPage(tviewServicePickerPage) {
		t.Fatalf("expected service picker escape to restore manager focus without closing edit")
	}

	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	manager, ok = ui.app.GetFocus().(*tview.Table)
	if !ok || manager.GetCell(1, 0).Text != "OrderNumber" {
		t.Fatalf("expected service selection to open OrderService value objects, got focus=%T first row=%q", ui.app.GetFocus(), manager.GetCell(1, 0).Text)
	}
}

func TestTViewFieldsManagerStartsOnOperableTableAndSavesAddedField(t *testing.T) {
	var fieldSettings application.FieldSettings
	ui := newTViewUI(
		application.GenerationPlan{},
		application.GenerateRequest{},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			fieldSettings = settings
			return application.UpdateFieldSettingsResult{}, nil
		},
	)

	ui.showFieldsManager(&tviewFieldsEditState{serviceName: "ProductService", entityName: "Product", rows: []tviewManagerRow{{original: "Name", name: "Name", typeName: "string"}}})

	manager, ok := ui.app.GetFocus().(*tview.Table)
	if !ok {
		t.Fatalf("expected fields modal to focus an operable table, got %T", ui.app.GetFocus())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if manager.GetRowCount() != 3 {
		t.Fatalf("expected add key to append a field row, got %d rows", manager.GetRowCount())
	}
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone))

	if fieldSettings.ServiceName != "ProductService" || fieldSettings.EntityName != "Product" {
		t.Fatalf("unexpected field context: %+v", fieldSettings)
	}
	if len(fieldSettings.Fields) != 2 || fieldSettings.Fields[1].Name != "NewField2" || fieldSettings.Fields[1].Type != "string" {
		t.Fatalf("expected saved fields to include added row, got %+v", fieldSettings.Fields)
	}
}

func TestTViewGenerateBlocksAfterSaveRefreshFailureUntilSuccessfulRefresh(t *testing.T) {
	refreshes := 0
	generated := make(chan struct{}, 1)
	completed := make(chan struct{}, 1)
	ui := newTViewUI(
		application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce"}},
		application.GenerateRequest{},
		func(application.GenerateRequest) (application.GenerationPlan, error) {
			refreshes++
			if refreshes == 1 {
				return application.GenerationPlan{}, errors.New("plan failed")
			}
			return application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce"}, FileCount: 3}, nil
		},
		func(application.GenerateRequest) (application.GenerateResult, error) {
			generated <- struct{}{}
			return application.GenerateResult{Plan: application.GenerationPlan{FileCount: 3}, OutputDir: "out"}, nil
		},
		func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
			return application.UpdateSolutionSettingsResult{Plan: application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce"}}}, nil
		},
	)
	ui.queueUpdateDraw = func(fn func()) {
		fn()
		completed <- struct{}{}
	}

	ui.open(tviewScreenProject)
	ui.handleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))
	sendKeyToTViewFocus(ui, tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone))
	ui.generateFiles()

	if len(generated) != 0 {
		t.Fatalf("expected generation to be blocked while plan is stale")
	}
	if !ui.planStale || !strings.Contains(ui.message, "blocked until the plan refreshes successfully") {
		t.Fatalf("expected stale plan block message, stale=%t message=%q", ui.planStale, ui.message)
	}

	ui.refreshPlan()
	ui.generateFiles()

	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("expected generation after successful refresh")
	}
	if len(generated) != 1 {
		t.Fatalf("expected generation after successful refresh, got %d", len(generated))
	}
}

func TestTViewTabCyclesVisiblePanels(t *testing.T) {
	ui := newTViewUI(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil)

	if ui.focus != tviewFocusSidebar || ui.app.GetFocus() != ui.sidebar {
		t.Fatalf("expected initial sidebar focus")
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyTAB, 0, tcell.ModNone))
	if ui.focus != tviewFocusDetails || ui.app.GetFocus() != ui.detail {
		t.Fatalf("expected details focus after tab")
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyTAB, 0, tcell.ModNone))
	if ui.focus != tviewFocusFiles || ui.app.GetFocus() != ui.files {
		t.Fatalf("expected files focus after second tab")
	}
	ui.handleKey(tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModShift))
	if ui.focus != tviewFocusDetails || ui.app.GetFocus() != ui.detail {
		t.Fatalf("expected details focus after shift tab")
	}
}

func TestNewLegacyModelPreservesEditableCallbacks(t *testing.T) {
	updateServices := func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
		return application.UpdateServiceSettingsResult{}, nil
	}
	updateEntities := func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error) {
		return application.UpdateEntitySettingsResult{}, nil
	}
	updateFields := func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
		return application.UpdateFieldSettingsResult{}, nil
	}
	updateValueObjects := func(application.GenerateRequest, application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		return application.UpdateValueObjectSettingsResult{}, nil
	}
	updateRelationships := func(application.GenerateRequest, application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error) {
		return application.UpdateRelationshipSettingsResult{}, nil
	}

	model := newLegacyModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil, updateServices, updateEntities, updateFields, updateValueObjects, updateRelationships, []string{"net10.0"})

	if model.updateServices == nil || model.updateEntities == nil || model.updateFields == nil || model.updateValueObjects == nil || model.updateRelationships == nil {
		t.Fatalf("expected legacy workspace callbacks to be preserved")
	}
	if got := model.targetFrameworkSuggestions; len(got) != 1 || got[0] != "net10.0" {
		t.Fatalf("expected target framework suggestions to be preserved, got %v", got)
	}
}

func TestLayoutModeForWidthUsesResponsiveBreakpoints(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  layoutMode
	}{
		{name: "narrow below medium", width: 75, want: layoutNarrow},
		{name: "medium lower bound", width: 76, want: layoutMedium},
		{name: "medium upper bound", width: 99, want: layoutMedium},
		{name: "wide lower bound", width: 100, want: layoutWide},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := layoutModeForWidth(tt.width); got != tt.want {
				t.Fatalf("layoutModeForWidth(%d) = %v, want %v", tt.width, got, tt.want)
			}
		})
	}
}

func TestClampIntKeepsValuesWithinBounds(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{name: "below lower bound", value: -1, want: 0},
		{name: "inside bounds", value: 2, want: 2},
		{name: "above upper bound", value: 9, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampInt(tt.value, 0, 4); got != tt.want {
				t.Fatalf("clampInt(%d, 0, 4) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestModelUpdateSelectsAndOpensWorkspaceRoutes(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.selectedScreen != screenProject || model.activeScreen() != screenOverview {
		t.Fatalf("expected Project to be selected without opening it, got screen=%v selected=%v cmd=%v", model.activeScreen(), model.selectedScreen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.screen != screenProject || model.currentStep != stepProject {
		t.Fatalf("expected enter to open Project, got screen=%v step=%v cmd=%v", model.screen, model.currentStep, cmd)
	}
	assertContains(t, model.View(), "Solution")
	assertNotContains(t, model.View(), "Wizard")
}

func TestModelUpdateRouteSelectorOpensMovesConfirmsAndCancels(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.openScreen(screenServices)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = updated.(Model)
	if cmd != nil || !model.routeSelectorOpen || model.routeSelectorScreen != screenServices || model.activeScreen() != screenServices {
		t.Fatalf("expected ctrl+p to open selector on active Services route, open=%v selector=%v active=%v cmd=%v", model.routeSelectorOpen, model.routeSelectorScreen, model.activeScreen(), cmd)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorScreen != screenEntities || model.activeScreen() != screenServices {
		t.Fatalf("expected selector down to highlight Entities without opening it, selector=%v active=%v cmd=%v", model.routeSelectorScreen, model.activeScreen(), cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorScreen != screenValueObjects || model.activeScreen() != screenServices {
		t.Fatalf("expected selector j to highlight Value Objects without opening it, selector=%v active=%v cmd=%v", model.routeSelectorScreen, model.activeScreen(), cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorScreen != screenEntities || model.activeScreen() != screenServices {
		t.Fatalf("expected selector up to highlight Entities without opening it, selector=%v active=%v cmd=%v", model.routeSelectorScreen, model.activeScreen(), cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorScreen != screenServices || model.activeScreen() != screenServices {
		t.Fatalf("expected selector k to highlight Services, selector=%v active=%v cmd=%v", model.routeSelectorScreen, model.activeScreen(), cmd)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorOpen || model.screen != screenEntities || model.selectedScreen != screenEntities || model.currentStep != stepEntities {
		t.Fatalf("expected enter to confirm selector route, open=%v screen=%v selected=%v step=%v cmd=%v", model.routeSelectorOpen, model.screen, model.selectedScreen, model.currentStep, cmd)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = updated.(Model)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.routeSelectorOpen || model.activeScreen() != screenEntities || model.screen != screenEntities {
		t.Fatalf("expected esc to cancel selector and preserve active route, open=%v active=%v screen=%v cmd=%v", model.routeSelectorOpen, model.activeScreen(), model.screen, cmd)
	}
}

func TestModelUpdateRouteSelectorIsBlockedDuringSafetyCriticalStates(t *testing.T) {
	tests := []struct {
		name       string
		status     modelStatus
		errContext string
	}{
		{name: "refreshing", status: statusRefreshing},
		{name: "generating", status: statusGenerating},
		{name: "saving", status: statusSaving},
		{name: "post-save refresh lock", status: statusFailed, errContext: "Refresh after save"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
			model.openScreen(screenServices)
			model.status = tt.status
			model.errContext = tt.errContext
			selectorBefore := model.routeSelectorScreen

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
			model = updated.(Model)
			if cmd != nil || model.routeSelectorOpen || model.routeSelectorScreen != selectorBefore || model.screen != screenServices {
				t.Fatalf("expected selector open to be blocked, open=%v selector=%v screen=%v cmd=%v", model.routeSelectorOpen, model.routeSelectorScreen, model.screen, cmd)
			}
		})
	}
}

func TestModelUpdateNestedEditorShortcutsAreBlockedDuringBusyStates(t *testing.T) {
	tests := []struct {
		name   string
		status modelStatus
	}{
		{name: "refreshing", status: statusRefreshing},
		{name: "generating", status: statusGenerating},
		{name: "saving", status: statusSaving},
	}

	for _, tt := range tests {
		t.Run("fields shortcut "+tt.name, func(t *testing.T) {
			model := modelOnStep(plannedFilesPlan(1), stepEntities)
			model.status = tt.status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
			model = updated.(Model)

			if cmd != nil || model.edit.mode == editModeFields || model.status != tt.status {
				t.Fatalf("expected busy state to block fields shortcut, status=%v mode=%v cmd=%v", model.status, model.edit.mode, cmd)
			}
		})

		t.Run("rules shortcut "+tt.name, func(t *testing.T) {
			model := modelOnStep(plannedFilesPlan(1), stepValueObjects)
			model.status = tt.status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
			model = updated.(Model)

			if cmd != nil || model.valueObjectsEdit.rulesOpen || model.status != tt.status {
				t.Fatalf("expected busy state to block rules shortcut, status=%v rulesOpen=%v cmd=%v", model.status, model.valueObjectsEdit.rulesOpen, cmd)
			}
		})
	}
}

func TestModelUpdateSwitchesServicesResourceContextsAndSelections(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{
		{Name: "ProductService", EntityNames: []string{"Product"}, ValueObjectNames: []string{"ProductName"}},
		{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}, ValueObjectNames: []string{"OrderNumber"}},
	}}
	model := modelOnStep(plan, stepServices)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.selectedService != 1 {
		t.Fatalf("expected second service selected, got selected=%d cmd=%v", model.selectedService, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if cmd != nil || model.serviceContext != serviceResourceEntities {
		t.Fatalf("expected Entities context, got context=%v cmd=%v", model.serviceContext, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil || model.selectedEntity != 1 {
		t.Fatalf("expected second entity selected, got selected=%d cmd=%v", model.selectedEntity, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(Model)
	if cmd != nil || model.serviceContext != serviceResourceValueObjects {
		t.Fatalf("expected Value Objects context, got context=%v cmd=%v", model.serviceContext, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	if cmd != nil || model.serviceContext != serviceResourceEntities {
		t.Fatalf("expected Entities context after left, got context=%v cmd=%v", model.serviceContext, cmd)
	}
}

func TestModelUpdateEntersNestedServicesEditorsAndBacksToWorkspace(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{{Name: "OrderService", EntityNames: []string{"Order", "OrderLine"}, Entities: []application.EntitySummary{{Name: "Order"}, {Name: "OrderLine"}}}}}
	model := modelOnStep(plan, stepServices)
	model.serviceContext = serviceResourceEntities
	model.selectedEntity = 1

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeEntities || model.entitiesEdit.selected != 1 {
		t.Fatalf("expected selected entity editor, got %#v cmd=%v", model, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	if cmd != nil || model.edit.mode != editModeFields {
		t.Fatalf("expected fields editor, got mode=%v cmd=%v", model.edit.mode, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.activeScreen() != screenEntities || model.serviceContext != serviceResourceEntities {
		t.Fatalf("expected fields editor to back to Entities workspace, got status=%v screen=%v context=%v cmd=%v", model.status, model.activeScreen(), model.serviceContext, cmd)
	}
}

func TestModelUpdateNumericShortcutsOpenCompatibilityRoutes(t *testing.T) {
	for key, want := range map[rune]workspaceScreen{'1': screenOverview, '2': screenProject, '3': screenServices, '4': screenEntities, '5': screenValueObjects, '6': screenPreview, '7': screenGenerate, '8': screenResult} {
		t.Run(fmt.Sprintf("route %c", key), func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
			model = updated.(Model)
			if cmd != nil || model.screen != want || model.selectedScreen != want {
				t.Fatalf("expected key %c to open %v, got screen=%v selected=%v cmd=%v", key, want, model.screen, model.selectedScreen, cmd)
			}
		})
	}
}

func TestModelUpdateProjectEditorUsesDedicatedRouteAndBackNavigation(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(Model)
	if cmd != nil || model.screen != screenProject {
		t.Fatalf("expected Project route, got screen=%v cmd=%v", model.screen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeProject {
		t.Fatalf("expected project editor, got status=%v mode=%v cmd=%v", model.status, model.edit.mode, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.screen != screenProject {
		t.Fatalf("expected esc to leave editor on Project, got status=%v screen=%v cmd=%v", model.status, model.screen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenOverview {
		t.Fatalf("expected second esc to return Overview, got screen=%v cmd=%v", model.screen, cmd)
	}
}

func TestModelUpdateHelpOverlayAndBusyLocks(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = updated.(Model)
	if cmd != nil || !model.helpOpen || !strings.Contains(stripANSI(model.View()), "Keys") {
		t.Fatalf("expected help overlay, got open=%v cmd=%v view=%q", model.helpOpen, cmd, model.View())
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.helpOpen {
		t.Fatalf("expected esc to close help, got open=%v cmd=%v", model.helpOpen, cmd)
	}

	model.status = statusRefreshing
	model.selectedScreen = screenProject
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	model = updated.(Model)
	if cmd != nil || model.screen != screenOverview || model.selectedScreen != screenProject {
		t.Fatalf("expected busy route lock, got screen=%v selected=%v cmd=%v", model.screen, model.selectedScreen, cmd)
	}

	model.status = statusFailed
	model.errContext = "Refresh after save"
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if cmd != nil || model.status != statusFailed {
		t.Fatalf("expected stale-plan generation lock, got status=%v cmd=%v", model.status, cmd)
	}
}

func TestModelViewUsesResponsiveWorkspaceShell(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		want     string
		unwanted string
	}{
		{name: "wide rail", width: 120, want: "enter open | ctrl+p routes", unwanted: "Navigation ["},
		{name: "medium top navigation", width: 90, want: "Routes > 1 Overview", unwanted: "Enter opens selected route"},
		{name: "narrow focused content", width: 60, want: "Overview", unwanted: "Enter opens selected route"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/out"}, nil, nil, nil)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: tt.width, Height: 24})
			model = updated.(Model)
			if cmd != nil || model.layout != layoutModeForWidth(tt.width) {
				t.Fatalf("expected layout %v, got %v cmd=%v", layoutModeForWidth(tt.width), model.layout, cmd)
			}
			view := stripANSI(model.View())
			assertContains(t, view, tt.want)
			assertNotContains(t, view, tt.unwanted)
			assertNotContains(t, view, "Wizard")
		})
	}
}

func TestModelViewRouteSelectorShowsModalDiscoveryAndSafetyContext(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.ForceRequired = true
	plan.Readiness.OutputForceRequired = true
	model := workspaceModel(plan, application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	model.openScreen(screenPreview)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = updated.(Model)
	if cmd != nil {
		t.Fatalf("expected selector open without command, got %v", cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil {
		t.Fatalf("expected selector movement without command, got %v", cmd)
	}

	view := stripANSI(model.View())
	assertContains(t, view, "Routes")
	assertContains(t, view, "active Preview  target Generate")
	assertContains(t, view, "1 Overview")
	assertContains(t, view, "8 Result")
	assertContains(t, view, "enter switch | esc cancel | up/down move")
	assertContains(t, view, "Safety: force required before writing to /tmp/generated")
}

func TestTUIViewHidesAdvancedWorkspaceWords(t *testing.T) {
	plan := plannedFilesPlan(3)
	plan.OutputDir = "/tmp/generated"
	tests := []struct {
		name    string
		prepare func(*Model)
	}{
		{name: "menu", prepare: func(model *Model) { model.enterWizardMenu() }},
		{name: "services", prepare: func(model *Model) { model.enterWizardServices() }},
		{name: "value objects", prepare: func(model *Model) { model.enterWizardValueObjects() }},
		{name: "result", prepare: func(model *Model) {
			model.mode = modeWizard
			model.wizardScreen = wizardResult
			model.status = statusGenerated
			model.result = application.GenerateResult{OutputDir: plan.OutputDir, Plan: plan}
		}},
		{name: "route shell", prepare: func(model *Model) { model.openScreen(screenPreview) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(plan, application.GenerateRequest{}, nil, nil, nil)
			tt.prepare(&model)
			view := strings.ToLower(stripANSI(model.View()))
			for _, forbidden := range []string{"advanced", "workspace", "guided setup"} {
				if strings.Contains(view, forbidden) {
					t.Fatalf("expected view not to contain %q, got %q", forbidden, view)
				}
			}
		})
	}
}

func TestWorkspaceViewClampsHeightAcrossRoutes(t *testing.T) {
	plan := plannedFilesPlan(40)
	for index := range plan.Files {
		plan.Files[index].Path = fmt.Sprintf("src/VeryLongGeneratedPathSegment%02d/AnotherLongSegment/Feature/ControllerWithVeryLongName%02d.cs", index, index)
	}
	model := workspaceModel(plan, application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 96, Height: 24})
	if cmd != nil {
		t.Fatal("expected no command from window size")
	}
	model = updated.(Model)

	for _, screen := range []workspaceScreen{screenOverview, screenProject, screenServices, screenEntities, screenValueObjects, screenPreview, screenGenerate, screenResult} {
		t.Run(screen.label(), func(t *testing.T) {
			model.openScreen(screen)
			view := stripANSI(model.View())
			if rows := renderedTestLineCount(view); rows > model.windowHeight {
				t.Fatalf("expected %s view to stay within %d rows, got %d rows in %q", screen.label(), model.windowHeight, rows, view)
			}
		})
	}
}

func TestPreviewClipsLongGeneratedPaths(t *testing.T) {
	longPath := strings.Repeat("very-long-segment/", 20) + "GeneratedController.cs"
	plan := plannedFilesPlan(2)
	plan.Files[0].Path = longPath
	plan.Files[1].Path = longPath + ".backup"
	model := workspaceModel(plan, application.GenerateRequest{OutputDir: "/tmp/generated"}, nil, nil, nil)
	model.openScreen(screenPreview)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	model = updated.(Model)

	view := stripANSI(model.View())
	if strings.Count(view, "very-long-segment") > 12 {
		t.Fatalf("expected long generated paths to be clipped, got view %q", view)
	}
	assertContains(t, view, "...")
}

func TestRouteSelectorRendersAsModal(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	model.openScreen(screenServices)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected selector open without command")
	}

	view := stripANSI(model.View())
	assertContains(t, view, "+")
	assertContains(t, view, "Routes")
	assertContains(t, view, "active Services  target Services")
	assertContains(t, view, "enter switch | esc cancel | up/down move")
	for lineIndex, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "active Services") && len([]rune(strings.TrimSpace(line))) >= 90 {
			t.Fatalf("expected bounded modal line, got line %d with %d modal columns: %q", lineIndex+1, len([]rune(strings.TrimSpace(line))), line)
		}
	}
}

func TestModelViewDistinguishesActiveSelectedAndPaneRegions(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "config.json"}, nil, nil, nil)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	model = updated.(Model)
	if cmd != nil {
		t.Fatalf("expected resize without command, got %v", cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil {
		t.Fatalf("expected route selection without command, got %v", cmd)
	}

	view := stripANSI(model.View())
	assertContains(t, view, "Route Overview/Project")
	assertContains(t, view, "Routes")
	assertNotContains(t, view, "Main detail")
	assertNotContains(t, view, "Command / status")
	assertContains(t, view, "ctrl+p routes")
	assertContains(t, view, "? help")
}

func TestModelViewResponsivePaneContextStaysReadable(t *testing.T) {
	for _, width := range []int{90, 48} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "config.json"}, nil, nil, nil)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width})
			model = updated.(Model)
			if cmd != nil {
				t.Fatalf("expected resize without command, got %v", cmd)
			}
			view := stripANSI(model.View())
			assertContains(t, view, "Routes")
			assertContains(t, view, "route Overview/Overview")
			assertNotContains(t, view, "Active Overview")
			assertNotContains(t, view, "Command / status")
			assertContains(t, view, "ctrl+p routes")
			assertContains(t, view, "? help")
		})
	}
}

func TestModelViewSelectorKeepsCriticalStatusRecoverable(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*Model)
		wantStatus string
	}{
		{name: "stale plan", configure: func(m *Model) { m.status = statusFailed; m.errContext = "Refresh after save" }, wantStatus: "Safety: stale plan locked; press r to retry refresh"},
		{name: "force required", configure: func(m *Model) { m.plan.ForceRequired = true; m.request.OutputDir = "/tmp/generated" }, wantStatus: "Safety: force required before writing to /tmp/generated"},
		{name: "overwrite output", configure: func(m *Model) {
			m.plan.OutputAction = "replace"
			m.plan.DeletedFiles = []string{"old.cs"}
			m.request.OutputDir = "/tmp/generated"
		}, wantStatus: "Safety: replace output may delete 1 file(s) in /tmp/generated"},
		{name: "generation busy", configure: func(m *Model) { m.status = statusGenerating }, wantStatus: "Safety: GENERATING in progress; route selector is locked"},
		{name: "callback failure", configure: func(m *Model) { m.status = statusFailed; m.errContext = "Generation" }, wantStatus: "Safety: Generation status is visible; route switching does not run callbacks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
			model.openScreen(screenGenerate)
			tt.configure(&model)
			if !model.busy() && !model.postSaveRefreshFailed() {
				updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
				model = updated.(Model)
				if cmd != nil {
					t.Fatalf("expected selector open without command, got %v", cmd)
				}
			}
			view := stripANSI(model.View())
			assertContains(t, view, tt.wantStatus)
		})
	}
}

func TestModelViewServicesWorkspaceUsesResponsiveResourceLayout(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{{
		Name:                  "ProductService",
		EntityNames:           []string{"Product"},
		ValueObjectNames:      []string{"ProductName"},
		ValueObjectReferences: []application.ValueObjectReferenceSummary{{ValueObjectName: "ProductName", EntityName: "Product", FieldName: "Name"}},
		Entities:              []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}},
	}}}

	for _, width := range []int{120, 90, 60} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			model := modelOnStep(plan, stepServices)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width})
			model = updated.(Model)
			if cmd != nil {
				t.Fatal("expected no command from window size")
			}
			view := stripANSI(model.View())
			assertContains(t, view, "Services")
			assertContains(t, view, "Selected service: ProductService")
			assertContains(t, view, "Context: [Services]  Entities  Value Objects")
			assertContains(t, view, "Entities: 1")
			assertContains(t, view, "Fields: 2")
			assertContains(t, view, "Value objects: 1")
			assertContains(t, view, "References: 1")
			assertContains(t, view, "ProductName <- Product.Name")
			assertNotContains(t, view, "Editing entities")
			assertNotContains(t, view, "Editing fields")
			assertNotContains(t, view, "Editing value objects")
		})
	}
}

func TestModelViewDedicatedResourceRoutesUseResponsiveListDetailLayout(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{{
		Name:                  "CatalogService",
		Entities:              []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "ProductName"}}}},
		ValueObjects:          []application.ValueObjectSummary{{Name: "ProductName", Type: "string", RulesLabel: "required, min=3"}},
		ValueObjectReferences: []application.ValueObjectReferenceSummary{{ValueObjectName: "ProductName", EntityName: "Product", FieldName: "Name"}},
	}}}

	for _, width := range []int{120, 90, 60} {
		t.Run(fmt.Sprintf("entities width %d", width), func(t *testing.T) {
			model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
			model.openScreen(screenEntities)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width})
			model = updated.(Model)
			if cmd != nil {
				t.Fatal("expected no command from window size")
			}
			view := stripANSI(model.View())
			assertContains(t, view, "Services > Entities")
			assertContains(t, view, "Entity list")
			assertContains(t, view, "Entity detail")
			assertContains(t, view, "Field count: 2")
			assertContains(t, view, "Name: ProductName")
			assertContains(t, view, "Referenced value objects")
			assertContains(t, view, "Enter/e edit entity | f edit fields")
		})
		t.Run(fmt.Sprintf("value objects width %d", width), func(t *testing.T) {
			model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
			model.openScreen(screenValueObjects)
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width})
			model = updated.(Model)
			if cmd != nil {
				t.Fatal("expected no command from window size")
			}
			view := stripANSI(model.View())
			assertContains(t, view, "Services > Value Objects")
			assertContains(t, view, "Value object list")
			assertContains(t, view, "Value object detail")
			assertContains(t, view, "Type: string")
			assertContains(t, view, "Validation rules: required, min=3")
			assertContains(t, view, "Referencing fields")
			assertContains(t, view, "Product.Name")
			assertContains(t, view, "Enter/e edit value object | o edit rules")
		})
	}
}

func TestModelUpdatePromotesServicesContextsAndNestedEditorsToDedicatedRoutes(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{{
		Name:         "CatalogService",
		Entities:     []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}},
		ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string"}},
	}}}
	model := modelOnStep(plan, stepServices)
	model.serviceContext = serviceResourceEntities

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.screen != screenEntities || model.status != statusEditing {
		t.Fatalf("expected Services Entities context to open dedicated entity route, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	if cmd != nil || model.screen != screenEntities || model.edit.mode != editModeFields {
		t.Fatalf("expected entity fields editor on dedicated route, got screen=%v mode=%v cmd=%v", model.screen, model.edit.mode, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenEntities || model.edit.mode != editModeEntities || model.status != statusReady {
		t.Fatalf("expected Fields esc to return to Entities route, got screen=%v mode=%v status=%v cmd=%v", model.screen, model.edit.mode, model.status, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenServices || model.serviceContext != serviceResourceEntities {
		t.Fatalf("expected Entities esc to return to Services context, got screen=%v context=%v cmd=%v", model.screen, model.serviceContext, cmd)
	}

	model.serviceContext = serviceResourceValueObjects
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.screen != screenValueObjects || model.status != statusEditing {
		t.Fatalf("expected Services Value Objects context to open dedicated route, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	if cmd != nil || !model.valueObjectsEdit.rulesOpen {
		t.Fatalf("expected rules editor on dedicated Value Objects route, got rulesOpen=%v cmd=%v", model.valueObjectsEdit.rulesOpen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenValueObjects || model.status != statusReady || model.valueObjectsEdit.rulesOpen {
		t.Fatalf("expected Rules esc to return to Value Objects route, got screen=%v status=%v rulesOpen=%v cmd=%v", model.screen, model.status, model.valueObjectsEdit.rulesOpen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenServices || model.serviceContext != serviceResourceValueObjects {
		t.Fatalf("expected Value Objects esc to return to Services context, got screen=%v context=%v cmd=%v", model.screen, model.serviceContext, cmd)
	}
}

func TestModelUpdateKeepsDedicatedRoutesLockedAfterSaveRefreshFailure(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{Services: []application.ServiceSummary{{Name: "CatalogService", Entities: []application.EntitySummary{{Name: "Product"}}}}}
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.openScreen(screenEntities)
	model.status = statusFailed
	model.errContext = "Refresh after save"

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.status != statusFailed || model.screen != screenEntities {
		t.Fatalf("expected stale entity route to stay locked, got status=%v screen=%v cmd=%v", model.status, model.screen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	model = updated.(Model)
	if cmd != nil || model.screen != screenEntities || model.status != statusFailed {
		t.Fatalf("expected stale numeric navigation to stay locked, got status=%v screen=%v cmd=%v", model.status, model.screen, cmd)
	}
}

func TestModelUpdateNavigatesSteps(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

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
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepEntities {
		t.Fatalf("expected ] to move to entities step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepValueObjects {
		t.Fatalf("expected ] to move to value objects step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepPreview {
		t.Fatalf("expected ] to move to preview step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepValueObjects {
		t.Fatalf("expected shift+tab to move back to value objects step, got step=%v cmd=%v", model.currentStep, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	model = updated.(Model)
	if cmd != nil || model.currentStep != stepEntities {
		t.Fatalf("expected [ to move back to entities step, got step=%v cmd=%v", model.currentStep, cmd)
	}
}

func TestModelUpdateIgnoresStepNavigationWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
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

	model := workspaceModel(plan, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil)
	model.currentStep = stepPreview
	view := model.View()

	assertContains(t, view, "DANGER replacement removes 4 previous generated file(s)")
	assertContains(t, view, "old-1.txt, old-2.txt, old-3.txt, and 1 more")
}

func TestModelViewShowsPlannedFileRangeAndCursor(t *testing.T) {
	view := modelOnStep(plannedFilesPlan(6), stepPreview).View()

	assertContains(t, view, "Rows 1-5/6 filter=all")
	assertContains(t, view, "Focus 1/6 [CREATE] file-01.txt")
	assertContains(t, view, "#   Action     Path")
	assertContains(t, view, ">  1 CREATE")
	assertContains(t, view, "   5 CREATE")
	assertNotContains(t, view, "file-06.txt")
}

func TestModelUpdateMovesPlannedFileCursorAndWindow(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(7), stepPreview)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	assertContains(t, model.View(), "Rows 1-5/7 filter=all")
	assertContains(t, model.View(), ">  2 CREATE")

	for range 4 {
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		model = updated.(Model)
		if cmd != nil {
			t.Fatal("expected no command")
		}
	}
	view := model.View()
	assertContains(t, view, "Rows 2-6/7 filter=all")
	assertContains(t, view, ">  6 CREATE")
	assertNotContains(t, view, "file-01.txt")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Rows 2-6/7 filter=all")
	assertContains(t, view, ">  5 CREATE")
}

func TestModelUpdateClampsPlannedFileNavigationBounds(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(3), stepPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Rows 1-3/3 filter=all")
	assertContains(t, view, ">  1 CREATE")

	for range 5 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	view = model.View()
	assertContains(t, view, "Rows 1-3/3 filter=all")
	assertContains(t, view, ">  3 CREATE")
}

func TestModelUpdateSupportsPlannedFileHomeEndAndPageKeys(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(12), stepPreview)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model = updated.(Model)
	view := model.View()
	assertContains(t, view, "Rows 2-6/12 filter=all")
	assertContains(t, view, ">  6 CREATE")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Rows 1-5/12 filter=all")
	assertContains(t, view, ">  1 CREATE")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Rows 8-12/12 filter=all")
	assertContains(t, view, "> 12 CREATE")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Rows 1-5/12 filter=all")
	assertContains(t, view, ">  1 CREATE")
}

func TestModelUpdateWindowSizeChangesVisibleFileRange(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(20), stepPreview)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view := model.View()
	assertContains(t, view, "Rows 1-6/20 filter=all")
	assertNotContains(t, view, "file-07.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	assertContains(t, view, "Rows 1-12/20 filter=all")
	assertContains(t, view, "  12 CREATE")
	assertNotContains(t, view, "file-13.txt")

	updated, cmd = model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	view = model.View()
	if model.windowRows != 3 {
		t.Fatalf("expected 3 visible file rows at height 19, got %d", model.windowRows)
	}
	assertNotContains(t, view, "file-04.txt")
}

func TestModelUpdateClampsNavigationAfterResize(t *testing.T) {
	model := modelOnStep(plannedFilesPlan(20), stepPreview)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(Model)
	assertContains(t, model.View(), "Rows 9-20/20 filter=all")
	assertContains(t, model.View(), "> 20 CREATE")

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command")
	}
	if model.fileCursor != 19 || model.fileOffset != 17 {
		t.Fatalf("expected resized file window to keep cursor 19 and offset 17, got cursor=%d offset=%d", model.fileCursor, model.fileOffset)
	}
	view := model.View()
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
	assertContains(t, view, "Rows 1-2/2 filter=create")
	assertContains(t, view, "Filter a cycles filters")
	assertContains(t, view, "Focus 1/2 [CREATE] create-1.txt")
	assertContains(t, view, ">  1 CREATE")
	assertNotContains(t, view, "replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Focus 2/2 [CREATE] create-2.txt")
	assertContains(t, view, ">  2 CREATE")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Rows 1-2/2 filter=replace")
	assertContains(t, view, "Focus 1/2 [REPLACE] replace-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	view = model.View()
	assertContains(t, view, "Rows 1-1/1 filter=unchanged")
	assertContains(t, view, "Focus 1/1 [UNCHANGED] unchanged-1.txt")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	assertContains(t, model.View(), "Rows 1-5/5 filter=all")
	assertNotContains(t, model.View(), "Filter a cycles filters")
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
			model := workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
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
			model := workspaceModel(plannedFilesPlan(6), application.GenerateRequest{}, nil, nil, nil)
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
			_, cmd := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(tt.msg)

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
			model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
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
			model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
			model.status = statusRefreshing

			_, cmd := model.Update(tt.msg)

			if cmd == nil {
				t.Fatal("expected quit command while refreshing")
			}
		})
	}
}

func TestModelViewShowsRefreshWaitHelpOnly(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil)
	model.status = statusRefreshing

	view := model.View()

	assertContains(t, view, "Microgen REFRESHING")
	assertContains(t, view, "Refreshing plan")
	assertContains(t, view, "refreshing plan | controls paused")
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
		{name: "success ctrl+c", finishMsg: generationFinishedMsg{}, msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
		{name: "failure q", finishMsg: generationFinishedMsg{err: errors.New("write failed")}, msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "failure ctrl+c", finishMsg: generationFinishedMsg{err: errors.New("write failed")}, msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
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
	_, cmd := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if cmd != nil {
		t.Fatal("expected no command")
	}
}

func TestModelUpdateStartsGenerationOnConfirmedKey(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	model := workspaceModel(application.GenerationPlan{}, request, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
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
	assertContains(t, view, "Microgen GENERATING")
	assertContains(t, view, "Generating files")
	assertContains(t, view, "generating files | controls paused")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "Exit: q/esc/ctrl+c")
}

func TestModelUpdateKeepsGenerationStateCurrent(t *testing.T) {
	tests := []struct {
		name       string
		startModel func() Model
		msg        tea.Msg
		wantStatus modelStatus
		wantScreen workspaceScreen
		wantOutput string
		wantErr    bool
	}{
		{
			name: "start clears stale result and reports progress",
			startModel: func() Model {
				model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
					return application.GenerateResult{Plan: application.GenerationPlan{FileCount: 3}, OutputDir: "out/current"}, nil
				}, nil)
				model.status = statusFailed
				model.err = errors.New("previous generation failed")
				model.errContext = "Generation"
				model.result = application.GenerateResult{Plan: application.GenerationPlan{FileCount: 9}, OutputDir: "out/stale"}
				model.message = "Generated 9 files in out/stale."
				return model
			},
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}},
			wantStatus: statusGenerating,
			wantScreen: screenGenerate,
			wantOutput: "",
		},
		{
			name: "success stores visible result without in-progress state",
			startModel: func() Model {
				model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
				model.status = statusGenerating
				return model
			},
			msg:        generationFinishedMsg{result: application.GenerateResult{Plan: application.GenerationPlan{FileCount: 3}, OutputDir: "out/current"}},
			wantStatus: statusGenerated,
			wantScreen: screenResult,
			wantOutput: "out/current",
		},
		{
			name: "failure clears stale success result and leaves visible failure",
			startModel: func() Model {
				model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
				model.status = statusGenerating
				model.result = application.GenerateResult{Plan: application.GenerationPlan{FileCount: 9}, OutputDir: "out/stale"}
				model.message = "Generated 9 files in out/stale."
				return model
			},
			msg:        generationFinishedMsg{err: errors.New("write failed")},
			wantStatus: statusFailed,
			wantScreen: screenResult,
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, cmd := tt.startModel().Update(tt.msg)
			updatedModel := updated.(Model)

			if updatedModel.status != tt.wantStatus {
				t.Fatalf("expected status %v, got %v", tt.wantStatus, updatedModel.status)
			}
			if updatedModel.activeScreen() != tt.wantScreen {
				t.Fatalf("expected screen %v, got %v", tt.wantScreen, updatedModel.activeScreen())
			}
			if updatedModel.result.OutputDir != tt.wantOutput {
				t.Fatalf("expected output %q, got %q", tt.wantOutput, updatedModel.result.OutputDir)
			}
			if (updatedModel.err != nil) != tt.wantErr {
				t.Fatalf("expected err presence %v, got %v", tt.wantErr, updatedModel.err)
			}
			view := updatedModel.View()
			assertNotContains(t, view, "out/stale")
			assertNotContains(t, view, "Generated 9 files")
			if tt.wantStatus == statusGenerating && cmd == nil {
				t.Fatal("expected generation command")
			}
			if tt.wantStatus != statusGenerating && cmd != nil {
				t.Fatal("expected no command after generation completion")
			}
		})
	}
}

func TestTViewFinishGenerationClearsStaleAsyncState(t *testing.T) {
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil)
	ui.result = application.GenerateResult{Plan: application.GenerationPlan{FileCount: 9}, OutputDir: "out/stale"}
	ui.message = "Generated 9 files in out/stale."
	ui.generating = true

	ui.finishGeneration(application.GenerateResult{}, errors.New("write failed"))

	if ui.generating || ui.result.OutputDir != "" || ui.err == nil || ui.screen != tviewScreenResult {
		t.Fatalf("expected failed generation to clear stale result and open Result, got generating=%v result=%#v err=%v screen=%d", ui.generating, ui.result, ui.err, ui.screen)
	}
	if !strings.Contains(ui.message, "Generation failed") || strings.Contains(ui.message, "out/stale") {
		t.Fatalf("expected visible failure without stale output, got %q", ui.message)
	}
}

func TestModelUpdateStartsRefreshOnRefreshKey(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	model := workspaceModel(application.GenerationPlan{}, request, func(actual application.GenerateRequest) (application.GenerationPlan, error) {
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
	assertContains(t, view, "Microgen REFRESHING")
	assertContains(t, view, "Refreshing plan")
	assertContains(t, view, "refreshing plan | controls paused")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "g to generate")
}

func TestModelUpdateRecordsRefreshSuccess(t *testing.T) {
	model := workspaceModel(plannedFilesPlan(10), application.GenerateRequest{}, nil, nil, nil)
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
	assertContains(t, view, "Rows 1-2/2 filter=all")
	assertContains(t, view, ">  2 CREATE")
}

func TestModelUpdateRecordsRefreshFailureAndAllowsRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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
	assertContains(t, view, "Microgen FAILED")
	assertContains(t, view, "g Retry generation")
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
		model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
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
		model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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
	model := workspaceModel(plan, request, nil, nil, func(actual application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
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
	assertContains(t, model.View(), "Use the Services, Value Objects, Entities, and Fields routes for resource editing.")

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
	assertContains(t, model.View(), "Microgen SAVING")
	assertContains(t, model.View(), "Saving settings")
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
	if capturedSettings.SolutionName != "CatalogPlatform" || capturedSettings.SolutionDescription != "New description" || capturedSettings.TargetFramework != "9" {
		t.Fatalf("expected edited text settings, got %#v", capturedSettings)
	}
	if capturedSettings.GatewayEnabled == nil || *capturedSettings.GatewayEnabled {
		t.Fatalf("expected explicit disabled gateway setting, got %#v", capturedSettings.GatewayEnabled)
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
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0", "net8.0"})
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
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
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
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
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
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
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
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
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
	model := workspaceModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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
	assertContains(t, view, "Microgen FAILED")
	assertContains(t, view, "r Retry refresh")
	assertContains(t, view, "Solution CatalogPlatform")
	assertContains(t, view, "Description New description")
	assertContains(t, view, "Target net9.0")
	assertContains(t, view, "Settings saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	assertContains(t, view, "Readiness is stale. Saved settings need a successful plan refresh before generation.")
	assertContains(t, view, "r Retry plan refresh. Other actions stay locked until refresh succeeds.")
	assertContains(t, view, "locked | r retry refresh | q quit")
	assertNotContains(t, view, "Readiness project=yes")
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
	model := workspaceModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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
	model := workspaceModel(plan, request, nil, nil, nil)
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
	assertContains(t, model.View(), "Services saved. Plan refreshed. Configure value objects before entities and fields.")
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

func TestModelUpdateServicesRenameAcceptsShortcutLetters(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ServiceNames: []string{"ProductService"}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("ProductService")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("rdgasService")})
	model = updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command while typing service name")
	}
	if !model.servicesEdit.renaming || model.servicesEdit.services[0].string() != "rdgasService" {
		t.Fatalf("expected shortcut letters to be inserted during rename, got %#v", model.servicesEdit)
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
	assertContains(t, model.View(), "Selected service: OrderService")
	assertContains(t, model.View(), "Entities: 2")
	assertContains(t, model.View(), "Fields: 4")
	assertContains(t, model.View(), "Value objects: 0")
	assertContains(t, model.View(), "References: 0")

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
	assertContains(t, model.View(), "Entities saved. Plan refreshed. Press f in the Entities editor to edit fields.")
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
	assertContains(t, model.View(), "Fields saved. Plan refreshed. Review the generation plan.")
}

func TestModelUpdateFieldsEditCancelReturnsToServicesWorkspace(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, EntityCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, Entities: []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.status != statusReady || model.activeScreen() != screenEntities || model.serviceContext != serviceResourceEntities {
		t.Fatalf("expected esc to return to Entities workspace, got cmd=%v model=%#v", cmd, model)
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

func TestModelUpdateOpensValueObjectsEditorAndSavesChanges(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated", Force: true}
	plan := plannedFilesPlan(2)
	plan.Config = application.ConfigSummary{
		ServiceCount:     1,
		ValueObjectCount: 2,
		Services:         []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, ValueObjectNames: []string{"ProductName", "LegacyName"}, ValueObjectReferences: []application.ValueObjectReferenceSummary{{ValueObjectName: "ProductName", EntityName: "Product", FieldName: "Name"}}}},
	}
	updatedPlan := plannedFilesPlan(4)
	updatedPlan.Config = application.ConfigSummary{
		ServiceCount:     1,
		ValueObjectCount: 2,
		Services:         []application.ServiceSummary{{Name: "ProductService", EntityNames: []string{"Product"}, ValueObjectNames: []string{"CatalogName", "ValueObject3"}}},
	}
	var capturedSettings application.ValueObjectSettings
	model := modelOnStep(plan, stepServices)
	model.request = request
	model.updateValueObjects = func(actual application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		capturedSettings = settings
		return application.UpdateValueObjectSettingsResult{Saved: true, Plan: updatedPlan}, nil
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	if cmd != nil || model.status != statusEditing || model.edit.mode != editModeValueObjects || model.valueObjectsEdit.serviceName != "ProductService" {
		t.Fatalf("expected value object editor, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Editing value objects for ProductService")
	assertContains(t, model.View(), "References:")
	assertContains(t, model.View(), "ProductName <- Product.Name")
	assertContains(t, model.View(), "Keys: up/down select, a add, r rename, o rules, d delete, enter save, esc cancel.")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("ProductName")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("CatalogName")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.valueObjectsEdit.renaming || model.valueObjectsEdit.valueObjects[0].name.string() != "CatalogName" {
		t.Fatalf("expected local value object rename confirmation, got %#v", model.valueObjectsEdit)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if model.valueObjectsEdit.selected != 2 || model.valueObjectsEdit.valueObjects[2].name.string() != "ValueObject3" {
		t.Fatalf("expected added placeholder value object selected, got %#v", model.valueObjectsEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if got := valueObjectEditNames(model.valueObjectsEdit.valueObjects); !reflect.DeepEqual(got, []string{"CatalogName", "ValueObject3"}) {
		t.Fatalf("expected legacy value object deleted, got %#v", got)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected saving state and command, got status=%v cmd=%v", model.status, cmd)
	}
	assertContains(t, model.View(), "Saving value objects...")
	msg := cmd()
	finished, ok := msg.(valueObjectsFinishedMsg)
	if !ok {
		t.Fatalf("expected valueObjectsFinishedMsg, got %#v", msg)
	}
	if finished.err != nil || finished.result.Plan.FileCount != 4 {
		t.Fatalf("expected successful value objects message, got %#v", finished)
	}
	wantValueObjects := []application.ValueObjectNameSetting{{OriginalName: "ProductName", Name: "CatalogName", Type: "string", Validations: application.ValidationRuleSettings{Required: boolPtr(true), MinLength: intPtr(1), MaxLength: intPtr(100), ValidExample: stringPtr("Sample")}}, {Name: "ValueObject3", Type: "string", Validations: application.ValidationRuleSettings{Required: boolPtr(true), MinLength: intPtr(1), MaxLength: intPtr(100), ValidExample: stringPtr("Sample")}}}
	if capturedSettings.ServiceName != "ProductService" || !reflect.DeepEqual(capturedSettings.ValueObjects, wantValueObjects) {
		t.Fatalf("expected captured value object settings, got %#v", capturedSettings)
	}
	updated, cmd = model.Update(finished)
	model = updated.(Model)
	if cmd != nil || model.status != statusReady || model.plan.FileCount != 4 || model.plan.Config.Services[0].ValueObjectNames[0] != "CatalogName" {
		t.Fatalf("expected ready state with refreshed value objects plan, got cmd=%v model=%#v", cmd, model)
	}
	assertContains(t, model.View(), "Value objects saved. Plan refreshed. Continue with entities and fields.")
}

func TestModelUpdateValueObjectsEditCancelKeepsPlan(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil || model.status != statusReady || !reflect.DeepEqual(model.plan.Config.Services[0].ValueObjectNames, []string{"ProductName"}) {
		t.Fatalf("expected cancel to keep original value objects, got cmd=%v model=%#v", cmd, model)
	}
}

func TestModelUpdateValueObjectsEditCancelRestoresGeneratedStatus(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}}}}
	model := modelOnStep(plan, stepServices)
	model.status = statusGenerated
	model.result = application.GenerateResult{OutputDir: "/tmp/generated", Plan: application.GenerationPlan{FileCount: 1}}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	updated, cmd := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command on cancel")
	}
	if model.status != statusGenerated || model.result.OutputDir != "/tmp/generated" {
		t.Fatalf("expected cancel to restore generated state, got %#v", model)
	}
}

func TestModelUpdateValueObjectsRenameInsertsShortcutRunes(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	for range len([]rune("ProductName")) {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	if len(model.valueObjectsEdit.valueObjects) != 1 || model.valueObjectsEdit.valueObjects[0].name.string() != "adv" {
		t.Fatalf("expected shortcut runes to edit value object name during rename, got %#v", model.valueObjectsEdit)
	}
}

func TestModelUpdateValueObjectRulesEditsStringRulesAndSaves(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}, ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string", Validations: application.ValidationRuleSummary{Required: boolPtr(true), MinLength: intPtr(1), MaxLength: intPtr(100), ValidExample: stringPtr("Sample")}, RulesLabel: "required, min=1, max=100, validExample"}}}}}
	var captured application.ValueObjectSettings
	model := modelOnStep(plan, stepServices)
	model.updateValueObjects = func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		captured = settings
		return application.UpdateValueObjectSettingsResult{Saved: true, Plan: plan}, nil
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	assertContains(t, model.View(), "Editing rules for ProductService/ProductName")
	assertContains(t, model.View(), "required: yes")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.status != statusSaving || cmd == nil {
		t.Fatalf("expected rules save command, got status=%v cmd=%v", model.status, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady {
		t.Fatalf("expected ready after string rules save, got %#v", model)
	}
	if len(captured.ValueObjects) != 1 || captured.ValueObjects[0].Type != "string" || captured.ValueObjects[0].Validations.Required != nil || captured.ValueObjects[0].Validations.MinLength == nil || *captured.ValueObjects[0].Validations.MinLength != 3 {
		t.Fatalf("expected captured string rule settings, got %#v", captured)
	}
}

func TestModelUpdateValueObjectRulesEditsNumericBounds(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductPrice"}, ValueObjects: []application.ValueObjectSummary{{Name: "ProductPrice", Type: "decimal", RulesLabel: "no rules"}}}}}
	var captured application.ValueObjectSettings
	model := modelOnStep(plan, stepServices)
	model.updateValueObjects = func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		captured = settings
		return application.UpdateValueObjectSettingsResult{Saved: true, Plan: plan}, nil
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("999999.99")})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || captured.ValueObjects[0].Validations.Minimum == nil || *captured.ValueObjects[0].Validations.Minimum != "0" || captured.ValueObjects[0].Validations.Maximum == nil || *captured.ValueObjects[0].Validations.Maximum != "999999.99" {
		t.Fatalf("expected numeric bounds save, model=%#v captured=%#v", model, captured)
	}
}

func TestValidationRuleSettingsFromEditSerializesSupportedScalarRules(t *testing.T) {
	tests := []struct {
		name        string
		typeName    string
		rules       valueObjectRuleEditState
		assertRules func(t *testing.T, settings application.ValidationRuleSettings)
	}{
		{
			name:     "long bounds",
			typeName: "long",
			rules:    valueObjectRuleEditState{minimum: newTextField("0"), maximum: newTextField("9223372036854775807")},
			assertRules: func(t *testing.T, settings application.ValidationRuleSettings) {
				t.Helper()
				if settings.Minimum == nil || *settings.Minimum != "0" || settings.Maximum == nil || *settings.Maximum != "9223372036854775807" {
					t.Fatalf("expected long bounds, got %+v", settings)
				}
			},
		},
		{
			name:     "double bounds",
			typeName: "double",
			rules:    valueObjectRuleEditState{minimum: newTextField("0"), maximum: newTextField("999999.99")},
			assertRules: func(t *testing.T, settings application.ValidationRuleSettings) {
				t.Helper()
				if settings.Minimum == nil || *settings.Minimum != "0" || settings.Maximum == nil || *settings.Maximum != "999999.99" {
					t.Fatalf("expected double bounds, got %+v", settings)
				}
			},
		},
		{
			name:     "DateTime not default",
			typeName: "DateTime",
			rules:    valueObjectRuleEditState{notDefault: true},
			assertRules: func(t *testing.T, settings application.ValidationRuleSettings) {
				t.Helper()
				if settings.NotDefault == nil || !*settings.NotDefault {
					t.Fatalf("expected DateTime notDefault, got %+v", settings)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := validationRuleSettingsFromEdit(tt.typeName, tt.rules)
			tt.assertRules(t, settings)
		})
	}
}

func TestRulesLabelForEditDisplaysSupportedScalarRules(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		rules    valueObjectRuleEditState
		want     string
	}{
		{
			name:     "long bounds",
			typeName: "long",
			rules:    valueObjectRuleEditState{minimum: newTextField("0"), maximum: newTextField("9223372036854775807")},
			want:     "minimum=0, maximum=9223372036854775807",
		},
		{
			name:     "double bounds",
			typeName: "double",
			rules:    valueObjectRuleEditState{minimum: newTextField("0"), maximum: newTextField("999999.99")},
			want:     "minimum=0, maximum=999999.99",
		},
		{
			name:     "DateTime not default",
			typeName: "DateTime",
			rules:    valueObjectRuleEditState{notDefault: true},
			want:     "notDefault",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rulesLabelForEdit(tt.typeName, tt.rules); got != tt.want {
				t.Fatalf("expected label %q, got %q", tt.want, got)
			}
		})
	}
}

func TestModelUpdateValueObjectRulesTogglesGuidNotEmpty(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductId"}, ValueObjects: []application.ValueObjectSummary{{Name: "ProductId", Type: "Guid", RulesLabel: "no rules"}}}}}
	var captured application.ValueObjectSettings
	model := modelOnStep(plan, stepServices)
	model.updateValueObjects = func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		captured = settings
		return application.UpdateValueObjectSettingsResult{Saved: true, Plan: plan}, nil
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	assertContains(t, model.View(), "notEmpty: no")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.status != statusReady || captured.ValueObjects[0].Validations.NotEmpty == nil || !*captured.ValueObjects[0].Validations.NotEmpty {
		t.Fatalf("expected notEmpty toggle save, model=%#v captured=%#v", model, captured)
	}
}

func TestModelUpdateValueObjectRulesBackCancelAndShortcutTextSafety(t *testing.T) {
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}, ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string", RulesLabel: "no rules"}}}}}
	model := modelOnStep(plan, stepServices)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("bear")})
	model = updated.(Model)
	if !model.valueObjectsEdit.rulesOpen || model.valueObjectsEdit.valueObjects[0].rules.pattern.string() != "bear" {
		t.Fatalf("expected shortcut runes to edit pattern text, got %#v", model.valueObjectsEdit)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = updated.(Model)
	if model.status != statusEditing || model.edit.mode != editModeValueObjects || model.valueObjectsEdit.rulesOpen {
		t.Fatalf("expected b to return to value object list, got %#v", model)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.status != statusReady {
		t.Fatalf("expected esc from rules to cancel editing, got cmd=%v model=%#v", cmd, model)
	}
}

func TestModelUpdateValueObjectsSaveSuccessWithRefreshFailureBlocksUntilRetry(t *testing.T) {
	refreshErr := errors.New("plan failed")
	retries := 0
	plan := plannedFilesPlan(1)
	plan.Config = application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 1, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName"}}}}
	savedConfig := application.ConfigSummary{ServiceCount: 1, ValueObjectCount: 2, Services: []application.ServiceSummary{{Name: "ProductService", ValueObjectNames: []string{"ProductName", "Sku"}}}}
	refreshedPlan := plannedFilesPlan(2)
	refreshedPlan.Config = savedConfig
	model := modelOnStep(plan, stepServices)
	model.planFunc = func(application.GenerateRequest) (application.GenerationPlan, error) {
		retries++
		return refreshedPlan, nil
	}
	model.updateValueObjects = func(application.GenerateRequest, application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
		return application.UpdateValueObjectSettingsResult{Saved: true, Config: savedConfig, PlanError: refreshErr}, nil
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	if model.status != statusFailed || model.err != refreshErr || model.errContext != "Refresh after save" || len(model.plan.Config.Services[0].ValueObjectNames) != 2 {
		t.Fatalf("expected value objects refresh-after-save failure state, got %#v", model)
	}
	view := model.View()
	assertContains(t, view, "Value objects saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	updated, editCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if editCmd != nil || updated.(Model).status != statusFailed {
		t.Fatalf("expected value object edit to be blocked after save refresh failure, got status=%v cmd=%v", updated.(Model).status, editCmd)
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

func TestModelUpdateBlocksValueObjectsEditWhileBusy(t *testing.T) {
	for _, status := range []modelStatus{statusRefreshing, statusGenerating, statusSaving} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			model := modelOnStep(plannedFilesPlan(1), stepServices)
			model.status = status

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
			updatedModel := updated.(Model)

			if cmd != nil || updatedModel.status != status || updatedModel.edit.mode == editModeValueObjects {
				t.Fatalf("expected value object edit to be ignored while busy, got status=%v mode=%v cmd=%v", updatedModel.status, updatedModel.edit.mode, cmd)
			}
		})
	}
}

func TestModelUpdateBlocksQuitAndActionsWhileSavingOrEditing(t *testing.T) {
	for _, msg := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}} {
		model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil)
		model.status = statusSaving
		updated, cmd := model.Update(msg)
		if cmd != nil || updated.(Model).status != statusSaving {
			t.Fatalf("expected quit to be blocked while saving for %q", msg.String())
		}
	}
	model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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

	updated, cmd := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil).Update(generationFinishedMsg{result: result})
	updatedModel := updated.(Model)

	if cmd != nil {
		t.Fatal("expected no command")
	}
	if updatedModel.status != statusGenerated || updatedModel.result.OutputDir != result.OutputDir || updatedModel.plan.FileCount != 3 {
		t.Fatalf("expected generated state, got %#v", updatedModel)
	}
	view := updatedModel.View()
	assertContains(t, view, "Microgen GENERATED")
	assertContains(t, view, "r Refresh")
	assertContains(t, view, "Generated 3 files written to /tmp/generated.")
	assertContains(t, view, "Next cd /tmp/generated && dotnet build")
	assertContains(t, view, "WARNING existing warning")
	assertContains(t, view, "result r refresh esc generate")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "g to generate")
}

func TestModelUpdateRecordsGenerationFailureAndAllowsRetry(t *testing.T) {
	generationErr := errors.New("write failed")
	retries := 0
	model := workspaceModel(application.GenerationPlan{}, application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
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
	assertContains(t, view, "Microgen FAILED")
	assertContains(t, view, "g Retry generation")
	assertContains(t, view, "FAILED Generation failed: write failed")
	assertContains(t, view, "g Retry generation, or r refresh the plan first.")
	assertContains(t, view, "result g retry esc generate r refresh")

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

func TestModelUpdatePreviewGenerateResultWorkflow(t *testing.T) {
	request := application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated"}
	plan := plannedFilesPlan(2)
	plan.OutputDir = request.OutputDir
	model := workspaceModel(plan, request, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
		if actual != request {
			t.Fatalf("expected request %#v, got %#v", request, actual)
		}
		return application.GenerateResult{OutputDir: request.OutputDir, Plan: plan}, nil
	}, nil)
	model.openScreen(screenPreview)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if cmd != nil || model.screen != screenGenerate || model.status != statusReady {
		t.Fatalf("expected Preview to continue to Generate without writing, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if cmd == nil || model.screen != screenGenerate || model.status != statusGenerating {
		t.Fatalf("expected Generate confirmation to start writing, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(Model)
	if cmd != nil || model.screen != screenResult || model.currentStep != stepResult || model.status != statusGenerated {
		t.Fatalf("expected successful generation to open Result, got screen=%v step=%v status=%v cmd=%v", model.screen, model.currentStep, model.status, cmd)
	}
	assertContains(t, model.View(), "Result")
	assertContains(t, model.View(), "2 files written to /tmp/generated")
}

func TestModelUpdateResultFailureBackAndRetry(t *testing.T) {
	generationErr := errors.New("write failed")
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
		return application.GenerateResult{}, generationErr
	}, nil)

	updated, cmd := model.Update(generationFinishedMsg{err: generationErr})
	model = updated.(Model)
	if cmd != nil || model.screen != screenResult || model.status != statusFailed {
		t.Fatalf("expected generation failure Result state, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}
	assertContains(t, model.View(), "FAILED Generation failed: write failed")
	assertContains(t, model.View(), "esc back to Generate")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.screen != screenGenerate {
		t.Fatalf("expected Result esc to return Generate, got screen=%v cmd=%v", model.screen, cmd)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if cmd == nil || model.status != statusGenerating || model.screen != screenGenerate {
		t.Fatalf("expected retry to start from Generate, got screen=%v status=%v cmd=%v", model.screen, model.status, cmd)
	}
}

func TestModelUpdateBlocksStaleAndForceUnsafeGeneration(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*Model)
		message string
		status  modelStatus
	}{
		{
			name: "stale plan",
			prepare: func(model *Model) {
				model.status = statusFailed
				model.errContext = "Refresh after save"
			},
			message: "Readiness is stale. Saved settings need a successful plan refresh before generation.",
			status:  statusFailed,
		},
		{
			name: "force confirmation",
			prepare: func(model *Model) {
				model.plan.ForceRequired = true
			},
			message: "Generation is locked until --force is confirmed",
			status:  statusReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, func(application.GenerateRequest) (application.GenerateResult, error) {
				t.Fatal("generation should remain blocked")
				return application.GenerateResult{}, nil
			}, nil)
			model.openScreen(screenGenerate)
			tt.prepare(&model)

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
			model = updated.(Model)
			if cmd != nil || model.status != tt.status {
				t.Fatalf("expected blocked generation without command, got status=%v cmd=%v", model.status, cmd)
			}
			assertContains(t, model.View(), tt.message)
		})
	}
}

func TestModelStalePlanRetryFailureKeepsGenerationLocked(t *testing.T) {
	refreshErr := errors.New("refresh failed again")
	model := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return application.GenerationPlan{}, refreshErr
	}, func(application.GenerateRequest) (application.GenerateResult, error) {
		t.Fatal("generation should remain blocked while the plan is stale")
		return application.GenerateResult{}, nil
	}, nil)
	model.openScreen(screenGenerate)
	model.status = statusFailed
	model.err = errors.New("initial refresh failed")
	model.errContext = "Refresh after save"

	updated, retryCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(Model)
	if retryCmd == nil || model.status != statusRefreshing {
		t.Fatalf("expected stale-plan refresh retry, got status=%v cmd=%v", model.status, retryCmd)
	}
	updated, _ = model.Update(retryCmd())
	model = updated.(Model)
	if model.status != statusFailed || !model.postSaveRefreshFailed() || model.err != refreshErr {
		t.Fatalf("expected failed retry to preserve stale lock, got status=%v stale=%v err=%v", model.status, model.postSaveRefreshFailed(), model.err)
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if command != nil || model.status != statusFailed {
		t.Fatalf("expected generation to stay blocked after failed retry, got status=%v cmd=%v", model.status, command)
	}
}

func TestModelForceConfirmationRemainsExplicitBeforeGeneration(t *testing.T) {
	called := false
	plan := plannedFilesPlan(1)
	plan.ForceRequired = true
	model := workspaceModel(plan, application.GenerateRequest{Force: true}, nil, func(request application.GenerateRequest) (application.GenerateResult, error) {
		called = true
		if !request.Force {
			t.Fatal("expected explicit force to reach generation callback")
		}
		return application.GenerateResult{Plan: plan, OutputDir: "/tmp/generated"}, nil
	}, nil)
	model.openScreen(screenGenerate)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(Model)
	if command == nil || model.status != statusGenerating {
		t.Fatalf("expected explicit force to allow generation, got status=%v cmd=%v", model.status, command)
	}
	updated, _ = model.Update(command())
	model = updated.(Model)
	if !called || model.status != statusGenerated {
		t.Fatalf("expected forced generation to complete, got called=%v status=%v", called, model.status)
	}
}

func TestModelViewsRemainSafeAtNarrowTerminalWidths(t *testing.T) {
	for _, screen := range []workspaceScreen{screenOverview, screenProject, screenServices, screenEntities, screenValueObjects, screenPreview, screenGenerate, screenResult} {
		for _, width := range []int{0, 1, 10, 19} {
			t.Run(fmt.Sprintf("%s width %d", screen.label(), width), func(t *testing.T) {
				model := workspaceModel(wizardPlan(), application.GenerateRequest{ConfigPath: "config.json", OutputDir: "/tmp/generated"}, nil, nil, nil)
				model.openScreen(screen)
				updated, command := model.Update(tea.WindowSizeMsg{Width: width, Height: 1})
				if command != nil {
					t.Fatal("expected no command from window resize")
				}
				view := stripANSI(updated.(Model).View())
				if strings.TrimSpace(view) == "" {
					t.Fatal("expected non-empty narrow view")
				}
				if strings.ContainsRune(view, '\x1b') {
					t.Fatalf("expected stripped narrow view to contain no raw terminal escape sequence: %q", view)
				}
			})
		}
	}
}

func TestModelViewWorkflowScreensShowResponsiveContent(t *testing.T) {
	plan := plannedFilesPlan(3)
	plan.OutputDir = "/tmp/generated"
	plan.Files[1].Action = "replace"
	plan.Files[2].Action = "unchanged"
	plan.DeletedFiles = []string{"old.cs"}
	plan.ExtraFileCount = 1
	model := workspaceModel(plan, application.GenerateRequest{OutputDir: plan.OutputDir}, nil, nil, nil)

	model.openScreen(screenPreview)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60})
	model = updated.(Model)
	view := stripANSI(model.View())
	assertContains(t, view, "Readiness")
	assertContains(t, view, "Change counts created=1, replaced=1, unchanged=1, deleted=1")
	assertContains(t, view, "Planned Files")
	assertContains(t, view, "File detail")
	assertContains(t, view, "DANGER replacement removes 1 previous generated file(s)")

	model.openScreen(screenGenerate)
	view = stripANSI(model.View())
	assertContains(t, view, "Readiness checklist")
	assertContains(t, view, "Press g to confirm the write")

	model.status = statusGenerated
	model.result = application.GenerateResult{OutputDir: plan.OutputDir, Plan: plan, Warning: "partial warning"}
	model.openScreen(screenResult)
	view = stripANSI(model.View())
	assertContains(t, view, "Result")
	assertContains(t, view, "3 files written to /tmp/generated")
	assertContains(t, view, "Deleted files")
	assertContains(t, view, "old.cs")
	assertContains(t, view, "WARNING partial warning")
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

func renderedTestLineCount(value string) int {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return 0
	}
	return len(strings.Split(value, "\n"))
}

func lineIndexContaining(lines []string, expected string) int {
	for index, line := range lines {
		if strings.Contains(line, expected) {
			return index
		}
	}
	return -1
}

func assertViewportSize(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d in %q", height, len(lines), view)
	}
	for index, line := range lines {
		if got := len([]rune(line)); got != width {
			t.Fatalf("expected line %d to be %d columns, got %d in %q", index+1, width, got, line)
		}
	}
}

func plannedFilesPlan(count int) application.GenerationPlan {
	files := make([]application.PlannedFile, count)
	for index := range files {
		files[index] = application.PlannedFile{Path: fmt.Sprintf("file-%02d.txt", index+1), Action: "create"}
	}
	return application.GenerationPlan{FileCount: count, Files: files}
}

func longPreviewPlan() application.GenerationPlan {
	longPath := "src/ExtremelyLongServiceNameThatWouldPreviouslyWrapAcrossTerminalRows/ExtremelyLongServiceNameThatWouldPreviouslyWrapAcrossTerminalRows.WebApi/Controllers/ExtremelyLongControllerNameThatNeedsEllipsis.cs"
	plan := plannedFilesPlan(8)
	plan.Config = wizardPlan().Config
	plan.OutputDir = "/tmp/generated"
	plan.OutputAction = "replace"
	plan.ExtraFileCount = 1
	plan.DeletedFiles = []string{longPath}
	for index := range plan.Files {
		plan.Files[index] = application.PlannedFile{Path: fmt.Sprintf("%s/file-%02d.cs", longPath, index+1), Action: "create"}
	}
	plan.FileCount = len(plan.Files)
	return plan
}

func wizardPlan() application.GenerationPlan {
	return application.GenerationPlan{
		Config: application.ConfigSummary{
			SolutionName:        "CommercePlatform",
			SolutionDescription: "Product management.",
			TargetFramework:     "net8.0",
			ServiceCount:        2,
			ServiceNames:        []string{"ProductService", "OrderService"},
			Services: []application.ServiceSummary{
				{
					Name:         "ProductService",
					Entities:     []application.EntitySummary{{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}}},
					ValueObjects: []application.ValueObjectSummary{{Name: "ProductName", Type: "string"}},
				},
				{
					Name:     "OrderService",
					Entities: []application.EntitySummary{{Name: "Order", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}},
				},
			},
		},
	}
}

func wizardPlanWithRelationships() application.GenerationPlan {
	plan := wizardPlan()
	plan.Config.Services[0].Entities = []application.EntitySummary{
		{Name: "Category", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}},
		{Name: "Product", Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}}},
	}
	plan.Config.Services[0].Relationships = []application.RelationshipSummary{{Name: "ProductCategory", Multiplicity: "one-to-many", PrincipalEntity: "Category", DependentEntity: "Product", ForeignKeyName: "CategoryId", ForeignKeyType: "Guid", Required: true, PrincipalNavigation: "Products", DependentNavigation: "Category", Summary: "Category 1-* Product via CategoryId (required)"}}
	return plan
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

func valueObjectEditNames(valueObjects []valueObjectEditItem) []string {
	names := make([]string, len(valueObjects))
	for index, valueObject := range valueObjects {
		names[index] = valueObject.name.string()
	}
	return names
}

type tuiConfigLoaderFunc func(string) (spec.Config, error)

func (loader tuiConfigLoaderFunc) LoadConfig(path string) (spec.Config, error) {
	return loader(path)
}

type tuiConfigSaverFunc func(string, spec.Config) error

func (saver tuiConfigSaverFunc) SaveConfig(path string, cfg spec.Config) error {
	return saver(path, cfg)
}

type tuiGeneratorFunc func(spec.Config) ([]application.GeneratedFile, error)

func (generator tuiGeneratorFunc) Generate(cfg spec.Config) ([]application.GeneratedFile, error) {
	return generator(cfg)
}

type tuiOutputPlannerFunc func(string, []application.GeneratedFile, bool) (application.OutputPlan, error)

func (planner tuiOutputPlannerFunc) PlanOutput(outputDir string, files []application.GeneratedFile, force bool) (application.OutputPlan, error) {
	return planner(outputDir, files, force)
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func modelOnStep(plan application.GenerationPlan, step tuiStep) Model {
	model := workspaceModel(plan, application.GenerateRequest{}, nil, nil, nil)
	model.currentStep = step
	return model
}

func workspaceModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	model := NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions...)
	model.mode = modeWorkspace
	return model
}

func wizardModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	model := NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions...)
	model.mode = modeWizard
	return model
}

func assertNotContains(t *testing.T, value, unexpected string) {
	t.Helper()
	value = stripANSI(value)
	if strings.Contains(value, unexpected) {
		t.Fatalf("expected %q not to contain %q", value, unexpected)
	}
}
