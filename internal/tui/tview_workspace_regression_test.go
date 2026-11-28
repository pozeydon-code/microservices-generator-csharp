package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
)

func TestTViewResultShowsCleanupWarning(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	plan := application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce"}, FileCount: 150}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)
	send := runTViewTest(t, ui, 140, 40, screen)
	const action = "inspect and remove that lock manually"
	ui.app.QueueUpdateDraw(func() {
		ui.finishGeneration(application.GenerateResult{
			Plan:      plan,
			OutputDir: "/home/developer/projects/commerce/generated",
			Warning:   "generated output was published, but cleanup of publication lock failed at /home/developer/projects/commerce/generated.lock: permission denied; " + action,
		}, nil)
	})
	send()
	lines := strings.Split(previewScreenText(screen), "\n")
	body := strings.Join(lines[:38], "\n") // Exclude the two fixed footer rows.
	for _, want := range []string{"Result", "Generated 150 files in /home/developer/projects/commerce/generated.", action} {
		if !strings.Contains(body, want) {
			t.Errorf("Result content missing %q", want)
		}
	}
}

func TestTViewOverviewShowsStarterReadinessHint(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	const hint = "Rename the starter entity and add domain fields."
	plan := application.GenerationPlan{
		Config: application.ConfigSummary{
			SolutionName: "Starter",
			Services:     []application.ServiceSummary{{Name: "Catalog", Entities: []application.EntitySummary{{Name: "Item"}}}},
		},
		Readiness: application.ReadinessSummary{Hints: []string{hint}},
		Files:     []application.PlannedFile{{Path: "preview-only.cs"}},
	}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)
	send := runTViewTest(t, ui, 140, 40, screen)
	send()
	got := previewScreenText(screen)
	for _, want := range []string{"Catalog / Item", hint} {
		if !strings.Contains(got, want) {
			t.Errorf("Overview missing %q", want)
		}
	}
	for _, unwanted := range []string{"preview-only.cs", "Readiness: services="} {
		if strings.Contains(got, unwanted) {
			t.Errorf("Overview duplicates global plan metadata: %q", unwanted)
		}
	}
}
