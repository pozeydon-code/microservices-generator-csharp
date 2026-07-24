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
			{Path: "src/ProductService/ProductService.Api/ProductEndpoints.cs", Action: "create"},
			{Path: "src/ProductService/ProductService.Domain/Product.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs", Action: "create"},
			{Path: "tests/ProductService/ProductService.Domain.Tests/ProductTests.cs", Action: "create"},
		},
	}

	model := workspaceModel(plan, application.GenerateRequest{ConfigPath: "microgen.json"}, nil, nil, nil)
	view := model.View()

	assertContains(t, view, "Microgen - READY")
	assertContains(t, view, "Workspace")
	assertContains(t, view, "Primary: g Generate")
	assertContains(t, view, "Navigation")
	assertContains(t, view, "1:Overview")
	assertContains(t, view, "2:Project")
	assertContains(t, view, "3:Services")
	assertContains(t, view, "4:Entities")
	assertContains(t, view, "5:Value Objects")
	assertContains(t, view, "6:Preview")
	assertContains(t, view, "7:Generate")
	assertContains(t, view, "8:Result")
	assertNotContains(t, view, "Wizard")
	assertNotContains(t, view, "Progress 1/5")
	assertContains(t, view, "Source")
	assertContains(t, view, "Source microgen.json (existing JSON)")
	assertContains(t, view, "Output /tmp/generated")
	assertContains(t, view, "Mode replace")
	assertContains(t, view, "Navigate: up/down select route, enter open, h/l switch, ? help.")

	model.currentStep = stepProject
	view = model.View()
	assertContains(t, view, "Solution CommercePlatform")
	assertContains(t, view, "Description Product management.")
	assertContains(t, view, "Target net8.0")
	assertContains(t, view, "Format .sln")
	assertContains(t, view, "e Edit solution name, description, or target framework.")

	model.currentStep = stepServices
	view = model.View()
	assertContains(t, view, "Services workspace")
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
	assertContains(t, view, "Files 1-5 of 6 (filter: all)")
	assertContains(t, view, "Selected: 1/6 [REPLACE] README.md")
	assertContains(t, view, "> [1/6] [REPLACE] README.md")
	assertContains(t, view, "  [5/6] [CREATE] tests/ProductService/ProductService.Api.Tests/ProductEndpointsTests.cs")
	assertContains(t, view, readyHelp)
	assertContains(t, view, "Back: esc | Exit: q/ctrl+c")

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

func TestModelViewShowsPrimaryActionOnce(t *testing.T) {
	view := stripANSI(workspaceModel(plannedFilesPlan(2), application.GenerateRequest{}, nil, nil, nil).View())

	if count := strings.Count(view, "Primary:"); count != 1 {
		t.Fatalf("expected one primary action, got %d in %q", count, view)
	}
}

func TestModelViewShowsBootstrappedConfigSource(t *testing.T) {
	view := workspaceModel(plannedFilesPlan(1), application.GenerateRequest{ConfigPath: "starter.json", ConfigBootstrapped: true}, nil, nil, nil).View()

	assertContains(t, view, "Source starter.json (starter config bootstrapped this run)")
	assertContains(t, view, "Created starter config. Edit project, service, entity, and basic field settings incrementally.")
}

func TestModelDefaultsToWizardMenu(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)

	if model.mode != modeWizard || model.wizardScreen != wizardMenu {
		t.Fatalf("expected wizard menu by default, got mode=%v screen=%v", model.mode, model.wizardScreen)
	}
	view := stripANSI(model.View())
	assertContains(t, view, "Breadcrumb: Wizard / Menu")
	assertContains(t, view, "What would you like to configure?")
	assertContains(t, view, "> Configure project")
	assertContains(t, view, "Footer:")
	assertNotContains(t, view, "Navigation")
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
			model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
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
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
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
	model := NewModel(plan, application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
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
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, func(_ application.GenerateRequest, _ application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
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
	model := NewModel(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
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
	model := NewModel(wizardPlan(), application.GenerateRequest{}, nil, nil, nil, []string{"net10.0", "net9.0", "net8.0"})
	model.enterWizardProject()

	view := stripANSI(model.View())
	assertContains(t, view, "Target framework: net8.0")
	assertContains(t, view, "Available target frameworks")
	assertContains(t, view, "net8.0 (current)")
	assertContains(t, view, "Edit solution name and description")
	assertContains(t, view, "Continue to services")
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
	assertContains(t, view, "Advanced configuration")

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
	assertContains(t, view, "Advanced configuration")
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
	assertContains(t, view, "Advanced configuration")
	assertContains(t, view, "Selected service")
	assertContains(t, view, "Entities: 1 | Fields: 2 | Value objects: 1")
	assertNotContains(t, view, "Navigation")
}

func TestWizardAdvancedWorkspaceHasBackPath(t *testing.T) {
	model := NewModel(plannedFilesPlan(1), application.GenerateRequest{}, nil, nil, nil)
	for range wizardAdvancedWorkspace {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWorkspace || model.screen != screenOverview || model.guidedWorkspace {
		t.Fatalf("expected advanced workspace, got mode=%v screen=%v guided=%v cmd=%v", model.mode, model.screen, model.guidedWorkspace, cmd)
	}
	view := stripANSI(model.View())
	assertContains(t, view, "Advanced workspace")
	assertContains(t, view, "Navigation")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.mode != modeWizard {
		t.Fatalf("expected esc to return from advanced workspace, got mode=%v cmd=%v", model.mode, cmd)
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
	model := NewModel(plan, request, nil, func(actual application.GenerateRequest) (application.GenerateResult, error) {
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
	assertContains(t, view, "Advanced workspace")
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
	assertContains(t, view, "Advanced configuration")
	assertNotContains(t, view, "Navigation")

	model.enterWizardReview()
	view = stripANSI(model.View())
	assertContains(t, view, "Review your generation plan")
	assertContains(t, view, "Generate solution")
	assertContains(t, view, "Inspect advanced preview")
	assertContains(t, view, "Back to fields")
	assertNotContains(t, view, "Navigation")
}

func TestRunUsesFullscreenProgramOption(t *testing.T) {
	original := runTeaProgram
	t.Cleanup(func() { runTeaProgram = original })
	optionCount := 0
	runTeaProgram = func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
		optionCount = len(options)
		return model, nil
	}

	if err := Run(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if optionCount != 1 {
		t.Fatalf("expected one fullscreen program option, got %d", optionCount)
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
	if cmd != nil || !model.helpOpen || !strings.Contains(stripANSI(model.View()), "Global:") {
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
		{name: "wide rail", width: 120, want: "Enter open", unwanted: "Navigation ["},
		{name: "medium top navigation", width: 90, want: "Navigation [1:Overview]", unwanted: "Enter open"},
		{name: "narrow focused content", width: 60, want: "Overview", unwanted: "Enter open"},
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
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width, Height: 24})
			model = updated.(Model)
			if cmd != nil {
				t.Fatal("expected no command from window size")
			}
			view := stripANSI(model.View())
			assertContains(t, view, "Services workspace")
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
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width, Height: 24})
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
			updated, cmd := model.Update(tea.WindowSizeMsg{Width: width, Height: 24})
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
	assertContains(t, view, "Microgen - GENERATING")
	assertContains(t, view, "Primary: Generating files")
	assertContains(t, view, "Generating files. Please wait; exit is available after generation finishes.")
	assertNotContains(t, view, readyHelp)
	assertNotContains(t, view, "Exit: q/esc/ctrl+c")
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
	assertContains(t, view, "Microgen - REFRESHING")
	assertContains(t, view, "Primary: Refreshing plan")
	assertContains(t, view, "Refreshing plan. Please wait; editing, filtering, and generation are paused.")
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
	assertContains(t, view, "Files 1-2 of 2 (filter: all)")
	assertContains(t, view, "> [2/2] [CREATE] file-02.txt")
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
	assertContains(t, view, "Microgen - FAILED")
	assertContains(t, view, "Primary: r Retry refresh")
	assertContains(t, view, "Solution CatalogPlatform")
	assertContains(t, view, "Description New description")
	assertContains(t, view, "Target net9.0")
	assertContains(t, view, "Settings saved, but the plan refresh failed. Press r to retry the refresh.")
	assertContains(t, view, "FAILED Refresh after save failed: plan failed")
	assertContains(t, view, "Readiness is stale. Saved settings need a successful plan refresh before generation.")
	assertContains(t, view, "r Retry plan refresh. Other actions stay locked until refresh succeeds.")
	assertContains(t, view, "Locked: r retry refresh | q/esc/ctrl+c quit")
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
	assertContains(t, view, "Microgen - GENERATED")
	assertContains(t, view, "Primary: r Refresh")
	assertContains(t, view, "Generated 3 files written to /tmp/generated.")
	assertContains(t, view, "Next cd /tmp/generated && dotnet build")
	assertContains(t, view, "WARNING existing warning")
	assertContains(t, view, generatedHelp)
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

func TestModelViewWorkflowScreensShowResponsiveContent(t *testing.T) {
	plan := plannedFilesPlan(3)
	plan.OutputDir = "/tmp/generated"
	plan.Files[1].Action = "replace"
	plan.Files[2].Action = "unchanged"
	plan.DeletedFiles = []string{"old.cs"}
	plan.ExtraFileCount = 1
	model := workspaceModel(plan, application.GenerateRequest{OutputDir: plan.OutputDir}, nil, nil, nil)

	model.openScreen(screenPreview)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
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

func plannedFilesPlan(count int) application.GenerationPlan {
	files := make([]application.PlannedFile, count)
	for index := range files {
		files[index] = application.PlannedFile{Path: fmt.Sprintf("file-%02d.txt", index+1), Action: "create"}
	}
	return application.GenerationPlan{FileCount: count, Files: files}
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

func assertNotContains(t *testing.T, value, unexpected string) {
	t.Helper()
	value = stripANSI(value)
	if strings.Contains(value, unexpected) {
		t.Fatalf("expected %q not to contain %q", value, unexpected)
	}
}
