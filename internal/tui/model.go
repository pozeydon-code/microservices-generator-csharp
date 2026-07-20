package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

const plannedFilePreviewLimit = 5

type Model struct {
	plan application.GenerationPlan
}

func NewModel(plan application.GenerationPlan) Model {
	return Model{plan: plan}
}

func Run(plan application.GenerationPlan) error {
	_, err := tea.NewProgram(NewModel(plan)).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "Microgen")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "Output directory: %s\n", m.plan.OutputDir)
	fmt.Fprintf(&builder, "Output action: %s\n", m.plan.OutputAction)
	fmt.Fprintf(&builder, "Force: required=%s, used=%s\n", yesNo(m.plan.ForceRequired), yesNo(m.plan.ForceUsed))
	fmt.Fprintf(&builder, "Files planned: %d\n", m.plan.FileCount)
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "Planned files:")

	limit := len(m.plan.Files)
	if limit > plannedFilePreviewLimit {
		limit = plannedFilePreviewLimit
	}
	for _, file := range m.plan.Files[:limit] {
		fmt.Fprintf(&builder, "- %s %s\n", file.Action, file.Path)
	}
	if len(m.plan.Files) > limit {
		fmt.Fprintf(&builder, "Showing first %d of %d planned files.\n", limit, len(m.plan.Files))
	}
	if len(m.plan.Files) == 0 {
		fmt.Fprintln(&builder, "- No files planned")
	}

	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "Press q, esc, or ctrl+c to quit.")
	return builder.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
