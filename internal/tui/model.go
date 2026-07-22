package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pozeydon-code/generator-microservices-go/internal/application"
)

const (
	defaultFileWindowRows = 5
	minFileWindowRows     = 3
	maxFileWindowRows     = 12
	reservedViewRows      = 18
)

const readyHelp = "Navigate files: arrows/k/j/pgup/pgdown/home/end. Actions: g generate, e edit settings, r refresh, a filter."
const generatedHelp = "Navigate files: arrows/k/j/pgup/pgdown/home/end. Actions: r refresh, a filter."

var (
	border = lipgloss.Border{
		Top:         "-",
		Bottom:      "-",
		Left:        "|",
		Right:       "|",
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
	}

	appTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	cardStyle         = lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	sectionTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	labelStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	dimStyle          = lipgloss.NewStyle().Faint(true)
	successStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	warningStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	dangerStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	busyStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	readyStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45"))
	badgeStyle        = lipgloss.NewStyle().Bold(true)
	selectedRowStyle  = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("231"))
)

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
	actionFilter               string
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
			indices := m.filteredFileIndices()
			if len(indices) > 0 {
				m.fileCursor = indices[0]
			}
			m.fileOffset = 0
			return m, nil
		case "end":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			indices := m.filteredFileIndices()
			if len(indices) > 0 {
				m.fileCursor = indices[len(indices)-1]
			}
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
		case "a":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			m.cycleActionFilter()
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
	indices := m.filteredFileIndices()
	if len(indices) == 0 {
		m.clampFileCursor()
		return
	}
	position := m.selectedFilteredPosition(indices)
	if position < 0 {
		position = closestFilteredPosition(indices, m.fileCursor)
	}
	position += delta
	if position < 0 {
		position = 0
	}
	if position >= len(indices) {
		position = len(indices) - 1
	}
	m.fileCursor = indices[position]
	m.clampFileCursor()
}

func (m *Model) clampFileCursor() {
	indices := m.filteredFileIndices()
	if len(indices) == 0 && m.actionFilter != "" {
		m.actionFilter = ""
		indices = m.filteredFileIndices()
	}
	fileCount := len(indices)
	if fileCount == 0 {
		m.fileCursor = 0
		m.fileOffset = 0
		return
	}
	position := m.selectedFilteredPosition(indices)
	if position < 0 {
		position = closestFilteredPosition(indices, m.fileCursor)
		m.fileCursor = indices[position]
	}
	if position >= fileCount {
		position = fileCount - 1
		m.fileCursor = indices[position]
	}
	if m.fileOffset > position {
		m.fileOffset = position
	}
	rows := m.visibleFileRows()
	if position >= m.fileOffset+rows {
		m.fileOffset = position - rows + 1
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

func (m *Model) cycleActionFilter() {
	actions := m.plannedFileActions()
	if len(actions) == 0 {
		m.actionFilter = ""
		m.clampFileCursor()
		return
	}
	if m.actionFilter == "" {
		m.actionFilter = actions[0]
		m.selectFirstFilteredFile()
		m.clampFileCursor()
		return
	}
	for index, action := range actions {
		if action == m.actionFilter {
			if index == len(actions)-1 {
				m.actionFilter = ""
			} else {
				m.actionFilter = actions[index+1]
			}
			m.selectFirstFilteredFile()
			m.clampFileCursor()
			return
		}
	}
	m.actionFilter = ""
	m.selectFirstFilteredFile()
	m.clampFileCursor()
}

func (m *Model) selectFirstFilteredFile() {
	indices := m.filteredFileIndices()
	if len(indices) == 0 {
		m.fileCursor = 0
		m.fileOffset = 0
		return
	}
	m.fileCursor = indices[0]
	m.fileOffset = 0
}

func (m Model) plannedFileActions() []string {
	seen := make(map[string]bool)
	for _, file := range m.plan.Files {
		if file.Action == "" || seen[file.Action] {
			continue
		}
		seen[file.Action] = true
	}
	actions := make([]string, 0, len(seen))
	for action := range seen {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

func (m Model) filteredFileIndices() []int {
	indices := make([]int, 0, len(m.plan.Files))
	for index, file := range m.plan.Files {
		if m.actionFilter == "" || file.Action == m.actionFilter {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) selectedFilteredPosition(indices []int) int {
	for position, index := range indices {
		if index == m.fileCursor {
			return position
		}
	}
	return -1
}

func closestFilteredPosition(indices []int, cursor int) int {
	position := 0
	for index, fileIndex := range indices {
		if fileIndex <= cursor {
			position = index
			continue
		}
		break
	}
	return position
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
	fmt.Fprintf(&builder, "%s - %s\n", appTitleStyle.Render("Microgen"), m.statusBadge())
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Primary:"), m.primaryActionStyle().Render(m.primaryAction()))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.configCard())
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.outputPreviewCard())
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.plannedFilesCard())
	fmt.Fprintln(&builder)
	if m.message != "" {
		fmt.Fprintln(&builder, successStyle.Render(m.message))
		fmt.Fprintln(&builder)
	}
	fmt.Fprintln(&builder, m.actionsCard())
	return builder.String()
}

func (m Model) configCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Config"))
	fmt.Fprintf(&builder, "%s %s %s\n", labelStyle.Render("Source"), m.request.ConfigPath, dimStyle.Render("("+m.configSourceLabel()+")"))
	if m.request.ConfigBootstrapped {
		fmt.Fprintln(&builder, dimStyle.Render("Created starter config. Edit settings incrementally; service/entity/field editing comes later."))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Product"), m.plan.Config.SolutionName)
	if m.plan.Config.SolutionDescription != "" {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Description"), m.plan.Config.SolutionDescription)
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Target"), m.plan.Config.TargetFramework)
	if m.plan.Config.SolutionFormat != "" {
		fmt.Fprintf(&builder, "%s .%s\n", labelStyle.Render("Format"), m.plan.Config.SolutionFormat)
	}
	fmt.Fprintf(&builder, "%s %d services, %d entities, %d value objects\n", labelStyle.Render("Contents"), m.plan.Config.ServiceCount, m.plan.Config.EntityCount, m.plan.Config.ValueObjectCount)
	if len(m.plan.Config.ServiceNames) > 0 {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Services"), strings.Join(m.plan.Config.ServiceNames, ", "))
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) outputPreviewCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Output Preview"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Directory"), m.plan.OutputDir)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Write mode"), m.plan.OutputAction)
	forceStyle := dimStyle
	if m.plan.ForceRequired && !m.plan.ForceUsed {
		forceStyle = dangerStyle
	} else if m.plan.ForceRequired {
		forceStyle = warningStyle
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Force"), forceStyle.Render(fmt.Sprintf("required=%s, used=%s", yesNo(m.plan.ForceRequired), yesNo(m.plan.ForceUsed))))
	fmt.Fprintf(&builder, "%s %d planned\n", labelStyle.Render("Files"), m.plan.FileCount)
	impact := m.impactSummary()
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Impact"), impact)
	if m.plan.FileCount > 0 && impact == "unchanged only" {
		fmt.Fprintln(&builder, successStyle.Render("No generated file content changes detected."))
	}
	if m.plan.ExtraFileCount > 0 {
		fmt.Fprintf(&builder, "%s replacement removes %d previous generated file(s)\n", dangerStyle.Render("DANGER"), m.plan.ExtraFileCount)
		fmt.Fprintf(&builder, "%s\n", dangerStyle.Render(deletedFilePreview(m.plan.DeletedFiles)))
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) plannedFilesCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Planned Files"))

	indices := m.filteredFileIndices()
	fileCount := len(indices)
	if fileCount == 0 {
		if m.actionFilter != "" {
			fmt.Fprintf(&builder, "%s\n", warningStyle.Render(fmt.Sprintf("No files match filter %q. Press a to reset the filter.", m.actionFilter)))
		} else {
			fmt.Fprintln(&builder, dimStyle.Render("No files planned."))
		}
	} else {
		start := m.fileOffset
		end := start + m.visibleFileRows()
		if end > fileCount {
			end = fileCount
		}
		filter := "all"
		if m.actionFilter != "" {
			filter = m.actionFilter
		}
		fmt.Fprintf(&builder, "%s %d-%d of %d %s\n", labelStyle.Render("Files"), start+1, end, fileCount, dimStyle.Render("(filter: "+filter+")"))
		if m.actionFilter != "" {
			fmt.Fprintf(&builder, "%s Press a to cycle filters back to all.\n", labelStyle.Render("Filter"))
		}
		selectedPosition := m.selectedFilteredPosition(indices)
		if selectedPosition >= 0 {
			selectedFile := m.plan.Files[indices[selectedPosition]]
			fmt.Fprintf(&builder, "%s %d/%d %s %s\n", labelStyle.Render("Selected:"), selectedPosition+1, fileCount, actionBadge(selectedFile.Action), selectedFile.Path)
		}
		for position, planIndex := range indices[start:end] {
			file := m.plan.Files[planIndex]
			row := fmt.Sprintf("  [%d/%d] %s %s", start+position+1, fileCount, actionBadge(file.Action), file.Path)
			if start+position == selectedPosition {
				row = selectedRowStyle.Render(fmt.Sprintf("> [%d/%d] %s %s", start+position+1, fileCount, actionBadge(file.Action), file.Path))
			}
			fmt.Fprintln(&builder, row)
		}
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) actionsCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Actions"))
	if m.status == statusEditing || m.status == statusSaving {
		m.renderSettingsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	switch m.status {
	case statusReady:
		fmt.Fprintf(&builder, "%s Generate files into the output directory.\n", successStyle.Render("g"))
		fmt.Fprintln(&builder, "e Edit solution settings. Service, entity, field, and value-object editing is not available yet.")
	case statusRefreshing:
		fmt.Fprintln(&builder, busyStyle.Render("Refreshing plan. Please wait; editing, filtering, and generation are paused."))
	case statusGenerating:
		fmt.Fprintln(&builder, busyStyle.Render("Generating files. Please wait; exit is available after generation finishes."))
	case statusGenerated:
		fmt.Fprintln(&builder, successStyle.Render(fmt.Sprintf("Generated %d files in %s.", m.result.Plan.FileCount, m.result.OutputDir)))
		if m.result.Warning != "" {
			fmt.Fprintf(&builder, "%s %s\n", warningStyle.Render("WARNING"), warningStyle.Render(m.result.Warning))
		}
	case statusSaving:
		fmt.Fprintln(&builder, busyStyle.Render("Saving settings. Please wait; exit is available after save finishes."))
	case statusFailed:
		context := m.errContext
		if context == "" {
			context = "Generation"
		}
		fmt.Fprintf(&builder, "%s %s failed: %v\n", dangerStyle.Render("FAILED"), context, m.err)
		if m.errContext == "Refresh after save" {
			fmt.Fprintln(&builder, dangerStyle.Render("r Retry plan refresh. Other actions stay locked until refresh succeeds."))
		} else {
			fmt.Fprintln(&builder, dangerStyle.Render("g Retry generation, or r refresh the plan first."))
		}
	}
	fmt.Fprintln(&builder)
	if m.status != statusGenerating && m.status != statusRefreshing {
		if m.postSaveRefreshFailed() {
			fmt.Fprintln(&builder, "Keys: r retry refresh | q/esc/ctrl+c quit")
			return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
		}
		if m.status == statusGenerated {
			fmt.Fprintln(&builder, generatedHelp)
		} else {
			fmt.Fprintln(&builder, readyHelp)
		}
		fmt.Fprintln(&builder, "Config modes: existing JSON with --config <path>; starter JSON with --new --config <path>.")
		fmt.Fprintln(&builder, "Exit: q/esc/ctrl+c")
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) statusLabel() string {
	switch m.status {
	case statusReady:
		return "READY"
	case statusRefreshing:
		return "REFRESHING"
	case statusGenerating:
		return "GENERATING"
	case statusGenerated:
		return "GENERATED"
	case statusFailed:
		return "FAILED"
	case statusEditing:
		return "EDITING"
	case statusSaving:
		return "SAVING"
	default:
		return "READY"
	}
}

func (m Model) statusBadge() string {
	return badgeStyle.Foreground(m.statusColor()).Render(m.statusLabel())
}

func (m Model) statusColor() lipgloss.Color {
	switch m.status {
	case statusReady:
		return lipgloss.Color("45")
	case statusRefreshing, statusGenerating, statusSaving, statusEditing:
		return lipgloss.Color("39")
	case statusGenerated:
		return lipgloss.Color("42")
	case statusFailed:
		return lipgloss.Color("196")
	default:
		return lipgloss.Color("45")
	}
}

func (m Model) primaryActionStyle() lipgloss.Style {
	switch m.status {
	case statusReady, statusGenerated:
		return successStyle
	case statusRefreshing, statusGenerating, statusSaving, statusEditing:
		return busyStyle
	case statusFailed:
		return dangerStyle
	default:
		return readyStyle
	}
}

func (m Model) primaryAction() string {
	switch m.status {
	case statusReady:
		return "g Generate"
	case statusRefreshing:
		return "Refreshing plan"
	case statusGenerating:
		return "Generating files"
	case statusGenerated:
		return "r Refresh"
	case statusFailed:
		if m.postSaveRefreshFailed() {
			return "r Retry refresh"
		}
		return "g Retry generation"
	case statusEditing:
		return "enter Save settings"
	case statusSaving:
		return "Saving settings"
	default:
		return "g Generate"
	}
}

func actionBadge(action string) string {
	label := "[" + strings.ToUpper(action) + "]"
	switch action {
	case "create":
		return badgeStyle.Foreground(lipgloss.Color("42")).Render(label)
	case "replace":
		return badgeStyle.Foreground(lipgloss.Color("214")).Render(label)
	case "unchanged":
		return badgeStyle.Foreground(lipgloss.Color("245")).Render(label)
	default:
		return badgeStyle.Foreground(lipgloss.Color("39")).Render(label)
	}
}

func (m Model) configSourceLabel() string {
	if m.request.ConfigBootstrapped {
		return "starter config bootstrapped this run"
	}
	return "existing JSON"
}

func deletedFilePreview(files []string) string {
	if len(files) == 0 {
		return "none"
	}
	limit := len(files)
	if limit > 3 {
		limit = 3
	}
	preview := strings.Join(files[:limit], ", ")
	if len(files) > limit {
		preview += fmt.Sprintf(", and %d more", len(files)-limit)
	}
	return preview
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
		fmt.Fprintf(builder, "  Suggestions (newest first): %s\n", strings.Join(m.targetFrameworkSuggestions, ", "))
	}
	fmt.Fprintln(builder)
	fmt.Fprintln(builder, "Type a major or TFM such as 6, 7, 10, or net10.0. Ctrl+n cycles suggestions. Enter saves. Esc cancels.")
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

func (m Model) impactSummary() string {
	if len(m.plan.Files) == 0 {
		return "none"
	}
	counts := make(map[string]int)
	for _, file := range m.plan.Files {
		counts[file.Action]++
	}
	actions := make([]string, 0, len(counts))
	for action := range counts {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		parts = append(parts, fmt.Sprintf("%s=%d", action, counts[action]))
	}
	if len(parts) > 1 {
		return strings.Join(parts, ", ") + " (mixed actions)"
	}
	if actions[0] == "unchanged" {
		return "unchanged only"
	}
	return parts[0]
}
