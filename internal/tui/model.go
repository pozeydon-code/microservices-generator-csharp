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
const stepHelp = "Steps: tab/] next, shift+tab/[ previous."

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
type UpdateServicesFunc func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error)
type UpdateEntitiesFunc func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error)
type UpdateFieldsFunc func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error)

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

type tuiStep int

const (
	stepSource tuiStep = iota
	stepProject
	stepServices
	stepPreview
	stepGenerate
	stepCount
)

type editField int

type editMode int

const (
	editFieldName editField = iota
	editFieldDescription
	editFieldTargetFramework
	editFieldCount
)

const (
	editModeProject editMode = iota
	editModeServices
	editModeEntities
	editModeFields
)

type textField struct {
	value  []rune
	cursor int
}

type editState struct {
	mode            editMode
	name            textField
	description     textField
	targetFramework textField
	focused         editField
	returnStatus    modelStatus
}

type servicesEditState struct {
	original     []string
	services     []textField
	selected     int
	renaming     bool
	returnStatus modelStatus
}

type entitiesEditState struct {
	serviceName  string
	original     []string
	entities     []textField
	selected     int
	renaming     bool
	returnStatus modelStatus
}

type fieldEditItem struct {
	originalName string
	name         textField
	typeName     textField
}

type fieldsEditState struct {
	serviceName string
	entityName  string
	fields      []fieldEditItem
	selected    int
	editingName bool
	editingType bool
}

type Model struct {
	plan                       application.GenerationPlan
	request                    application.GenerateRequest
	planFunc                   PlanFunc
	generate                   GenerateFunc
	update                     UpdateSettingsFunc
	updateServices             UpdateServicesFunc
	updateEntities             UpdateEntitiesFunc
	updateFields               UpdateFieldsFunc
	status                     modelStatus
	result                     application.GenerateResult
	err                        error
	errContext                 string
	message                    string
	edit                       editState
	servicesEdit               servicesEditState
	entitiesEdit               entitiesEditState
	fieldsEdit                 fieldsEditState
	targetFrameworkSuggestions []string
	fileCursor                 int
	fileOffset                 int
	windowRows                 int
	actionFilter               string
	currentStep                tuiStep
	selectedService            int
}

func NewModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	suggestions := []string(nil)
	if len(targetFrameworkSuggestions) > 0 {
		suggestions = append([]string(nil), targetFrameworkSuggestions[0]...)
	}
	return Model{plan: plan, request: request, planFunc: planFunc, generate: generate, update: update, status: statusReady, targetFrameworkSuggestions: suggestions, windowRows: defaultFileWindowRows, currentStep: stepSource}
}

func Run(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, updateServices UpdateServicesFunc, updateEntities UpdateEntitiesFunc, updateFields UpdateFieldsFunc, targetFrameworkSuggestions []string) error {
	model := NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions)
	model.updateServices = updateServices
	model.updateEntities = updateEntities
	model.updateFields = updateFields
	_, err := tea.NewProgram(model).Run()
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
			if m.edit.mode == editModeFields {
				return m.updateFieldsEdit(msg)
			}
			if m.edit.mode == editModeEntities {
				return m.updateEntitiesEdit(msg)
			}
			if m.edit.mode == editModeServices {
				return m.updateServicesEdit(msg)
			}
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
				m.message = ""
				return m, m.planCmd()
			default:
				return m, nil
			}
		}
		switch key {
		case "tab", "]":
			if m.busy() {
				return m, nil
			}
			m.moveStep(1)
			return m, nil
		case "shift+tab", "[":
			if m.busy() {
				return m, nil
			}
			m.moveStep(-1)
			return m, nil
		case "q", "esc", "ctrl+c":
			if m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			if m.currentStep == stepServices {
				m.moveSelectedService(-1)
				return m, nil
			}
			m.moveFileCursor(-1)
			return m, nil
		case "down", "j":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			if m.currentStep == stepServices {
				m.moveSelectedService(1)
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
			m.currentStep = stepGenerate
			m.err = nil
			m.errContext = ""
			m.message = ""
			return m, m.generateCmd()
		case "e":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.currentStep == stepServices {
				m.startServicesEditing()
			} else {
				m.currentStep = stepProject
				m.startEditing()
			}
			return m, nil
		case "enter":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.currentStep == stepServices {
				m.startEntitiesEditing()
			}
			return m, nil
		}
	case planFinishedMsg:
		if msg.err != nil {
			errContext := "Refresh"
			if m.errContext == "Refresh after save" {
				errContext = m.errContext
			}
			m.status = statusFailed
			m.err = msg.err
			m.errContext = errContext
			m.currentStep = stepGenerate
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = ""
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil

	case generationFinishedMsg:
		if msg.err != nil {
			m.status = statusFailed
			m.err = msg.err
			m.errContext = "Generation"
			m.currentStep = stepGenerate
			return m, nil
		}
		m.status = statusGenerated
		m.result = msg.result
		m.plan = msg.result.Plan
		m.currentStep = stepGenerate
		m.err = nil
		m.errContext = ""
		m.message = ""
		m.clampSelectedService()
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
		m.message = "Settings saved. Plan refreshed. Use Services to edit services and entities; field and value-object editing is upcoming."
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil

	case servicesFinishedMsg:
		if msg.err != nil {
			m.status = statusEditing
			m.edit.mode = editModeServices
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
			m.message = "Services saved, but the plan refresh failed. Press r to retry the refresh."
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.result.Plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = "Services saved. Plan refreshed. Use enter on the Services step to edit entities and fields."
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil

	case entitiesFinishedMsg:
		if msg.err != nil {
			m.status = statusEditing
			m.edit.mode = editModeEntities
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
			m.message = "Entities saved, but the plan refresh failed. Press r to retry the refresh."
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.result.Plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = "Entities saved. Plan refreshed. Press f in the entity editor to edit fields."
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil

	case fieldsFinishedMsg:
		if msg.err != nil {
			m.status = statusEditing
			m.edit.mode = editModeFields
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
			m.message = "Fields saved, but the plan refresh failed. Press r to retry the refresh."
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.result.Plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = "Fields saved. Plan refreshed. Value-object editing is upcoming."
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) busy() bool {
	return m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving
}

func (m *Model) moveStep(delta int) {
	next := int(m.currentStep) + delta
	if next < 0 {
		next = 0
	}
	if next >= int(stepCount) {
		next = int(stepCount) - 1
	}
	m.currentStep = tuiStep(next)
}

func (step tuiStep) label() string {
	switch step {
	case stepSource:
		return "Source"
	case stepProject:
		return "Project"
	case stepServices:
		return "Services"
	case stepPreview:
		return "Preview"
	case stepGenerate:
		return "Generate"
	default:
		return "Source"
	}
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
		mode:            editModeProject,
		name:            newTextField(m.plan.Config.SolutionName),
		description:     newTextField(m.plan.Config.SolutionDescription),
		targetFramework: newTextField(m.plan.Config.TargetFramework),
		returnStatus:    returnStatus,
	}
	if m.edit.targetFramework.string() == "" {
		m.edit.targetFramework = newTextField("net8.0")
	}
}

func (m *Model) startServicesEditing() {
	returnStatus := m.status
	m.status = statusEditing
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.edit = editState{mode: editModeServices}
	m.servicesEdit = servicesEditState{returnStatus: returnStatus, original: append([]string(nil), m.plan.Config.ServiceNames...), services: make([]textField, 0, len(m.plan.Config.ServiceNames))}
	for _, name := range m.plan.Config.ServiceNames {
		m.servicesEdit.services = append(m.servicesEdit.services, newTextField(name))
	}
	if len(m.servicesEdit.services) == 0 {
		m.servicesEdit.services = append(m.servicesEdit.services, newTextField("CatalogService"))
	}
}

func (m *Model) startEntitiesEditing() {
	returnStatus := m.status
	m.status = statusEditing
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.edit = editState{mode: editModeEntities}
	service := m.selectedServiceSummary()
	m.entitiesEdit = entitiesEditState{returnStatus: returnStatus, serviceName: service.Name, original: append([]string(nil), service.EntityNames...), entities: make([]textField, 0, len(service.EntityNames))}
	for _, name := range service.EntityNames {
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(name))
	}
	if len(m.entitiesEdit.entities) == 0 {
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(m.nextEntityPlaceholder()))
		m.entitiesEdit.original = append(m.entitiesEdit.original, "")
	}
}

func (m *Model) startFieldsEditing() {
	entity := m.selectedEntitySummary()
	m.edit.mode = editModeFields
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.fieldsEdit = fieldsEditState{serviceName: m.entitiesEdit.serviceName, entityName: entity.Name, fields: make([]fieldEditItem, 0, len(entity.Fields))}
	for _, field := range entity.Fields {
		m.fieldsEdit.fields = append(m.fieldsEdit.fields, fieldEditItem{originalName: field.Name, name: newTextField(field.Name), typeName: newTextField(field.Type)})
	}
	if len(m.fieldsEdit.fields) == 0 {
		m.fieldsEdit.fields = append(m.fieldsEdit.fields, fieldEditItem{name: newTextField(m.nextFieldPlaceholder()), typeName: newTextField("string")})
	}
}

func (m Model) updateEntitiesEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && m.entitiesEdit.renaming {
		m.selectedEntityField().insert(msg.Runes)
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.status = m.entitiesEdit.returnStatus
		m.err = nil
		m.errContext = ""
		return m, nil
	case "enter":
		if m.entitiesEdit.renaming {
			m.entitiesEdit.renaming = false
			return m, nil
		}
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveEntitiesCmd()
	case "up", "k":
		if !m.entitiesEdit.renaming {
			m.moveEntitySelection(-1)
		}
		return m, nil
	case "down", "j":
		if !m.entitiesEdit.renaming {
			m.moveEntitySelection(1)
		}
		return m, nil
	case "a":
		if !m.entitiesEdit.renaming {
			m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(m.nextEntityPlaceholder()))
			m.entitiesEdit.original = append(m.entitiesEdit.original, "")
			m.entitiesEdit.selected = len(m.entitiesEdit.entities) - 1
		}
		return m, nil
	case "r":
		if !m.entitiesEdit.renaming && len(m.entitiesEdit.entities) > 0 {
			m.entitiesEdit.renaming = true
		}
		return m, nil
	case "d":
		if !m.entitiesEdit.renaming && len(m.entitiesEdit.entities) > 1 {
			selected := m.entitiesEdit.selected
			m.entitiesEdit.entities = append(m.entitiesEdit.entities[:selected], m.entitiesEdit.entities[selected+1:]...)
			m.entitiesEdit.original = append(m.entitiesEdit.original[:selected], m.entitiesEdit.original[selected+1:]...)
			if m.entitiesEdit.selected >= len(m.entitiesEdit.entities) {
				m.entitiesEdit.selected = len(m.entitiesEdit.entities) - 1
			}
		}
		return m, nil
	case "f":
		if !m.entitiesEdit.renaming && len(m.entitiesEdit.entities) > 0 {
			m.startFieldsEditing()
		}
		return m, nil
	case "left":
		if m.entitiesEdit.renaming {
			m.selectedEntityField().move(-1)
		}
		return m, nil
	case "right":
		if m.entitiesEdit.renaming {
			m.selectedEntityField().move(1)
		}
		return m, nil
	case "backspace":
		if m.entitiesEdit.renaming {
			m.selectedEntityField().backspace()
		}
		return m, nil
	case "delete":
		if m.entitiesEdit.renaming {
			m.selectedEntityField().delete()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateFieldsEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && (m.fieldsEdit.editingName || m.fieldsEdit.editingType) {
		m.selectedFieldText().insert(msg.Runes)
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.edit.mode = editModeEntities
		m.err = nil
		m.errContext = ""
		return m, nil
	case "enter":
		if m.fieldsEdit.editingName || m.fieldsEdit.editingType {
			m.fieldsEdit.editingName = false
			m.fieldsEdit.editingType = false
			return m, nil
		}
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveFieldsCmd()
	case "up", "k":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType {
			m.moveFieldSelection(-1)
		}
		return m, nil
	case "down", "j":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType {
			m.moveFieldSelection(1)
		}
		return m, nil
	case "a":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType {
			m.fieldsEdit.fields = append(m.fieldsEdit.fields, fieldEditItem{name: newTextField(m.nextFieldPlaceholder()), typeName: newTextField("string")})
			m.fieldsEdit.selected = len(m.fieldsEdit.fields) - 1
		}
		return m, nil
	case "r":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType && len(m.fieldsEdit.fields) > 0 {
			m.fieldsEdit.editingName = true
		}
		return m, nil
	case "t":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType && len(m.fieldsEdit.fields) > 0 {
			m.fieldsEdit.editingType = true
		}
		return m, nil
	case "d":
		if !m.fieldsEdit.editingName && !m.fieldsEdit.editingType && len(m.fieldsEdit.fields) > 1 {
			selected := m.fieldsEdit.selected
			m.fieldsEdit.fields = append(m.fieldsEdit.fields[:selected], m.fieldsEdit.fields[selected+1:]...)
			if m.fieldsEdit.selected >= len(m.fieldsEdit.fields) {
				m.fieldsEdit.selected = len(m.fieldsEdit.fields) - 1
			}
		}
		return m, nil
	case "left":
		if m.fieldsEdit.editingName || m.fieldsEdit.editingType {
			m.selectedFieldText().move(-1)
		}
		return m, nil
	case "right":
		if m.fieldsEdit.editingName || m.fieldsEdit.editingType {
			m.selectedFieldText().move(1)
		}
		return m, nil
	case "backspace":
		if m.fieldsEdit.editingName || m.fieldsEdit.editingType {
			m.selectedFieldText().backspace()
		}
		return m, nil
	case "delete":
		if m.fieldsEdit.editingName || m.fieldsEdit.editingType {
			m.selectedFieldText().delete()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateServicesEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.status = m.servicesEdit.returnStatus
		m.err = nil
		m.errContext = ""
		return m, nil
	case "enter":
		if m.servicesEdit.renaming {
			m.servicesEdit.renaming = false
			return m, nil
		}
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveServicesCmd()
	case "up", "k":
		if !m.servicesEdit.renaming {
			m.moveServiceSelection(-1)
		}
		return m, nil
	case "down", "j":
		if !m.servicesEdit.renaming {
			m.moveServiceSelection(1)
		}
		return m, nil
	case "a":
		if !m.servicesEdit.renaming {
			m.servicesEdit.services = append(m.servicesEdit.services, newTextField(m.nextServicePlaceholder()))
			m.servicesEdit.original = append(m.servicesEdit.original, "")
			m.servicesEdit.selected = len(m.servicesEdit.services) - 1
		}
		return m, nil
	case "r":
		if !m.servicesEdit.renaming && len(m.servicesEdit.services) > 0 {
			m.servicesEdit.renaming = true
		}
		return m, nil
	case "d":
		if !m.servicesEdit.renaming && len(m.servicesEdit.services) > 1 {
			selected := m.servicesEdit.selected
			m.servicesEdit.services = append(m.servicesEdit.services[:selected], m.servicesEdit.services[selected+1:]...)
			m.servicesEdit.original = append(m.servicesEdit.original[:selected], m.servicesEdit.original[selected+1:]...)
			if m.servicesEdit.selected >= len(m.servicesEdit.services) {
				m.servicesEdit.selected = len(m.servicesEdit.services) - 1
			}
		}
		return m, nil
	case "left":
		if m.servicesEdit.renaming {
			m.selectedServiceField().move(-1)
		}
		return m, nil
	case "right":
		if m.servicesEdit.renaming {
			m.selectedServiceField().move(1)
		}
		return m, nil
	case "backspace":
		if m.servicesEdit.renaming {
			m.selectedServiceField().backspace()
		}
		return m, nil
	case "delete":
		if m.servicesEdit.renaming {
			m.selectedServiceField().delete()
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes && m.servicesEdit.renaming {
		m.selectedServiceField().insert(msg.Runes)
	}
	return m, nil
}

func (m *Model) moveServiceSelection(delta int) {
	if len(m.servicesEdit.services) == 0 {
		m.servicesEdit.selected = 0
		return
	}
	m.servicesEdit.selected += delta
	if m.servicesEdit.selected < 0 {
		m.servicesEdit.selected = 0
	}
	if m.servicesEdit.selected >= len(m.servicesEdit.services) {
		m.servicesEdit.selected = len(m.servicesEdit.services) - 1
	}
}

func (m *Model) moveSelectedService(delta int) {
	m.selectedService += delta
	m.clampSelectedService()
}

func (m *Model) clampSelectedService() {
	if len(m.plan.Config.Services) == 0 {
		m.selectedService = 0
		return
	}
	if m.selectedService < 0 {
		m.selectedService = 0
	}
	if m.selectedService >= len(m.plan.Config.Services) {
		m.selectedService = len(m.plan.Config.Services) - 1
	}
}

func (m Model) selectedServiceSummary() application.ServiceSummary {
	if len(m.plan.Config.Services) == 0 {
		return application.ServiceSummary{Name: "CatalogService", EntityNames: []string{"Catalog"}}
	}
	selected := m.selectedService
	if selected < 0 {
		selected = 0
	}
	if selected >= len(m.plan.Config.Services) {
		selected = len(m.plan.Config.Services) - 1
	}
	return m.plan.Config.Services[selected]
}

func (m Model) selectedEntitySummary() application.EntitySummary {
	service := m.selectedServiceSummary()
	name := "Product"
	if len(m.entitiesEdit.entities) > 0 {
		name = m.entitiesEdit.entities[m.entitiesEdit.selected].string()
	}
	if m.entitiesEdit.selected < len(service.Entities) && service.Entities[m.entitiesEdit.selected].Name == name {
		return service.Entities[m.entitiesEdit.selected]
	}
	for _, entity := range service.Entities {
		if entity.Name == name {
			return entity
		}
	}
	return application.EntitySummary{Name: name, Fields: []application.FieldSummary{{Name: "Id", Type: "Guid"}}}
}

func (m *Model) moveEntitySelection(delta int) {
	if len(m.entitiesEdit.entities) == 0 {
		m.entitiesEdit.selected = 0
		return
	}
	m.entitiesEdit.selected += delta
	if m.entitiesEdit.selected < 0 {
		m.entitiesEdit.selected = 0
	}
	if m.entitiesEdit.selected >= len(m.entitiesEdit.entities) {
		m.entitiesEdit.selected = len(m.entitiesEdit.entities) - 1
	}
}

func (m *Model) selectedEntityField() *textField {
	if len(m.entitiesEdit.entities) == 0 {
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(m.nextEntityPlaceholder()))
		m.entitiesEdit.original = append(m.entitiesEdit.original, "")
		m.entitiesEdit.selected = 0
	}
	return &m.entitiesEdit.entities[m.entitiesEdit.selected]
}

func (m *Model) moveFieldSelection(delta int) {
	if len(m.fieldsEdit.fields) == 0 {
		m.fieldsEdit.selected = 0
		return
	}
	m.fieldsEdit.selected += delta
	if m.fieldsEdit.selected < 0 {
		m.fieldsEdit.selected = 0
	}
	if m.fieldsEdit.selected >= len(m.fieldsEdit.fields) {
		m.fieldsEdit.selected = len(m.fieldsEdit.fields) - 1
	}
}

func (m *Model) selectedFieldText() *textField {
	if len(m.fieldsEdit.fields) == 0 {
		m.fieldsEdit.fields = append(m.fieldsEdit.fields, fieldEditItem{name: newTextField(m.nextFieldPlaceholder()), typeName: newTextField("string")})
		m.fieldsEdit.selected = 0
	}
	if m.fieldsEdit.editingType {
		return &m.fieldsEdit.fields[m.fieldsEdit.selected].typeName
	}
	return &m.fieldsEdit.fields[m.fieldsEdit.selected].name
}

func (m Model) nextFieldPlaceholder() string {
	used := map[string]bool{}
	for _, field := range m.fieldsEdit.fields {
		used[field.name.string()] = true
	}
	if !used["Name"] {
		return "Name"
	}
	for index := len(m.fieldsEdit.fields) + 1; ; index++ {
		name := fmt.Sprintf("Field%d", index)
		if !used[name] {
			return name
		}
	}
}

func (m Model) nextEntityPlaceholder() string {
	used := map[string]bool{}
	for _, entity := range m.entitiesEdit.entities {
		used[entity.string()] = true
	}
	for index := len(m.entitiesEdit.entities) + 1; ; index++ {
		name := fmt.Sprintf("Entity%d", index)
		if !used[name] {
			return name
		}
	}
}

func (m *Model) selectedServiceField() *textField {
	if len(m.servicesEdit.services) == 0 {
		m.servicesEdit.services = append(m.servicesEdit.services, newTextField("CatalogService"))
		m.servicesEdit.original = append(m.servicesEdit.original, "")
		m.servicesEdit.selected = 0
	}
	return &m.servicesEdit.services[m.servicesEdit.selected]
}

func (m Model) nextServicePlaceholder() string {
	used := map[string]bool{}
	for _, service := range m.servicesEdit.services {
		used[service.string()] = true
	}
	for index := len(m.servicesEdit.services) + 1; ; index++ {
		name := fmt.Sprintf("Service%dService", index)
		if !used[name] {
			return name
		}
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

type servicesFinishedMsg struct {
	result application.UpdateServiceSettingsResult
	err    error
}

type entitiesFinishedMsg struct {
	result application.UpdateEntitySettingsResult
	err    error
}

type fieldsFinishedMsg struct {
	result application.UpdateFieldSettingsResult
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

func (m Model) saveServicesCmd() tea.Cmd {
	settings := application.ServiceSettings{Services: make([]application.ServiceNameSetting, 0, len(m.servicesEdit.services))}
	for index, service := range m.servicesEdit.services {
		original := ""
		if index < len(m.servicesEdit.original) {
			original = m.servicesEdit.original[index]
		}
		settings.Services = append(settings.Services, application.ServiceNameSetting{OriginalName: original, Name: service.string()})
	}
	return func() tea.Msg {
		result, err := m.updateServices(m.request, settings)
		return servicesFinishedMsg{result: result, err: err}
	}
}

func (m Model) saveEntitiesCmd() tea.Cmd {
	settings := application.EntitySettings{ServiceName: m.entitiesEdit.serviceName, Entities: make([]application.EntityNameSetting, 0, len(m.entitiesEdit.entities))}
	for index, entity := range m.entitiesEdit.entities {
		original := ""
		if index < len(m.entitiesEdit.original) {
			original = m.entitiesEdit.original[index]
		}
		settings.Entities = append(settings.Entities, application.EntityNameSetting{OriginalName: original, Name: entity.string()})
	}
	return func() tea.Msg {
		result, err := m.updateEntities(m.request, settings)
		return entitiesFinishedMsg{result: result, err: err}
	}
}

func (m Model) saveFieldsCmd() tea.Cmd {
	settings := application.FieldSettings{ServiceName: m.fieldsEdit.serviceName, EntityName: m.fieldsEdit.entityName, Fields: make([]application.FieldSetting, 0, len(m.fieldsEdit.fields))}
	for _, field := range m.fieldsEdit.fields {
		settings.Fields = append(settings.Fields, application.FieldSetting{OriginalName: field.originalName, Name: field.name.string(), Type: field.typeName.string()})
	}
	return func() tea.Msg {
		result, err := m.updateFields(m.request, settings)
		return fieldsFinishedMsg{result: result, err: err}
	}
}

func (m Model) View() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s - %s\n", appTitleStyle.Render("Microgen"), m.statusBadge())
	fmt.Fprintf(&builder, "%s\n", dimStyle.Render("Step-based generator dashboard"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Primary:"), m.primaryActionStyle().Render(m.primaryAction()))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.stepperCard())
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.currentStepPanel())
	if m.message != "" {
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.footerCard())
	return builder.String()
}

func (m Model) stepperCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Wizard"))
	for step := stepSource; step < stepCount; step++ {
		marker := " "
		style := dimStyle
		if step < m.currentStep {
			marker = "x"
			style = successStyle
		}
		if step == m.currentStep {
			marker = ">"
			style = readyStyle
		}
		fmt.Fprintf(&builder, "%s %d/%d %s\n", style.Render(marker), step+1, stepCount, style.Render(step.label()))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Progress"), fmt.Sprintf("%d/%d", m.currentStep+1, stepCount))
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) currentStepPanel() string {
	if m.postSaveRefreshFailed() {
		return strings.Join([]string{m.projectStepCard(), m.generateStepCard()}, "\n\n")
	}
	switch m.currentStep {
	case stepProject:
		return m.projectStepCard()
	case stepServices:
		return m.servicesStepCard()
	case stepPreview:
		return strings.Join([]string{m.outputPreviewCard(), m.plannedFilesCard()}, "\n\n")
	case stepGenerate:
		return m.generateStepCard()
	default:
		return m.sourceStepCard()
	}
}

func (m Model) sourceStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Source"))
	fmt.Fprintf(&builder, "%s %s %s\n", labelStyle.Render("Source"), m.request.ConfigPath, dimStyle.Render("("+m.configSourceLabel()+")"))
	if m.request.ConfigBootstrapped {
		fmt.Fprintln(&builder, dimStyle.Render("Created starter config. Edit project, service, entity, and basic field settings incrementally."))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Product"), m.plan.Config.SolutionName)
	if m.plan.Config.SolutionDescription != "" {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Description"), m.plan.Config.SolutionDescription)
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Target"), m.plan.Config.TargetFramework)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Output"), m.plan.OutputDir)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Mode"), m.plan.OutputAction)
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "Use this step to confirm where the JSON comes from and where generated files will be planned.")
	fmt.Fprintln(&builder, dimStyle.Render("Project, service, entity, and basic field editing are available; value objects are planned for later steps."))
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) projectStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Project"))
	if m.status == statusEditing || m.status == statusSaving {
		m.renderSettingsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Solution"), m.plan.Config.SolutionName)
	if m.plan.Config.SolutionDescription != "" {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Description"), m.plan.Config.SolutionDescription)
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Target"), m.plan.Config.TargetFramework)
	if m.plan.Config.SolutionFormat != "" {
		fmt.Fprintf(&builder, "%s .%s\n", labelStyle.Render("Format"), m.plan.Config.SolutionFormat)
	}
	fmt.Fprintln(&builder)
	if m.busy() || m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, busyStyle.Render("Project editing is paused until the current operation finishes or the stale plan is refreshed."))
	} else {
		fmt.Fprintln(&builder, successStyle.Render("e Edit solution name, description, or target framework."))
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) servicesStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
	if (m.status == statusEditing && m.edit.mode == editModeServices) || (m.status == statusSaving && m.edit.mode == editModeServices) {
		m.renderServicesEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if (m.status == statusEditing && m.edit.mode == editModeEntities) || (m.status == statusSaving && m.edit.mode == editModeEntities) {
		m.renderEntitiesEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if (m.status == statusEditing && m.edit.mode == editModeFields) || (m.status == statusSaving && m.edit.mode == editModeFields) {
		m.renderFieldsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	fmt.Fprintf(&builder, "%s %d services, %d entities, %d value objects\n", labelStyle.Render("Summary"), m.plan.Config.ServiceCount, m.plan.Config.EntityCount, m.plan.Config.ValueObjectCount)
	if len(m.plan.Config.Services) > 0 {
		for index, service := range m.plan.Config.Services {
			cursor := " "
			if index == m.selectedService {
				cursor = ">"
			}
			entities := "no entities"
			if len(service.EntityNames) > 0 {
				entities = strings.Join(service.EntityNames, ", ")
			}
			fmt.Fprintf(&builder, "%s %s: %s\n", cursor, service.Name, entities)
		}
	} else if len(m.plan.Config.ServiceNames) > 0 {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Services"), strings.Join(m.plan.Config.ServiceNames, ", "))
	}
	fmt.Fprintln(&builder)
	if m.busy() || m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, busyStyle.Render("Service and entity editing is paused until the current operation finishes or the stale plan is refreshed."))
	} else {
		fmt.Fprintln(&builder, successStyle.Render("up/down choose service, enter edit entities, e edit services."))
	}
	fmt.Fprintln(&builder, dimStyle.Render("Entity fields can be edited from the entity editor; value objects are upcoming. New entities get Id Guid."))
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) generateStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Generate"))
	switch m.status {
	case statusGenerating:
		fmt.Fprintln(&builder, busyStyle.Render("Generating files. Please wait; exit is available after generation finishes."))
	case statusGenerated:
		fmt.Fprintln(&builder, successStyle.Render(fmt.Sprintf("Generated %d files in %s.", m.result.Plan.FileCount, m.result.OutputDir)))
		if m.result.Warning != "" {
			fmt.Fprintf(&builder, "%s %s\n", warningStyle.Render("WARNING"), warningStyle.Render(m.result.Warning))
		}
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
	default:
		fmt.Fprintf(&builder, "%s Generate %d planned file(s) into %s.\n", successStyle.Render("g"), m.plan.FileCount, m.plan.OutputDir)
		fmt.Fprintf(&builder, "%s Review the Preview step before confirming writes.\n", labelStyle.Render("Before"))
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) footerCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Shortcuts"))
	if m.status == statusRefreshing {
		fmt.Fprintln(&builder, busyStyle.Render("Refreshing plan. Please wait; editing, filtering, and generation are paused."))
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if m.status == statusGenerating {
		fmt.Fprintln(&builder, busyStyle.Render("Generating files. Please wait; exit is available after generation finishes."))
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if m.status == statusSaving {
		fmt.Fprintln(&builder, busyStyle.Render("Saving settings. Please wait; exit is available after save finishes."))
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, "Keys: r retry refresh | q/esc/ctrl+c quit")
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	fmt.Fprintln(&builder, stepHelp)
	switch m.currentStep {
	case stepProject:
		fmt.Fprintln(&builder, "Project: e edit settings, r refresh.")
	case stepServices:
		fmt.Fprintln(&builder, "Services: e edit services, enter edit entities, r refresh.")
	case stepPreview:
		fmt.Fprintln(&builder, readyHelp)
	case stepGenerate:
		if m.status == statusGenerated {
			fmt.Fprintln(&builder, generatedHelp)
		} else {
			fmt.Fprintln(&builder, "Generate: g generate, r refresh.")
		}
	default:
		fmt.Fprintln(&builder, "Actions: r refresh. Move to Project to edit or Generate to write files.")
	}
	fmt.Fprintln(&builder, "Exit: q/esc/ctrl+c")
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) configCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Config"))
	fmt.Fprintf(&builder, "%s %s %s\n", labelStyle.Render("Source"), m.request.ConfigPath, dimStyle.Render("("+m.configSourceLabel()+")"))
	if m.request.ConfigBootstrapped {
		fmt.Fprintln(&builder, dimStyle.Render("Created starter config. Edit project, service, entity, and basic field settings incrementally."))
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
		fmt.Fprintln(&builder, "e Edit solution or service settings. Enter on Services edits selected service entities.")
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
	fmt.Fprintln(builder, "Use the Services step for service, entity, and basic field editing. Value-object editing is upcoming.")
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

func (m Model) renderServicesEditor(builder *strings.Builder) {
	if m.status == statusSaving {
		fmt.Fprintln(builder, "Saving services...")
		fmt.Fprintln(builder, "Save is in progress. Exit will be available after it finishes.")
		return
	}
	fmt.Fprintln(builder, "Editing services")
	fmt.Fprintln(builder, "Press enter from the Services step to edit entities and their fields. Value-object editing is upcoming.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	for index, service := range m.servicesEdit.services {
		cursor := " "
		if index == m.servicesEdit.selected {
			cursor = ">"
		}
		label := service.string()
		if index == m.servicesEdit.selected && m.servicesEdit.renaming {
			label += "_"
		}
		fmt.Fprintf(builder, "%s %d. %s\n", cursor, index+1, label)
	}
	fmt.Fprintln(builder)
	if m.servicesEdit.renaming {
		fmt.Fprintln(builder, "Rename mode: type the service name. Enter confirms the local rename. Esc cancels service editing.")
		fmt.Fprintln(builder, "Left/right, backspace, and delete edit the selected service name.")
		return
	}
	fmt.Fprintln(builder, "Keys: up/down select, a add, r rename, d delete, enter save, esc cancel.")
	fmt.Fprintln(builder, "Deletion keeps at least one service locally; final validation runs before save.")
}

func (m Model) renderEntitiesEditor(builder *strings.Builder) {
	if m.status == statusSaving {
		fmt.Fprintln(builder, "Saving entities...")
		fmt.Fprintln(builder, "Save is in progress. Exit will be available after it finishes.")
		return
	}
	fmt.Fprintf(builder, "Editing entities for %s\n", m.entitiesEdit.serviceName)
	fmt.Fprintln(builder, "Press f to edit fields for the selected saved entity. New entities get Id Guid.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	for index, entity := range m.entitiesEdit.entities {
		cursor := " "
		if index == m.entitiesEdit.selected {
			cursor = ">"
		}
		label := entity.string()
		if index == m.entitiesEdit.selected && m.entitiesEdit.renaming {
			label += "_"
		}
		fmt.Fprintf(builder, "%s %d. %s\n", cursor, index+1, label)
	}
	fmt.Fprintln(builder)
	if m.entitiesEdit.renaming {
		fmt.Fprintln(builder, "Rename mode: type the entity name. Enter confirms the local rename. Esc cancels entity editing.")
		fmt.Fprintln(builder, "Left/right, backspace, and delete edit the selected entity name.")
		return
	}
	fmt.Fprintln(builder, "Keys: up/down select, a add, r rename, d delete, f fields, enter save, esc cancel.")
	fmt.Fprintln(builder, "Deletion keeps at least one entity locally; final validation runs before save.")
}

func (m Model) renderFieldsEditor(builder *strings.Builder) {
	if m.status == statusSaving {
		fmt.Fprintln(builder, "Saving fields...")
		fmt.Fprintln(builder, "Save is in progress. Exit will be available after it finishes.")
		return
	}
	fmt.Fprintf(builder, "Editing fields for %s/%s\n", m.fieldsEdit.serviceName, m.fieldsEdit.entityName)
	fmt.Fprintln(builder, "Field details are name and type. Value-object validation rules are upcoming.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	for index, field := range m.fieldsEdit.fields {
		cursor := " "
		if index == m.fieldsEdit.selected {
			cursor = ">"
		}
		name := field.name.string()
		fieldType := field.typeName.string()
		if index == m.fieldsEdit.selected && m.fieldsEdit.editingName {
			name += "_"
		}
		if index == m.fieldsEdit.selected && m.fieldsEdit.editingType {
			fieldType += "_"
		}
		fmt.Fprintf(builder, "%s %d. %s: %s\n", cursor, index+1, name, fieldType)
	}
	fmt.Fprintln(builder)
	if m.fieldsEdit.editingName {
		fmt.Fprintln(builder, "Rename mode: type the field name. Enter confirms the local rename. Esc returns to entities.")
		fmt.Fprintln(builder, "Left/right, backspace, and delete edit the selected field name.")
		return
	}
	if m.fieldsEdit.editingType {
		fmt.Fprintln(builder, "Type mode: enter a scalar type such as string, Guid, int, decimal, bool, DateTime, double, or long.")
		fmt.Fprintln(builder, "Enter confirms the local type. Esc returns to entities.")
		return
	}
	fmt.Fprintln(builder, "Keys: up/down select, a add string field, r rename, t edit type, d delete, enter save, esc back.")
	fmt.Fprintln(builder, "Deletion keeps at least one field locally; config validation still requires exactly one Id Guid field.")
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
