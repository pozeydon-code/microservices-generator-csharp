package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

const (
	defaultFileWindowRows = 5
	minFileWindowRows     = 3
	maxFileWindowRows     = 12
	reservedViewRows      = 18
)

const readyHelp = "Navigate files: up/down, k/j, pgup/pgdown, home/end. Press r to refresh the plan, g to generate."
const generatedHelp = "Generation is complete. Navigate files: up/down, k/j, pgup/pgdown, home/end. Press r to refresh the plan."

type PlanFunc func(application.GenerateRequest) (application.GenerationPlan, error)
type GenerateFunc func(application.GenerateRequest) (application.GenerateResult, error)
type UpdateSettingsFunc func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error)

type modelStatus int

const (
	statusReady modelStatus = iota
	statusRefreshing
	statusGenerating
	statusGenerated
	statusFailed
	statusEditing
	statusSaving
)

type editField int

const (
	editFieldName editField = iota
	editFieldDescription
	editFieldTargetFramework
	editFieldCount
)

type textField struct {
	value  []rune
	cursor int
}

type editState struct {
	name            textField
	description     textField
	targetFramework textField
	focused         editField
	returnStatus    modelStatus
}

type Model struct {
	plan                       application.GenerationPlan
	request                    application.GenerateRequest
	planFunc                   PlanFunc
	generate                   GenerateFunc
	update                     UpdateSettingsFunc
	status                     modelStatus
	result                     application.GenerateResult
	err                        error
	errContext                 string
	message                    string
	edit                       editState
	targetFrameworkSuggestions []string
	fileCursor                 int
	fileOffset                 int
	windowRows                 int
}

func NewModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	suggestions := []string(nil)
	if len(targetFrameworkSuggestions) > 0 {
		suggestions = append([]string(nil), targetFrameworkSuggestions[0]...)
	}
	return Model{plan: plan, request: request, planFunc: planFunc, generate: generate, update: update, status: statusReady, targetFrameworkSuggestions: suggestions, windowRows: defaultFileWindowRows}
}

func Run(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions []string) error {
	_, err := tea.NewProgram(NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions)).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowRows = visibleFileRows(msg.Height)
		m.clampFileCursor()
		return m, nil
	case tea.KeyMsg:
		if m.status == statusEditing {
			return m.updateEdit(msg)
		}
		key := msg.String()
		if m.postSaveRefreshFailed() {
			switch key {
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			case "r":
				m.status = statusRefreshing
				m.err = nil
				m.errContext = ""
				m.message = ""
				return m, m.planCmd()
			default:
				return m, nil
			}
		}
		switch key {
		case "q", "esc", "ctrl+c":
			if m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.moveFileCursor(-1)
			return m, nil
		case "down", "j":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.moveFileCursor(1)
			return m, nil
		case "home":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.fileCursor = 0
			m.fileOffset = 0
			return m, nil
		case "end":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.fileCursor = len(m.plan.Files) - 1
			m.clampFileCursor()
			return m, nil
		case "pgup":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.moveFileCursor(-m.visibleFileRows())
			return m, nil
		case "pgdown":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.moveFileCursor(m.visibleFileRows())
			return m, nil
		case "r":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.status = statusRefreshing
			m.err = nil
			m.errContext = ""
			m.message = ""
			return m, m.planCmd()
		case "g":
			if m.status == statusFailed && m.errContext == "Refresh after save" {
				return m, nil
			}
			if m.status == statusRefreshing || m.status == statusSaving || (m.status != statusReady && m.status != statusFailed) {
				return m, nil
			}
			m.status = statusGenerating
			m.err = nil
			m.errContext = ""
			m.message = ""
			return m, m.generateCmd()
		case "e":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			m.startEditing()
			return m, nil
		}
	case planFinishedMsg:
		if msg.err != nil {
			m.status = statusFailed
			m.err = msg.err
			m.errContext = "Refresh"
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = ""
		m.clampFileCursor()
		return m, nil

	case generationFinishedMsg:
		if msg.err != nil {
			m.status = statusFailed
			m.err = msg.err
			m.errContext = "Generation"
			return m, nil
		}
		m.status = statusGenerated
		m.result = msg.result
		m.plan = msg.result.Plan
		m.err = nil
		m.errContext = ""
		m.message = ""
		m.clampFileCursor()
		return m, nil

	case settingsFinishedMsg:
		if msg.err != nil {
			m.status = statusEditing
			m.err = msg.err
			m.errContext = "Save"
			m.message = ""
			return m, nil
		}
		if msg.result.Saved && msg.result.PlanError != nil {
			m.plan.Config = msg.result.Config
			m.status = statusFailed
			m.err = msg.result.PlanError
			m.errContext = "Refresh after save"
			m.message = "Settings saved, but the plan refresh failed. Press r to retry the refresh."
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.result.Plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = "Settings saved. Plan refreshed. Service, entity, field, and value-object editing is not available yet."
		m.clampFileCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) postSaveRefreshFailed() bool {
	return m.status == statusFailed && m.errContext == "Refresh after save"
}

func (m *Model) startEditing() {
	returnStatus := m.status
	m.status = statusEditing
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.edit = editState{
		name:            newTextField(m.plan.Config.SolutionName),
		description:     newTextField(m.plan.Config.SolutionDescription),
		targetFramework: newTextField(m.plan.Config.TargetFramework),
		returnStatus:    returnStatus,
	}
	if m.edit.targetFramework.string() == "" {
		m.edit.targetFramework = newTextField("net8.0")
	}
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.status = m.edit.returnStatus
		m.err = nil
		m.errContext = ""
		return m, nil
	case "enter":
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveSettingsCmd()
	case "tab", "down":
		m.edit.focused = (m.edit.focused + 1) % editFieldCount
		return m, nil
	case "shift+tab", "up":
		m.edit.focused = (m.edit.focused + editFieldCount - 1) % editFieldCount
		return m, nil
	case " ":
		if m.edit.focused == editFieldTargetFramework {
			return m, nil
		}
	case "left":
		m.focusedTextField().move(-1)
		return m, nil
	case "right":
		m.focusedTextField().move(1)
		return m, nil
	case "ctrl+n":
		if m.edit.focused == editFieldTargetFramework {
			m.cycleTargetFrameworkSuggestion()
		}
		return m, nil
	case "backspace":
		m.focusedTextField().backspace()
		return m, nil
	case "delete":
		m.focusedTextField().delete()
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		m.focusedTextField().insert(msg.Runes)
		return m, nil
	}
	return m, nil
}

func newTextField(value string) textField {
	runes := []rune(value)
	return textField{value: runes, cursor: len(runes)}
}

func (field textField) string() string {
	return string(field.value)
}

func (field *textField) insert(runes []rune) {
	if len(runes) == 0 {
		return
	}
	field.value = append(field.value[:field.cursor], append(runes, field.value[field.cursor:]...)...)
	field.cursor += len(runes)
}

func (field *textField) backspace() {
	if field.cursor == 0 {
		return
	}
	field.value = append(field.value[:field.cursor-1], field.value[field.cursor:]...)
	field.cursor--
}

func (field *textField) delete() {
	if field.cursor >= len(field.value) {
		return
	}
	field.value = append(field.value[:field.cursor], field.value[field.cursor+1:]...)
}

func (field *textField) move(delta int) {
	field.cursor += delta
	if field.cursor < 0 {
		field.cursor = 0
	}
	if field.cursor > len(field.value) {
		field.cursor = len(field.value)
	}
}

func (m *Model) focusedTextField() *textField {
	switch m.edit.focused {
	case editFieldDescription:
		return &m.edit.description
	case editFieldTargetFramework:
		return &m.edit.targetFramework
	default:
		return &m.edit.name
	}
}

func (m *Model) cycleTargetFrameworkSuggestion() {
	if len(m.targetFrameworkSuggestions) == 0 {
		return
	}
	current := m.edit.targetFramework.string()
	for index, suggestion := range m.targetFrameworkSuggestions {
		if suggestion == current {
			m.edit.targetFramework = newTextField(m.targetFrameworkSuggestions[(index+1)%len(m.targetFrameworkSuggestions)])
			return
		}
	}
	m.edit.targetFramework = newTextField(m.targetFrameworkSuggestions[0])
}

func (m *Model) moveFileCursor(delta int) {
	m.fileCursor += delta
	m.clampFileCursor()
}

func (m *Model) clampFileCursor() {
	fileCount := len(m.plan.Files)
	if fileCount == 0 {
		m.fileCursor = 0
		m.fileOffset = 0
		return
	}
	if m.fileCursor < 0 {
		m.fileCursor = 0
	}
	if m.fileCursor >= fileCount {
		m.fileCursor = fileCount - 1
	}
	if m.fileOffset > m.fileCursor {
		m.fileOffset = m.fileCursor
	}
	rows := m.visibleFileRows()
	if m.fileCursor >= m.fileOffset+rows {
		m.fileOffset = m.fileCursor - rows + 1
	}
	lastOffset := fileCount - rows
	if lastOffset < 0 {
		lastOffset = 0
	}
	if m.fileOffset > lastOffset {
		m.fileOffset = lastOffset
	}
	if m.fileOffset < 0 {
		m.fileOffset = 0
	}
}

func (m Model) visibleFileRows() int {
	if m.windowRows == 0 {
		return defaultFileWindowRows
	}
	return m.windowRows
}

func visibleFileRows(height int) int {
	if height <= 0 {
		return defaultFileWindowRows
	}
	rows := height - reservedViewRows
	if rows < minFileWindowRows {
		return minFileWindowRows
	}
	if rows > maxFileWindowRows {
		return maxFileWindowRows
	}
	return rows
}

type planFinishedMsg struct {
	plan application.GenerationPlan
	err  error
}

type generationFinishedMsg struct {
	result application.GenerateResult
	err    error
}

type settingsFinishedMsg struct {
	result application.UpdateSolutionSettingsResult
	err    error
}

func (m Model) planCmd() tea.Cmd {
	return func() tea.Msg {
		plan, err := m.planFunc(m.request)
		return planFinishedMsg{plan: plan, err: err}
	}
}

func (m Model) generateCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := m.generate(m.request)
		return generationFinishedMsg{result: result, err: err}
	}
}

func (m Model) saveSettingsCmd() tea.Cmd {
	settings := application.SolutionSettings{
		SolutionName:        m.edit.name.string(),
		SolutionDescription: m.edit.description.string(),
		TargetFramework:     m.edit.targetFramework.string(),
	}
	return func() tea.Msg {
		result, err := m.update(m.request, settings)
		return settingsFinishedMsg{result: result, err: err}
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

	fileCount := len(m.plan.Files)
	if fileCount == 0 {
		fmt.Fprintln(&builder, "- No files planned")
	} else {
		start := m.fileOffset
		end := start + m.visibleFileRows()
		if end > fileCount {
			end = fileCount
		}
		fmt.Fprintf(&builder, "Files %d-%d of %d\n", start+1, end, fileCount)
		for index, file := range m.plan.Files[start:end] {
			cursor := " "
			if start+index == m.fileCursor {
				cursor = ">"
			}
			fmt.Fprintf(&builder, "%s %s %s\n", cursor, file.Action, file.Path)
		}
	}

	fmt.Fprintln(&builder)
	if m.message != "" {
		fmt.Fprintln(&builder, m.message)
		fmt.Fprintln(&builder)
	}
	if m.status == statusEditing || m.status == statusSaving {
		m.renderSettingsEditor(&builder)
		return builder.String()
	}
	switch m.status {
	case statusReady:
		fmt.Fprintln(&builder, "Press r to refresh the plan or g to generate files. Generation writes files to the output directory.")
	case statusRefreshing:
		fmt.Fprintln(&builder, "Refreshing plan...")
		fmt.Fprintln(&builder, "Please wait while the read-only plan refresh finishes. Press q, esc, or ctrl+c to quit.")
	case statusGenerating:
		fmt.Fprintln(&builder, "Generating files...")
		fmt.Fprintln(&builder, "Generation is in progress. Exit will be available after it finishes.")
	case statusGenerated:
		fmt.Fprintf(&builder, "Generated %d files in %s.\n", m.result.Plan.FileCount, m.result.OutputDir)
		if m.result.Warning != "" {
			fmt.Fprintf(&builder, "Warning: %s\n", m.result.Warning)
		}
	case statusSaving:
		fmt.Fprintln(&builder, "Saving settings...")
		fmt.Fprintln(&builder, "Save is in progress. Exit will be available after it finishes.")
	case statusFailed:
		context := m.errContext
		if context == "" {
			context = "Generation"
		}
		fmt.Fprintf(&builder, "%s failed: %v\n", context, m.err)
		if m.errContext == "Refresh after save" {
			fmt.Fprintln(&builder, "Only refresh retry is available until the plan refresh succeeds.")
		} else {
			fmt.Fprintln(&builder, "Press r to refresh the plan or g to retry generation.")
		}
	}
	fmt.Fprintln(&builder)
	if m.status != statusGenerating && m.status != statusRefreshing {
		if m.postSaveRefreshFailed() {
			fmt.Fprintln(&builder, "Press r to retry the plan refresh.")
			fmt.Fprintln(&builder, "Press q, esc, or ctrl+c to quit.")
			return builder.String()
		}
		if m.status == statusGenerated {
			fmt.Fprintln(&builder, generatedHelp)
		} else {
			fmt.Fprintln(&builder, readyHelp)
		}
		fmt.Fprintln(&builder, "Press e to edit solution settings. Service, entity, field, and value-object editing is not available yet.")
		fmt.Fprintln(&builder, "Press q, esc, or ctrl+c to quit.")
	}
	return builder.String()
}

func (m Model) renderSettingsEditor(builder *strings.Builder) {
	if m.status == statusSaving {
		fmt.Fprintln(builder, "Saving settings...")
		fmt.Fprintln(builder, "Save is in progress. Exit will be available after it finishes.")
		return
	}
	fmt.Fprintln(builder, "Editing solution settings")
	fmt.Fprintln(builder, "Service, entity, field, and value-object editing is not available yet.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	fmt.Fprintf(builder, "%s Solution name: %s\n", editCursor(m.edit.focused == editFieldName), m.edit.name.string())
	fmt.Fprintf(builder, "%s Description: %s\n", editCursor(m.edit.focused == editFieldDescription), m.edit.description.string())
	fmt.Fprintf(builder, "%s Target framework: %s\n", editCursor(m.edit.focused == editFieldTargetFramework), m.edit.targetFramework.string())
	if len(m.targetFrameworkSuggestions) > 0 {
		fmt.Fprintf(builder, "  Suggestions: %s\n", strings.Join(m.targetFrameworkSuggestions, ", "))
	}
	fmt.Fprintln(builder)
	fmt.Fprintln(builder, "Type a major or TFM such as 10 or net10.0. Ctrl+n cycles suggestions. Enter saves. Esc cancels.")
	fmt.Fprintln(builder, "Tab/down and shift+tab/up move fields.")
}

func editCursor(focused bool) string {
	if focused {
		return ">"
	}
	return " "
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
