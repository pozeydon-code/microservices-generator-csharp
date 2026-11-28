package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
)

func previewScreenText(screen tcell.SimulationScreen) string {
	cells, width, _ := screen.GetContents()
	var out strings.Builder
	for i, cell := range cells {
		if i > 0 && i%width == 0 {
			out.WriteByte('\n')
		}
		if len(cell.Runes) == 0 {
			out.WriteByte(' ')
		} else {
			out.WriteRune(cell.Runes[0])
		}
	}
	return out.String()
}

func TestTViewPreviewShowsRefreshError(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	const reason = "config.json: invalid JSON after external edit"
	plan := application.GenerationPlan{Config: application.ConfigSummary{SolutionName: "Commerce"}}
	ui := newTViewUI(plan, application.GenerateRequest{}, func(application.GenerateRequest) (application.GenerationPlan, error) {
		return application.GenerationPlan{}, errors.New(reason)
	}, nil)
	send := runTViewTest(t, ui, 140, 40, screen)
	for i := 0; i < int(tviewScreenPreview); i++ {
		send(tviewKey(tcell.KeyDown))
	}
	send(tviewRune('r'))
	if got := previewScreenText(screen); !strings.Contains(got, reason) {
		t.Errorf("Preview does not show the refresh error: %q\n%s", reason, got)
	}
}
