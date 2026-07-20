package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

const plannedFilePreviewLimit = 5

type GenerateFunc func(application.GenerateRequest) (application.GenerateResult, error)

type modelStatus int

const (
	statusReady modelStatus = iota
	statusGenerating
	statusGenerated
	statusFailed
)

type Model struct {
	plan     application.GenerationPlan
	request  application.GenerateRequest
	generate GenerateFunc
	status   modelStatus
	result   application.GenerateResult
	err      error
}

func NewModel(plan application.GenerationPlan, request application.GenerateRequest, generate GenerateFunc) Model {
	return Model{plan: plan, request: request, generate: generate, status: statusReady}
}

func Run(plan application.GenerationPlan, request application.GenerateRequest, generate GenerateFunc) error {
	_, err := tea.NewProgram(NewModel(plan, request, generate)).Run()
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
			if m.status == statusGenerating {
				return m, nil
			}
			return m, tea.Quit
		case "g":
			if m.status != statusReady && m.status != statusFailed {
				return m, nil
			}
			m.status = statusGenerating
			return m, m.generateCmd()
		}
	case generationFinishedMsg:
		if msg.err != nil {
			m.status = statusFailed
			m.err = msg.err
			return m, nil
		}
		m.status = statusGenerated
		m.result = msg.result
		m.plan = msg.result.Plan
		m.err = nil
		return m, nil
	}
	return m, nil
}

type generationFinishedMsg struct {
	result application.GenerateResult
	err    error
}

func (m Model) generateCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := m.generate(m.request)
		return generationFinishedMsg{result: result, err: err}
	}
}

func (m Model) View() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "Microgen")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "Product: %s\n", m.plan.Config.SolutionName)
	if m.plan.Config.SolutionDescription != "" {
		fmt.Fprintf(&builder, "Description: %s\n", m.plan.Config.SolutionDescription)
	}
	fmt.Fprintf(&builder, "Target framework: %s\n", m.plan.Config.TargetFramework)
	fmt.Fprintf(&builder, "Services: %d, entities: %d, value objects: %d\n", m.plan.Config.ServiceCount, m.plan.Config.EntityCount, m.plan.Config.ValueObjectCount)
	if len(m.plan.Config.ServiceNames) > 0 {
		fmt.Fprintf(&builder, "Service names: %s\n", strings.Join(m.plan.Config.ServiceNames, ", "))
	}
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
	switch m.status {
	case statusReady:
		fmt.Fprintln(&builder, "Press g to generate files. This writes files to the output directory.")
	case statusGenerating:
		fmt.Fprintln(&builder, "Generating files...")
		fmt.Fprintln(&builder, "Generation is in progress. Exit will be available after it finishes.")
	case statusGenerated:
		fmt.Fprintf(&builder, "Generated %d files in %s.\n", m.result.Plan.FileCount, m.result.OutputDir)
		if m.result.Warning != "" {
			fmt.Fprintf(&builder, "Warning: %s\n", m.result.Warning)
		}
	case statusFailed:
		fmt.Fprintf(&builder, "Generation failed: %v\n", m.err)
		fmt.Fprintln(&builder, "Press g to retry generation. This writes files to the output directory.")
	}
	fmt.Fprintln(&builder)
	if m.status != statusGenerating {
		fmt.Fprintln(&builder, "Press q, esc, or ctrl+c to quit.")
	}
	return builder.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
