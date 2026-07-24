package tui

import (
	"fmt"
	"sort"
	"strconv"
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
	wideTerminalWidth     = 100
	mediumTerminalWidth   = 76
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
type UpdateValueObjectsFunc func(application.GenerateRequest, application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error)

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

type workspaceScreen int

const (
	screenOverview workspaceScreen = iota
	screenProject
	screenServices
	screenPreview
	screenGenerate
	screenCount
)

type layoutMode int

const (
	layoutNarrow layoutMode = iota
	layoutMedium
	layoutWide
)

type serviceResourceContext int

const (
	serviceResourceServices serviceResourceContext = iota
	serviceResourceEntities
	serviceResourceValueObjects
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
	editModeValueObjects
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

type valueObjectEditItem struct {
	originalName string
	name         textField
	typeName     textField
	rules        valueObjectRuleEditState
}

type valueObjectRuleEditState struct {
	required       bool
	minLength      textField
	maxLength      textField
	pattern        textField
	validExample   textField
	invalidExample textField
	minimum        textField
	maximum        textField
	notEmpty       bool
	notDefault     bool
}

type valueObjectRuleField int

const (
	valueObjectRuleFieldType valueObjectRuleField = iota
	valueObjectRuleFieldRequired
	valueObjectRuleFieldMinLength
	valueObjectRuleFieldMaxLength
	valueObjectRuleFieldPattern
	valueObjectRuleFieldValidExample
	valueObjectRuleFieldInvalidExample
	valueObjectRuleFieldMinimum
	valueObjectRuleFieldMaximum
	valueObjectRuleFieldNotEmpty
	valueObjectRuleFieldNotDefault
)

type fieldsEditState struct {
	serviceName string
	entityName  string
	fields      []fieldEditItem
	selected    int
	editingName bool
	editingType bool
}

type valueObjectsEditState struct {
	serviceName  string
	valueObjects []valueObjectEditItem
	selected     int
	renaming     bool
	rulesOpen    bool
	editingRule  bool
	focusedRule  valueObjectRuleField
	returnStatus modelStatus
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
	updateValueObjects         UpdateValueObjectsFunc
	status                     modelStatus
	result                     application.GenerateResult
	err                        error
	errContext                 string
	message                    string
	edit                       editState
	servicesEdit               servicesEditState
	entitiesEdit               entitiesEditState
	fieldsEdit                 fieldsEditState
	valueObjectsEdit           valueObjectsEditState
	targetFrameworkSuggestions []string
	fileCursor                 int
	fileOffset                 int
	windowRows                 int
	windowWidth                int
	windowHeight               int
	layout                     layoutMode
	actionFilter               string
	currentStep                tuiStep
	selectedService            int
	selectedEntity             int
	selectedValueObject        int
	serviceContext             serviceResourceContext
	screen                     workspaceScreen
	selectedScreen             workspaceScreen
	helpOpen                   bool
}

func NewModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	suggestions := []string(nil)
	if len(targetFrameworkSuggestions) > 0 {
		suggestions = append([]string(nil), targetFrameworkSuggestions[0]...)
	}
	return Model{plan: plan, request: request, planFunc: planFunc, generate: generate, update: update, status: statusReady, targetFrameworkSuggestions: suggestions, windowRows: defaultFileWindowRows, layout: layoutModeForWidth(0), currentStep: stepSource, screen: screenOverview, selectedScreen: screenOverview}
}

func Run(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, updateServices UpdateServicesFunc, updateEntities UpdateEntitiesFunc, updateFields UpdateFieldsFunc, updateValueObjects UpdateValueObjectsFunc, targetFrameworkSuggestions []string) error {
	model := NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions)
	model.updateServices = updateServices
	model.updateEntities = updateEntities
	model.updateFields = updateFields
	model.updateValueObjects = updateValueObjects
	_, err := tea.NewProgram(model).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.layout = layoutModeForWidth(msg.Width)
		m.windowRows = visibleFileRows(msg.Height)
		m.clampFileCursor()
		return m, nil
	case tea.KeyMsg:
		if m.helpOpen {
			if msg.String() == "esc" || msg.String() == "?" {
				m.helpOpen = false
			}
			return m, nil
		}
		if m.status == statusEditing {
			if m.edit.mode == editModeValueObjects {
				return m.updateValueObjectsEdit(msg)
			}
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
		case "tab":
			if m.busy() {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				m.moveServiceContext(1)
				return m, nil
			}
			m.moveStep(1)
			return m, nil
		case "]":
			if m.busy() {
				return m, nil
			}
			m.moveStep(1)
			return m, nil
		case "shift+tab":
			if m.busy() {
				return m, nil
			}
			m.moveStep(-1)
			return m, nil
		case "[":
			if m.busy() {
				return m, nil
			}
			m.moveStep(-1)
			return m, nil
		case "q", "esc", "ctrl+c":
			if m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if key == "esc" && m.activeScreen() != screenOverview {
				m.openScreen(screenOverview)
				return m, nil
			}
			return m, tea.Quit
		case "?":
			m.helpOpen = true
			return m, nil
		case "1", "2", "3", "4", "5":
			if m.busy() {
				return m, nil
			}
			m.openScreen(workspaceScreen(key[0] - '1'))
			return m, nil
		case "up", "k":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				m.moveSelectedServiceResource(-1)
				return m, nil
			}
			if m.activeScreen() != screenPreview {
				m.moveRouteSelection(-1)
				return m, nil
			}
			m.moveFileCursor(-1)
			return m, nil
		case "down", "j":
			if m.status == statusGenerating || m.status == statusRefreshing || m.status == statusSaving {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				m.moveSelectedServiceResource(1)
				return m, nil
			}
			if m.activeScreen() != screenPreview {
				m.moveRouteSelection(1)
				return m, nil
			}
			m.moveFileCursor(1)
			return m, nil
		case "left", "right":
			if m.busy() {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				delta := 1
				if key == "left" {
					delta = -1
				}
				m.moveServiceContext(delta)
				return m, nil
			}
			if key == "left" {
				m.navigateRoute(-1)
			} else {
				m.navigateRoute(1)
			}
			return m, nil
		case "h", "l":
			if m.busy() {
				return m, nil
			}
			if key == "h" {
				m.navigateRoute(-1)
			} else {
				m.navigateRoute(1)
			}
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
			m.openScreen(screenGenerate)
			m.err = nil
			m.errContext = ""
			m.message = ""
			return m, m.generateCmd()
		case "e":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				m.startServicesEditing()
			} else {
				m.openScreen(screenProject)
				m.startEditing()
			}
			return m, nil
		case "enter":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				switch m.serviceContext {
				case serviceResourceEntities:
					m.startEntitiesEditing()
				case serviceResourceValueObjects:
					m.startValueObjectsEditing()
				default:
					// Keep the established service-selection flow: Enter opens the selected service's entities.
					m.startEntitiesEditing()
				}
				return m, nil
			}
			if m.selectedScreen != m.activeScreen() {
				m.openScreen(m.selectedScreen)
			}
			return m, nil
		case "v":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.activeScreen() == screenServices {
				m.startValueObjectsEditing()
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
			m.openScreen(screenGenerate)
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
			m.openScreen(screenGenerate)
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
		m.message = "Settings saved. Plan refreshed. Use Services to edit services, entities, fields, and value objects."
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
		m.message = "Fields saved. Plan refreshed. Value-object names can be edited from the Services step."
		m.clampSelectedService()
		m.clampFileCursor()
		return m, nil

	case valueObjectsFinishedMsg:
		if msg.err != nil {
			m.status = statusEditing
			m.edit.mode = editModeValueObjects
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
			m.message = "Value objects saved, but the plan refresh failed. Press r to retry the refresh."
			return m, nil
		}
		m.status = statusReady
		m.plan = msg.result.Plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = "Value objects saved. Plan refreshed. Basic type and rules can be edited from the value objects editor."
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

func layoutModeForWidth(width int) layoutMode {
	if width >= wideTerminalWidth {
		return layoutWide
	}
	if width >= mediumTerminalWidth {
		return layoutMedium
	}
	return layoutNarrow
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func screenForStep(step tuiStep) workspaceScreen {
	switch step {
	case stepProject:
		return screenProject
	case stepServices:
		return screenServices
	case stepPreview:
		return screenPreview
	case stepGenerate:
		return screenGenerate
	default:
		return screenOverview
	}
}

func stepForScreen(screen workspaceScreen) tuiStep {
	switch screen {
	case screenProject:
		return stepProject
	case screenServices:
		return stepServices
	case screenPreview:
		return stepPreview
	case screenGenerate:
		return stepGenerate
	default:
		return stepSource
	}
}

func (screen workspaceScreen) label() string {
	switch screen {
	case screenProject:
		return "Project"
	case screenServices:
		return "Services"
	case screenPreview:
		return "Preview"
	case screenGenerate:
		return "Generate"
	default:
		return "Overview"
	}
}

func (m Model) activeScreen() workspaceScreen {
	if m.screen == screenOverview && m.currentStep != stepSource {
		return screenForStep(m.currentStep)
	}
	return m.screen
}

func (m *Model) openScreen(screen workspaceScreen) {
	screen = workspaceScreen(clampInt(int(screen), int(screenOverview), int(screenCount)-1))
	m.screen = screen
	m.selectedScreen = screen
	m.currentStep = stepForScreen(screen)
}

func (m *Model) moveRouteSelection(delta int) {
	m.selectedScreen = workspaceScreen(clampInt(int(m.selectedScreen)+delta, int(screenOverview), int(screenCount)-1))
}

func (m *Model) navigateRoute(delta int) {
	m.selectedScreen = workspaceScreen(clampInt(int(m.activeScreen())+delta, int(screenOverview), int(screenCount)-1))
	m.openScreen(m.selectedScreen)
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
	m.openScreen(screenProject)
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
	m.openScreen(screenServices)
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
	m.openScreen(screenServices)
	returnStatus := m.status
	m.status = statusEditing
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.edit = editState{mode: editModeEntities}
	service := m.selectedServiceSummary()
	entities := m.serviceEntitySummaries()
	entityNames := make([]string, 0, len(entities))
	for _, entity := range entities {
		entityNames = append(entityNames, entity.Name)
	}
	m.entitiesEdit = entitiesEditState{returnStatus: returnStatus, serviceName: service.Name, original: append([]string(nil), entityNames...), entities: make([]textField, 0, len(entityNames))}
	for _, name := range entityNames {
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(name))
	}
	if len(m.entitiesEdit.entities) == 0 {
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(m.nextEntityPlaceholder()))
		m.entitiesEdit.original = append(m.entitiesEdit.original, "")
	}
	m.entitiesEdit.selected = clampInt(m.selectedEntity, 0, len(m.entitiesEdit.entities)-1)
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

func (m *Model) startValueObjectsEditing() {
	m.openScreen(screenServices)
	returnStatus := m.status
	service := m.selectedServiceSummary()
	m.status = statusEditing
	m.err = nil
	m.errContext = ""
	m.message = ""
	m.edit = editState{mode: editModeValueObjects}
	valueObjects := m.serviceValueObjectSummaries()
	m.valueObjectsEdit = valueObjectsEditState{returnStatus: returnStatus, serviceName: service.Name, valueObjects: make([]valueObjectEditItem, 0, len(valueObjects))}
	for _, valueObject := range valueObjects {
		if len(service.ValueObjects) > 0 {
			m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, valueObjectEditItemFromSummary(valueObject))
		} else {
			m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, newValueObjectEditItem(valueObject.Name, valueObject.Name))
		}
	}
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, newValueObjectEditItem("", m.nextValueObjectPlaceholder()))
	}
	m.valueObjectsEdit.selected = clampInt(m.selectedValueObject, 0, len(m.valueObjectsEdit.valueObjects)-1)
}

func valueObjectEditItemFromSummary(summary application.ValueObjectSummary) valueObjectEditItem {
	item := newValueObjectEditItem(summary.Name, summary.Name)
	item.typeName = newTextField(summary.Type)
	item.rules = valueObjectRuleEditStateFromSummary(summary.Validations)
	return item
}

func newValueObjectEditItem(originalName, name string) valueObjectEditItem {
	return valueObjectEditItem{originalName: originalName, name: newTextField(name), typeName: newTextField("string"), rules: valueObjectRuleEditState{required: true, minLength: newTextField("1"), maxLength: newTextField("100"), validExample: newTextField("Sample")}}
}

func valueObjectRuleEditStateFromSummary(summary application.ValidationRuleSummary) valueObjectRuleEditState {
	rules := valueObjectRuleEditState{}
	if summary.Required != nil {
		rules.required = *summary.Required
	}
	rules.minLength = intRuleText(summary.MinLength)
	rules.maxLength = intRuleText(summary.MaxLength)
	rules.pattern = stringRuleText(summary.Pattern)
	rules.validExample = stringRuleText(summary.ValidExample)
	rules.invalidExample = stringRuleText(summary.InvalidExample)
	rules.minimum = stringRuleText(summary.Minimum)
	rules.maximum = stringRuleText(summary.Maximum)
	if summary.NotEmpty != nil {
		rules.notEmpty = *summary.NotEmpty
	}
	if summary.NotDefault != nil {
		rules.notDefault = *summary.NotDefault
	}
	return rules
}

func intRuleText(value *int) textField {
	if value == nil {
		return textField{}
	}
	return newTextField(fmt.Sprintf("%d", *value))
}

func stringRuleText(value *string) textField {
	if value == nil {
		return textField{}
	}
	return newTextField(*value)
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
		m.status = m.entitiesEdit.returnStatus
		m.edit.mode = editModeEntities
		m.err = nil
		m.errContext = ""
		m.serviceContext = serviceResourceEntities
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

func (m Model) updateValueObjectsEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && m.valueObjectsEdit.rulesOpen && m.valueObjectsEdit.editingRule {
		if field := m.selectedValueObjectRuleTextField(); field != nil {
			field.insert(msg.Runes)
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes && m.valueObjectsEdit.renaming {
		m.selectedValueObjectField().insert(msg.Runes)
		return m, nil
	}
	if m.valueObjectsEdit.rulesOpen {
		return m.updateValueObjectRulesEdit(msg)
	}
	switch msg.String() {
	case "esc":
		m.status = m.valueObjectsEdit.returnStatus
		m.err = nil
		m.errContext = ""
		return m, nil
	case "enter":
		if m.valueObjectsEdit.renaming {
			m.valueObjectsEdit.renaming = false
			return m, nil
		}
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveValueObjectsCmd()
	case "up", "k":
		if !m.valueObjectsEdit.renaming {
			m.moveValueObjectSelection(-1)
		}
		return m, nil
	case "down", "j":
		if !m.valueObjectsEdit.renaming {
			m.moveValueObjectSelection(1)
		}
		return m, nil
	case "a":
		if !m.valueObjectsEdit.renaming {
			m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, newValueObjectEditItem("", m.nextValueObjectPlaceholder()))
			m.valueObjectsEdit.selected = len(m.valueObjectsEdit.valueObjects) - 1
		}
		return m, nil
	case "r":
		if !m.valueObjectsEdit.renaming && len(m.valueObjectsEdit.valueObjects) > 0 {
			m.valueObjectsEdit.renaming = true
		}
		return m, nil
	case "d":
		if !m.valueObjectsEdit.renaming && len(m.valueObjectsEdit.valueObjects) > 0 {
			selected := m.valueObjectsEdit.selected
			m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects[:selected], m.valueObjectsEdit.valueObjects[selected+1:]...)
			if m.valueObjectsEdit.selected >= len(m.valueObjectsEdit.valueObjects) {
				m.valueObjectsEdit.selected = len(m.valueObjectsEdit.valueObjects) - 1
			}
			if m.valueObjectsEdit.selected < 0 {
				m.valueObjectsEdit.selected = 0
			}
		}
		return m, nil
	case "o":
		if !m.valueObjectsEdit.renaming && len(m.valueObjectsEdit.valueObjects) > 0 {
			m.valueObjectsEdit.rulesOpen = true
			m.valueObjectsEdit.focusedRule = valueObjectRuleFieldType
		}
		return m, nil
	case "left":
		if m.valueObjectsEdit.renaming {
			m.selectedValueObjectField().move(-1)
		}
		return m, nil
	case "right":
		if m.valueObjectsEdit.renaming {
			m.selectedValueObjectField().move(1)
		}
		return m, nil
	case "backspace":
		if m.valueObjectsEdit.renaming {
			m.selectedValueObjectField().backspace()
		}
		return m, nil
	case "delete":
		if m.valueObjectsEdit.renaming {
			m.selectedValueObjectField().delete()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateValueObjectRulesEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.status = m.valueObjectsEdit.returnStatus
		m.err = nil
		m.errContext = ""
		return m, nil
	case "b":
		if !m.valueObjectsEdit.editingRule {
			m.valueObjectsEdit.rulesOpen = false
		}
		return m, nil
	case "enter":
		if m.valueObjectsEdit.editingRule {
			m.valueObjectsEdit.editingRule = false
			return m, nil
		}
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		return m, m.saveValueObjectsCmd()
	case "tab", "down", "j":
		if !m.valueObjectsEdit.editingRule {
			m.moveValueObjectRuleFocus(1)
		}
		return m, nil
	case "shift+tab", "up", "k":
		if !m.valueObjectsEdit.editingRule {
			m.moveValueObjectRuleFocus(-1)
		}
		return m, nil
	case " ":
		if !m.valueObjectsEdit.editingRule {
			m.toggleSelectedValueObjectRule()
		}
		return m, nil
	case "e":
		if !m.valueObjectsEdit.editingRule && m.selectedValueObjectRuleTextField() != nil {
			m.valueObjectsEdit.editingRule = true
		}
		return m, nil
	case "left":
		if m.valueObjectsEdit.editingRule {
			if field := m.selectedValueObjectRuleTextField(); field != nil {
				field.move(-1)
			}
		}
		return m, nil
	case "right":
		if m.valueObjectsEdit.editingRule {
			if field := m.selectedValueObjectRuleTextField(); field != nil {
				field.move(1)
			}
		}
		return m, nil
	case "backspace":
		if m.valueObjectsEdit.editingRule {
			if field := m.selectedValueObjectRuleTextField(); field != nil {
				field.backspace()
			}
		}
		return m, nil
	case "delete":
		if m.valueObjectsEdit.editingRule {
			if field := m.selectedValueObjectRuleTextField(); field != nil {
				field.delete()
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateServicesEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && m.servicesEdit.renaming {
		m.selectedServiceField().insert(msg.Runes)
		return m, nil
	}
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

func (m *Model) moveServiceContext(delta int) {
	m.serviceContext = serviceResourceContext(clampInt(int(m.serviceContext)+delta, int(serviceResourceServices), int(serviceResourceValueObjects)))
	m.clampServiceResourceSelection()
}

func (m *Model) moveSelectedServiceResource(delta int) {
	switch m.serviceContext {
	case serviceResourceEntities:
		m.selectedEntity += delta
	case serviceResourceValueObjects:
		m.selectedValueObject += delta
	default:
		m.moveSelectedService(delta)
		return
	}
	m.clampServiceResourceSelection()
}

func (m *Model) clampSelectedService() {
	serviceCount := len(m.serviceSummaries())
	if serviceCount == 0 {
		m.selectedService = 0
	} else {
		if m.selectedService < 0 {
			m.selectedService = 0
		}
		if m.selectedService >= serviceCount {
			m.selectedService = serviceCount - 1
		}
	}
	m.clampServiceResourceSelection()
}

func (m *Model) clampServiceResourceSelection() {
	entityCount := len(m.serviceEntitySummaries())
	if m.selectedEntity < 0 {
		m.selectedEntity = 0
	}
	if entityCount == 0 || m.selectedEntity >= entityCount {
		m.selectedEntity = maxInt(entityCount-1, 0)
	}

	valueObjectCount := len(m.serviceValueObjectSummaries())
	if m.selectedValueObject < 0 {
		m.selectedValueObject = 0
	}
	if valueObjectCount == 0 || m.selectedValueObject >= valueObjectCount {
		m.selectedValueObject = maxInt(valueObjectCount-1, 0)
	}
}

func maxInt(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func (m Model) selectedServiceSummary() application.ServiceSummary {
	services := m.serviceSummaries()
	if len(services) == 0 {
		return application.ServiceSummary{Name: "CatalogService", EntityNames: []string{"Catalog"}}
	}
	selected := m.selectedService
	if selected < 0 {
		selected = 0
	}
	if selected >= len(services) {
		selected = len(services) - 1
	}
	return services[selected]
}

func (m Model) serviceSummaries() []application.ServiceSummary {
	if len(m.plan.Config.Services) > 0 {
		return m.plan.Config.Services
	}
	services := make([]application.ServiceSummary, 0, len(m.plan.Config.ServiceNames))
	for _, name := range m.plan.Config.ServiceNames {
		services = append(services, application.ServiceSummary{Name: name})
	}
	return services
}

func (m Model) serviceEntitySummaries() []application.EntitySummary {
	service := m.selectedServiceSummary()
	if len(service.Entities) > 0 {
		return service.Entities
	}
	entities := make([]application.EntitySummary, 0, len(service.EntityNames))
	for _, name := range service.EntityNames {
		entities = append(entities, application.EntitySummary{Name: name})
	}
	return entities
}

func (m Model) serviceValueObjectSummaries() []application.ValueObjectSummary {
	service := m.selectedServiceSummary()
	if len(service.ValueObjects) > 0 {
		return service.ValueObjects
	}
	valueObjects := make([]application.ValueObjectSummary, 0, len(service.ValueObjectNames))
	for _, name := range service.ValueObjectNames {
		valueObjects = append(valueObjects, application.ValueObjectSummary{Name: name})
	}
	return valueObjects
}

func (m Model) selectedEntitySummary() application.EntitySummary {
	service := m.selectedServiceSummary()
	name := "Product"
	if len(m.entitiesEdit.entities) > 0 {
		name = m.entitiesEdit.entities[m.entitiesEdit.selected].string()
	} else if entities := m.serviceEntitySummaries(); len(entities) > 0 {
		selected := clampInt(m.selectedEntity, 0, len(entities)-1)
		return entities[selected]
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

func (m *Model) moveValueObjectSelection(delta int) {
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		m.valueObjectsEdit.selected = 0
		return
	}
	m.valueObjectsEdit.selected += delta
	if m.valueObjectsEdit.selected < 0 {
		m.valueObjectsEdit.selected = 0
	}
	if m.valueObjectsEdit.selected >= len(m.valueObjectsEdit.valueObjects) {
		m.valueObjectsEdit.selected = len(m.valueObjectsEdit.valueObjects) - 1
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

func (m *Model) selectedValueObjectField() *textField {
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, newValueObjectEditItem("", m.nextValueObjectPlaceholder()))
		m.valueObjectsEdit.selected = 0
	}
	return &m.valueObjectsEdit.valueObjects[m.valueObjectsEdit.selected].name
}

func (m Model) nextValueObjectPlaceholder() string {
	used := map[string]bool{}
	for _, valueObject := range m.valueObjectsEdit.valueObjects {
		used[valueObject.name.string()] = true
	}
	for index := len(m.valueObjectsEdit.valueObjects) + 1; ; index++ {
		name := fmt.Sprintf("ValueObject%d", index)
		if !used[name] {
			return name
		}
	}
}

func (m *Model) selectedValueObjectRuleTextField() *textField {
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		return nil
	}
	selected := &m.valueObjectsEdit.valueObjects[m.valueObjectsEdit.selected]
	switch m.valueObjectsEdit.focusedRule {
	case valueObjectRuleFieldType:
		return &selected.typeName
	case valueObjectRuleFieldMinLength:
		return &selected.rules.minLength
	case valueObjectRuleFieldMaxLength:
		return &selected.rules.maxLength
	case valueObjectRuleFieldPattern:
		return &selected.rules.pattern
	case valueObjectRuleFieldValidExample:
		return &selected.rules.validExample
	case valueObjectRuleFieldInvalidExample:
		return &selected.rules.invalidExample
	case valueObjectRuleFieldMinimum:
		return &selected.rules.minimum
	case valueObjectRuleFieldMaximum:
		return &selected.rules.maximum
	}
	return nil
}

func (m *Model) moveValueObjectRuleFocus(delta int) {
	fields := m.visibleValueObjectRuleFields()
	if len(fields) == 0 {
		m.valueObjectsEdit.focusedRule = valueObjectRuleFieldType
		return
	}
	position := 0
	for index, field := range fields {
		if field == m.valueObjectsEdit.focusedRule {
			position = index
			break
		}
	}
	position += delta
	if position < 0 {
		position = len(fields) - 1
	}
	if position >= len(fields) {
		position = 0
	}
	m.valueObjectsEdit.focusedRule = fields[position]
}

func (m Model) visibleValueObjectRuleFields() []valueObjectRuleField {
	fields := []valueObjectRuleField{valueObjectRuleFieldType}
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		return fields
	}
	switch m.valueObjectsEdit.valueObjects[m.valueObjectsEdit.selected].typeName.string() {
	case "string":
		fields = append(fields, valueObjectRuleFieldRequired, valueObjectRuleFieldMinLength, valueObjectRuleFieldMaxLength, valueObjectRuleFieldPattern, valueObjectRuleFieldValidExample, valueObjectRuleFieldInvalidExample)
	case "int", "decimal":
		fields = append(fields, valueObjectRuleFieldMinimum, valueObjectRuleFieldMaximum)
	case "Guid":
		fields = append(fields, valueObjectRuleFieldNotEmpty)
	}
	return fields
}

func (m *Model) toggleSelectedValueObjectRule() {
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		return
	}
	rules := &m.valueObjectsEdit.valueObjects[m.valueObjectsEdit.selected].rules
	switch m.valueObjectsEdit.focusedRule {
	case valueObjectRuleFieldRequired:
		rules.required = !rules.required
	case valueObjectRuleFieldNotEmpty:
		rules.notEmpty = !rules.notEmpty
	case valueObjectRuleFieldNotDefault:
		rules.notDefault = !rules.notDefault
	}
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
		m.openScreen(screenProject)
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

type valueObjectsFinishedMsg struct {
	result application.UpdateValueObjectSettingsResult
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

func (m Model) saveValueObjectsCmd() tea.Cmd {
	settings := application.ValueObjectSettings{ServiceName: m.valueObjectsEdit.serviceName, ValueObjects: make([]application.ValueObjectNameSetting, 0, len(m.valueObjectsEdit.valueObjects))}
	for _, valueObject := range m.valueObjectsEdit.valueObjects {
		settings.ValueObjects = append(settings.ValueObjects, application.ValueObjectNameSetting{OriginalName: valueObject.originalName, Name: valueObject.name.string(), Type: valueObject.typeName.string(), Validations: validationRuleSettingsFromEdit(valueObject.typeName.string(), valueObject.rules)})
	}
	return func() tea.Msg {
		result, err := m.updateValueObjects(m.request, settings)
		return valueObjectsFinishedMsg{result: result, err: err}
	}
}

func validationRuleSettingsFromEdit(valueObjectType string, rules valueObjectRuleEditState) application.ValidationRuleSettings {
	switch valueObjectType {
	case "string":
		return application.ValidationRuleSettings{Required: boolPtr(rules.required), MinLength: intPtrFromText(rules.minLength), MaxLength: intPtrFromText(rules.maxLength), Pattern: stringPtrFromText(rules.pattern), ValidExample: stringPtrFromText(rules.validExample), InvalidExample: stringPtrFromText(rules.invalidExample)}
	case "int", "decimal":
		return application.ValidationRuleSettings{Minimum: stringPtrFromText(rules.minimum), Maximum: stringPtrFromText(rules.maximum)}
	case "Guid":
		return application.ValidationRuleSettings{NotEmpty: boolPtr(rules.notEmpty)}
	default:
		return application.ValidationRuleSettings{}
	}
}

func boolPtr(value bool) *bool {
	if !value {
		return nil
	}
	return &value
}

func intPtrFromText(field textField) *int {
	value := strings.TrimSpace(field.string())
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		invalid := -1
		return &invalid
	}
	return &parsed
}

func stringPtrFromText(field textField) *string {
	value := strings.TrimSpace(field.string())
	if value == "" {
		return nil
	}
	return &value
}

func (m Model) View() string {
	if m.helpOpen {
		return m.workspaceHeader() + "\n\n" + m.helpOverlay()
	}
	var builder strings.Builder
	fmt.Fprintln(&builder, m.workspaceHeader())
	if m.layout == layoutWide {
		rail := cardStyle.Width(22).Render(m.navigationRail())
		content := m.workspaceContent()
		fmt.Fprintln(&builder, lipgloss.JoinHorizontal(lipgloss.Top, rail, "  ", content))
	} else {
		fmt.Fprintln(&builder, m.compactNavigation())
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.workspaceContent())
	}
	if m.message != "" {
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.footerCard())
	return builder.String()
}

func (m Model) workspaceHeader() string {
	project := m.plan.Config.SolutionName
	if project == "" {
		project = "Unconfigured project"
	}
	return fmt.Sprintf("%s - %s  %s  %s %s\n%s", appTitleStyle.Render("Microgen"), m.statusBadge(), dimStyle.Render("Workspace"), labelStyle.Render("Current project:"), project, m.primaryActionStyle().Render("Primary: "+m.primaryAction()))
}

func (m Model) navigationRail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Navigation"))
	for screen := screenOverview; screen < screenCount; screen++ {
		cursor := " "
		style := dimStyle
		if screen == m.activeScreen() {
			cursor = ">"
			style = readyStyle
		} else if screen == m.selectedScreen {
			cursor = "*"
			style = labelStyle
		}
		fmt.Fprintf(&builder, "%s %d %s\n", style.Render(cursor), screen+1, style.Render(screen.label()))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, dimStyle.Render("Enter open"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) compactNavigation() string {
	parts := make([]string, 0, int(screenCount))
	for screen := screenOverview; screen < screenCount; screen++ {
		label := fmt.Sprintf("%d:%s", screen+1, screen.label())
		if screen == m.activeScreen() {
			label = readyStyle.Render("[" + label + "]")
		}
		parts = append(parts, label)
	}
	return dimStyle.Render("Navigation ") + strings.Join(parts, "  ")
}

func (m Model) workspaceContent() string {
	if m.postSaveRefreshFailed() {
		return strings.Join([]string{m.projectStepCard(), m.generateStepCard()}, "\n\n")
	}
	switch m.activeScreen() {
	case screenProject:
		return m.projectStepCard()
	case screenServices:
		return m.servicesStepCard()
	case screenPreview:
		return strings.Join([]string{m.outputPreviewCard(), m.plannedFilesCard()}, "\n\n")
	case screenGenerate:
		return m.generateStepCard()
	default:
		return m.overviewCard()
	}
}

func (m Model) helpOverlay() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, cardStyle.Render(strings.Join([]string{
		sectionTitleStyle.Render("Help"),
		"Global: 1-5 open screens | up/down select route | enter open | h/l or left/right switch",
		"Global: ? close help | esc back | q/ctrl+c quit",
		"Overview: r refresh | g generate",
		"Project: e edit settings | r refresh",
		"Services: tab or left/right switch Services/Entities/Value Objects | up/down select | enter open",
		"Services actions: e service list | f fields from entity editor | v value objects | a/r/d edit resources",
		"Preview: arrows/k/j inspect files | a filter | r refresh | g generate",
		"Generate: g generate | r refresh",
	}, "\n")))
	fmt.Fprintln(&builder, dimStyle.Render("Press ? or esc to close help."))
	return builder.String()
}

func (m Model) overviewCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Overview"))
	fmt.Fprintf(&builder, "%s %s %s\n", labelStyle.Render("Source"), m.request.ConfigPath, dimStyle.Render("("+m.configSourceLabel()+")"))
	if m.request.ConfigBootstrapped {
		fmt.Fprintln(&builder, dimStyle.Render("Created starter config. Edit project, service, entity, and basic field settings incrementally."))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Product"), m.plan.Config.SolutionName)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Target"), m.plan.Config.TargetFramework)
	outputDir := m.plan.OutputDir
	if outputDir == "" {
		outputDir = m.request.OutputDir
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Output"), outputDir)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Mode"), m.plan.OutputAction)
	readiness := m.plan.Readiness
	if m.status == statusGenerated {
		readiness = m.result.Plan.Readiness
	}
	projectCount := boolAsInt(readiness.ProjectPresent || m.plan.Config.SolutionName != "")
	serviceCount := readiness.ServiceCount
	if serviceCount == 0 {
		serviceCount = m.plan.Config.ServiceCount
	}
	entityCount := readiness.EntityCount
	if entityCount == 0 {
		entityCount = m.plan.Config.EntityCount
	}
	valueObjectCount := readiness.ValueObjectCount
	if valueObjectCount == 0 {
		valueObjectCount = m.plan.Config.ValueObjectCount
	}
	badge := successStyle.Render("READY")
	if m.postSaveRefreshFailed() {
		badge = dangerStyle.Render("STALE")
	} else if m.busy() {
		badge = busyStyle.Render(m.statusLabel())
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Readiness"), badge)
	fmt.Fprintf(&builder, "%s project=%d, services=%d, entities=%d, fields=%d, value objects=%d\n", labelStyle.Render("Counts"), projectCount, serviceCount, entityCount, readiness.FieldCount, valueObjectCount)
	if len(readiness.Hints) > 0 {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Next"), readiness.Hints[0])
	} else if m.plan.FileCount > 0 {
		fmt.Fprintf(&builder, "%s Review Preview before generating.\n", labelStyle.Render("Next"))
	} else {
		fmt.Fprintf(&builder, "%s Refresh the plan before continuing.\n", labelStyle.Render("Next"))
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func boolAsInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (m Model) sourceStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Overview"))
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
	fmt.Fprintln(&builder, dimStyle.Render("Project, service, entity, basic field, and basic value-object rule editing are available."))
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
	if (m.status == statusEditing && m.edit.mode == editModeServices) || (m.status == statusSaving && m.edit.mode == editModeServices) {
		var builder strings.Builder
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
		m.renderServicesEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if (m.status == statusEditing && m.edit.mode == editModeEntities) || (m.status == statusSaving && m.edit.mode == editModeEntities) {
		var builder strings.Builder
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
		m.renderEntitiesEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if (m.status == statusEditing && m.edit.mode == editModeFields) || (m.status == statusSaving && m.edit.mode == editModeFields) {
		var builder strings.Builder
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
		m.renderFieldsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	if (m.status == statusEditing && m.edit.mode == editModeValueObjects) || (m.status == statusSaving && m.edit.mode == editModeValueObjects) {
		var builder strings.Builder
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
		m.renderValueObjectsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	return m.servicesWorkspace()
}

func (m Model) servicesWorkspace() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Services workspace"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Selected service:"), m.selectedServiceSummary().Name)
	fmt.Fprintf(&builder, "%s\n", m.serviceContextTabs())

	serviceList := cardStyle.Render(m.servicesResourceList())
	detail := cardStyle.Render(m.serviceDetail())
	if m.layout == layoutWide {
		available := m.windowWidth - 30
		if available < 60 {
			available = 60
		}
		listWidth := available / 3
		detailWidth := available - listWidth - 2
		serviceList = cardStyle.Width(listWidth).Render(m.servicesResourceList())
		detail = cardStyle.Width(detailWidth).Render(m.serviceDetail())
		fmt.Fprintln(&builder, lipgloss.JoinHorizontal(lipgloss.Top, serviceList, "  ", detail))
	} else {
		fmt.Fprintln(&builder, serviceList)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, detail)
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) serviceContextTabs() string {
	labels := []string{"Services", "Entities", "Value Objects"}
	parts := make([]string, 0, len(labels))
	for index, label := range labels {
		if serviceResourceContext(index) == m.serviceContext {
			parts = append(parts, readyStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, dimStyle.Render(label))
		}
	}
	return labelStyle.Render("Context:") + " " + strings.Join(parts, "  ")
}

func (m Model) servicesResourceList() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Services"))
	services := m.serviceSummaries()
	if len(services) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No services configured."))
		return strings.TrimRight(builder.String(), "\n")
	}
	for index, service := range services {
		row := fmt.Sprintf("  %s", service.Name)
		if index == m.selectedService {
			row = fmt.Sprintf("> %s", service.Name)
		}
		if index == m.selectedService {
			row = selectedRowStyle.Render(row)
		}
		fmt.Fprintln(&builder, row)
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, dimStyle.Render("Primary resource list"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) serviceDetail() string {
	service := m.selectedServiceSummary()
	entities := m.serviceEntitySummaries()
	valueObjects := m.serviceValueObjectSummaries()
	fieldCount := 0
	for _, entity := range entities {
		fieldCount += len(entity.Fields)
	}

	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Service detail"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Service:"), service.Name)
	fmt.Fprintln(&builder, labelStyle.Render("Summary"))
	fmt.Fprintf(&builder, "  Entities: %d\n", len(entities))
	fmt.Fprintf(&builder, "  Fields: %d\n", fieldCount)
	fmt.Fprintf(&builder, "  Value objects: %d\n", len(valueObjects))
	fmt.Fprintf(&builder, "  References: %d\n", len(service.ValueObjectReferences))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, labelStyle.Render("Entities"))
	if len(entities) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("  No entities"))
	} else {
		for index, entity := range entities {
			cursor := "  "
			if m.serviceContext == serviceResourceEntities && index == m.selectedEntity {
				cursor = "> "
			}
			fmt.Fprintf(&builder, "%s%s\n", cursor, entity.Name)
		}
	}
	fmt.Fprintln(&builder, labelStyle.Render("Value Objects"))
	if len(valueObjects) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("  No value objects"))
	} else {
		for index, valueObject := range valueObjects {
			cursor := "  "
			if m.serviceContext == serviceResourceValueObjects && index == m.selectedValueObject {
				cursor = "> "
			}
			fmt.Fprintf(&builder, "%s%s\n", cursor, valueObject.Name)
		}
	}
	if len(service.ValueObjectReferences) > 0 {
		fmt.Fprintln(&builder, labelStyle.Render("References"))
		for _, reference := range service.ValueObjectReferences {
			fmt.Fprintf(&builder, "  %s <- %s.%s\n", reference.ValueObjectName, reference.EntityName, reference.FieldName)
		}
	}
	fmt.Fprintln(&builder)
	if m.busy() || m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, busyStyle.Render("Editing is paused until the current operation finishes or the stale plan is refreshed."))
	} else {
		fmt.Fprintln(&builder, successStyle.Render("Enter open selected resource/editor."))
		fmt.Fprintln(&builder, dimStyle.Render("e services | f fields | v value objects | a/r/d in editors"))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) generateStepCard() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Generate"))
	m.renderReadinessSummary(&builder)
	switch m.status {
	case statusGenerating:
		fmt.Fprintln(&builder, busyStyle.Render("Generating files. Please wait; exit is available after generation finishes."))
	case statusGenerated:
		m.renderPostGenerateSummary(&builder)
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

func (m Model) renderPostGenerateSummary(builder *strings.Builder) {
	plan := m.result.Plan
	outputDir := m.result.OutputDir
	if outputDir == "" {
		outputDir = plan.OutputDir
	}
	fmt.Fprintf(builder, "%s %d files written to %s.\n", successStyle.Render("Generated"), plan.FileCount, outputDir)
	if len(plan.Files) > 0 {
		fmt.Fprintf(builder, "%s %s\n", labelStyle.Render("Impact"), postGenerateImpactSummary(plan))
	}
	if len(plan.DeletedFiles) > 0 {
		fmt.Fprintf(builder, "%s deleted %d previous generated file(s)\n", dangerStyle.Render("Cleanup"), len(plan.DeletedFiles))
	}
	fmt.Fprintf(builder, "%s cd %s && dotnet build\n", labelStyle.Render("Next"), outputDir)
	if m.result.Warning != "" {
		fmt.Fprintf(builder, "%s %s\n", warningStyle.Render("WARNING"), warningStyle.Render(m.result.Warning))
	}
}

func (m Model) renderReadinessSummary(builder *strings.Builder) {
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(builder, dangerStyle.Render("Readiness is stale. Saved settings need a successful plan refresh before generation."))
		fmt.Fprintln(builder)
		return
	}
	readiness := m.plan.Readiness
	if m.status == statusGenerated {
		readiness = m.result.Plan.Readiness
	}
	project := "no"
	if readiness.ProjectPresent {
		project = "yes"
	}
	force := "no"
	if readiness.OutputForceRequired {
		force = "yes"
	}
	fmt.Fprintf(builder, "%s project=%s, services=%d, entities=%d, fields=%d, value objects=%d, force required=%s\n", labelStyle.Render("Readiness"), project, readiness.ServiceCount, readiness.EntityCount, readiness.FieldCount, readiness.ValueObjectCount, force)
	for _, hint := range readiness.Hints {
		fmt.Fprintf(builder, "%s %s\n", labelStyle.Render("Next"), hint)
	}
	fmt.Fprintln(builder)
}

func (m Model) footerCard() string {
	var builder strings.Builder
	if m.status == statusRefreshing {
		return busyStyle.Render("Refreshing plan. Please wait; editing, filtering, and generation are paused.")
	}
	if m.status == statusGenerating {
		return busyStyle.Render("Generating files. Please wait; exit is available after generation finishes.")
	}
	if m.status == statusSaving {
		return busyStyle.Render("Saving settings. Please wait; exit is available after save finishes.")
	}
	if m.postSaveRefreshFailed() {
		return "Locked: r retry refresh | q/esc/ctrl+c quit"
	}
	fmt.Fprintln(&builder, "Navigate: up/down select route, enter open, h/l switch, ? help.")
	switch m.activeScreen() {
	case screenProject:
		fmt.Fprintln(&builder, "Project: e edit settings, r refresh.")
	case screenServices:
		fmt.Fprintln(&builder, "Services: tab/left/right context; up/down select; enter open.")
		fmt.Fprintln(&builder, "Actions: e services; f fields; v value objects; a/r/d in editors; r refresh.")
	case screenPreview:
		fmt.Fprintln(&builder, readyHelp)
	case screenGenerate:
		if m.status == statusGenerated {
			fmt.Fprintln(&builder, generatedHelp)
		} else {
			fmt.Fprintln(&builder, "Generate: g generate, r refresh.")
		}
	default:
		fmt.Fprintln(&builder, "Overview: r refresh, 2 Project, 3 Services, 4 Preview, 5 Generate.")
	}
	fmt.Fprintln(&builder, "Back: esc | Exit: q/ctrl+c")
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
		m.renderPostGenerateSummary(&builder)
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
	fmt.Fprintln(builder, "Use the Services step for service, entity, basic field, and value-object name editing.")
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
	fmt.Fprintln(builder, "Press enter from the Services step to edit entities and their fields, or v to edit value objects.")
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
	fmt.Fprintln(builder, "Field details are name and type. Value-object validation rules are edited from the value objects editor.")
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

func (m Model) renderValueObjectsEditor(builder *strings.Builder) {
	if m.status == statusSaving {
		fmt.Fprintln(builder, "Saving value objects...")
		fmt.Fprintln(builder, "Save is in progress. Exit will be available after it finishes.")
		return
	}
	if m.valueObjectsEdit.rulesOpen {
		m.renderValueObjectRulesEditor(builder)
		return
	}
	fmt.Fprintf(builder, "Editing value objects for %s\n", m.valueObjectsEdit.serviceName)
	fmt.Fprintln(builder, "Edit value-object names or open basic type/rule editing for the selected value object.")
	fmt.Fprintln(builder, "Referenced value objects cannot be renamed or deleted, but their type and rules can be edited safely.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		fmt.Fprintln(builder, "No value objects selected for this service.")
	}
	for index, valueObject := range m.valueObjectsEdit.valueObjects {
		cursor := " "
		if index == m.valueObjectsEdit.selected {
			cursor = ">"
		}
		label := valueObject.name.string()
		if index == m.valueObjectsEdit.selected && m.valueObjectsEdit.renaming {
			label += "_"
		}
		fmt.Fprintf(builder, "%s %d. %s: %s (%s)\n", cursor, index+1, label, valueObject.typeName.string(), rulesLabelForEdit(valueObject.typeName.string(), valueObject.rules))
	}
	if service, ok := m.valueObjectEditServiceSummary(); ok && len(service.ValueObjectReferences) > 0 {
		fmt.Fprintln(builder)
		fmt.Fprintln(builder, "References:")
		for _, reference := range service.ValueObjectReferences {
			fmt.Fprintf(builder, "  %s <- %s.%s\n", reference.ValueObjectName, reference.EntityName, reference.FieldName)
		}
	}
	fmt.Fprintln(builder)
	if m.valueObjectsEdit.renaming {
		fmt.Fprintln(builder, "Rename mode: type the value object name. Enter confirms the local rename. Esc cancels value-object editing.")
		fmt.Fprintln(builder, "Left/right, backspace, and delete edit the selected value object name.")
		return
	}
	fmt.Fprintln(builder, "Keys: up/down select, a add, r rename, o rules, d delete, enter save, esc cancel.")
	fmt.Fprintln(builder, "Final validation checks blank, duplicate, colliding, and referenced value-object changes before save.")
}

func (m Model) renderValueObjectRulesEditor(builder *strings.Builder) {
	if len(m.valueObjectsEdit.valueObjects) == 0 {
		fmt.Fprintln(builder, "Editing value-object rules")
		fmt.Fprintln(builder, "No value object is selected. Press b to go back.")
		return
	}
	valueObject := m.valueObjectsEdit.valueObjects[m.valueObjectsEdit.selected]
	fmt.Fprintf(builder, "Editing rules for %s/%s\n", m.valueObjectsEdit.serviceName, valueObject.name.string())
	fmt.Fprintln(builder, "Basic rules map directly to the current JSON spec; no advanced rule DSL is available.")
	if m.err != nil {
		fmt.Fprintf(builder, "Save failed: %v\n", m.err)
	}
	for _, field := range m.visibleValueObjectRuleFields() {
		fmt.Fprintf(builder, "%s %s\n", editCursor(field == m.valueObjectsEdit.focusedRule), m.valueObjectRuleLine(valueObject, field))
	}
	fmt.Fprintln(builder)
	if m.valueObjectsEdit.editingRule {
		fmt.Fprintln(builder, "Edit mode: type text. Enter confirms the local value. Esc cancels value-object editing.")
		fmt.Fprintln(builder, "Left/right, backspace, and delete edit the selected field. Shortcut letters are inserted as text.")
		return
	}
	fmt.Fprintln(builder, "Keys: up/down select rule, e edit text, space toggle, enter save, b back, esc cancel.")
	fmt.Fprintln(builder, "Types in this editor: string, decimal, int, Guid, bool. Validation runs before save.")
}

func (m Model) valueObjectRuleLine(valueObject valueObjectEditItem, field valueObjectRuleField) string {
	suffix := ""
	if field == m.valueObjectsEdit.focusedRule && m.valueObjectsEdit.editingRule && m.selectedValueObjectRuleTextField() != nil {
		suffix = "_"
	}
	switch field {
	case valueObjectRuleFieldType:
		return "Type: " + valueObject.typeName.string() + suffix
	case valueObjectRuleFieldRequired:
		return "required: " + yesNo(valueObject.rules.required)
	case valueObjectRuleFieldMinLength:
		return "minLength: " + valueObject.rules.minLength.string() + suffix
	case valueObjectRuleFieldMaxLength:
		return "maxLength: " + valueObject.rules.maxLength.string() + suffix
	case valueObjectRuleFieldPattern:
		return "pattern: " + valueObject.rules.pattern.string() + suffix
	case valueObjectRuleFieldValidExample:
		return "validExample: " + valueObject.rules.validExample.string() + suffix
	case valueObjectRuleFieldInvalidExample:
		return "invalidExample: " + valueObject.rules.invalidExample.string() + suffix
	case valueObjectRuleFieldMinimum:
		return "minimum: " + valueObject.rules.minimum.string() + suffix
	case valueObjectRuleFieldMaximum:
		return "maximum: " + valueObject.rules.maximum.string() + suffix
	case valueObjectRuleFieldNotEmpty:
		return "notEmpty: " + yesNo(valueObject.rules.notEmpty)
	case valueObjectRuleFieldNotDefault:
		return "notDefault: " + yesNo(valueObject.rules.notDefault)
	default:
		return ""
	}
}

func rulesLabelForEdit(valueObjectType string, rules valueObjectRuleEditState) string {
	parts := []string{}
	switch valueObjectType {
	case "string":
		if rules.required {
			parts = append(parts, "required")
		}
		if rules.minLength.string() != "" {
			parts = append(parts, "min="+rules.minLength.string())
		}
		if rules.maxLength.string() != "" {
			parts = append(parts, "max="+rules.maxLength.string())
		}
		if rules.pattern.string() != "" {
			parts = append(parts, "pattern")
		}
	case "int", "decimal":
		if rules.minimum.string() != "" {
			parts = append(parts, "minimum="+rules.minimum.string())
		}
		if rules.maximum.string() != "" {
			parts = append(parts, "maximum="+rules.maximum.string())
		}
	case "Guid":
		if rules.notEmpty {
			parts = append(parts, "notEmpty")
		}
	}
	if len(parts) == 0 {
		return "no rules"
	}
	return strings.Join(parts, ", ")
}

func (m Model) valueObjectEditServiceSummary() (application.ServiceSummary, bool) {
	for _, service := range m.plan.Config.Services {
		if service.Name == m.valueObjectsEdit.serviceName {
			return service, true
		}
	}
	return application.ServiceSummary{}, false
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
	return impactSummaryForPlan(m.plan)
}

func impactSummaryForPlan(plan application.GenerationPlan) string {
	if len(plan.Files) == 0 {
		return "none"
	}
	counts := make(map[string]int)
	for _, file := range plan.Files {
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

func postGenerateImpactSummary(plan application.GenerationPlan) string {
	if len(plan.Files) == 0 {
		return "none"
	}
	counts := make(map[string]int)
	for _, file := range plan.Files {
		counts[file.Action]++
	}
	labels := []struct {
		action string
		label  string
	}{
		{action: "create", label: "created"},
		{action: "replace", label: "replaced"},
		{action: "unchanged", label: "unchanged"},
	}
	parts := []string{}
	for _, item := range labels {
		if count := counts[item.action]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", item.label, count))
			delete(counts, item.action)
		}
	}
	unknownActions := make([]string, 0, len(counts))
	for action := range counts {
		unknownActions = append(unknownActions, action)
	}
	sort.Strings(unknownActions)
	for _, action := range unknownActions {
		parts = append(parts, fmt.Sprintf("%s=%d", action, counts[action]))
	}
	return strings.Join(parts, ", ")
}
