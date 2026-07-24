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
	stepEntities
	stepValueObjects
	stepPreview
	stepGenerate
	stepResult
	stepCount
)

type workspaceScreen int

const (
	screenOverview workspaceScreen = iota
	screenProject
	screenServices
	screenEntities
	screenValueObjects
	screenPreview
	screenGenerate
	screenResult
	screenCount
)

type layoutMode int

const (
	layoutNarrow layoutMode = iota
	layoutMedium
	layoutWide
)

type tuiMode int

const (
	modeWizard tuiMode = iota
	modeWorkspace
)

type wizardScreen int

const (
	wizardMenu wizardScreen = iota
	wizardProject
	wizardServices
	wizardValueObjects
	wizardEntities
	wizardFields
	wizardReview
	wizardResult
)

const (
	wizardConfigureProject = iota
	wizardConfigureServices
	wizardReviewChanges
	wizardGenerateSolution
	wizardAdvancedWorkspace
	wizardQuit
)

const (
	wizardValueObjectConfigure = iota
	wizardValueObjectSkip
	wizardValueObjectAdvanced
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
	plan                        application.GenerationPlan
	request                     application.GenerateRequest
	planFunc                    PlanFunc
	generate                    GenerateFunc
	update                      UpdateSettingsFunc
	updateServices              UpdateServicesFunc
	updateEntities              UpdateEntitiesFunc
	updateFields                UpdateFieldsFunc
	updateValueObjects          UpdateValueObjectsFunc
	status                      modelStatus
	result                      application.GenerateResult
	err                         error
	errContext                  string
	message                     string
	edit                        editState
	servicesEdit                servicesEditState
	entitiesEdit                entitiesEditState
	fieldsEdit                  fieldsEditState
	valueObjectsEdit            valueObjectsEditState
	targetFrameworkSuggestions  []string
	fileCursor                  int
	fileOffset                  int
	windowRows                  int
	windowWidth                 int
	windowHeight                int
	layout                      layoutMode
	actionFilter                string
	currentStep                 tuiStep
	selectedService             int
	selectedEntity              int
	selectedValueObject         int
	serviceContext              serviceResourceContext
	screen                      workspaceScreen
	selectedScreen              workspaceScreen
	helpOpen                    bool
	mode                        tuiMode
	returnToWizard              bool
	guidedWorkspace             bool
	wizardScreen                wizardScreen
	wizardBackScreen            wizardScreen
	wizardSelection             int
	wizardProjectSelection      int
	wizardServiceSelection      int
	wizardEntitySelection       int
	wizardFieldSelection        int
	wizardResultSelection       int
	wizardValueObjectSelection  int
	wizardValueObjectConfigured bool
}

var runTeaProgram = func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	return tea.NewProgram(model, options...).Run()
}

func NewModel(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, targetFrameworkSuggestions ...[]string) Model {
	suggestions := []string(nil)
	if len(targetFrameworkSuggestions) > 0 {
		suggestions = append([]string(nil), targetFrameworkSuggestions[0]...)
	}
	return Model{plan: plan, request: request, planFunc: planFunc, generate: generate, update: update, status: statusReady, targetFrameworkSuggestions: suggestions, windowRows: defaultFileWindowRows, layout: layoutModeForWidth(0), currentStep: stepSource, screen: screenOverview, selectedScreen: screenOverview, mode: modeWizard, wizardScreen: wizardMenu}
}

func Run(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, update UpdateSettingsFunc, updateServices UpdateServicesFunc, updateEntities UpdateEntitiesFunc, updateFields UpdateFieldsFunc, updateValueObjects UpdateValueObjectsFunc, targetFrameworkSuggestions []string) error {
	model := NewModel(plan, request, planFunc, generate, update, targetFrameworkSuggestions)
	model.updateServices = updateServices
	model.updateEntities = updateEntities
	model.updateFields = updateFields
	model.updateValueObjects = updateValueObjects
	_, err := runTeaProgram(model, tea.WithAltScreen())
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
		if m.mode == modeWizard {
			return m.updateWizard(msg)
		}
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
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				if m.returnToWizard && m.guidedWorkspace {
					m.enterGuidedWizardBack()
					return m, nil
				}
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
			if key == "esc" {
				switch m.activeScreen() {
				case screenEntities:
					m.openScreen(screenServices)
					m.serviceContext = serviceResourceEntities
					return m, nil
				case screenValueObjects:
					m.openScreen(screenServices)
					m.serviceContext = serviceResourceValueObjects
					return m, nil
				case screenResult:
					if m.returnToWizard && m.guidedWorkspace {
						m.enterGuidedWizardBack()
					} else {
						m.openScreen(screenGenerate)
					}
					return m, nil
				case screenServices, screenProject, screenPreview, screenGenerate:
					if m.returnToWizard && m.guidedWorkspace {
						m.enterGuidedWizardBack()
					} else {
						m.openScreen(screenOverview)
					}
					return m, nil
				case screenOverview:
					if m.returnToWizard {
						m.enterWizardMenu()
						return m, nil
					}
				}
			}
			return m, tea.Quit
		case "?":
			m.helpOpen = true
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8":
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
			if m.activeScreen() == screenEntities {
				m.selectedEntity -= 1
				m.clampServiceResourceSelection()
				return m, nil
			}
			if m.activeScreen() == screenValueObjects {
				m.selectedValueObject -= 1
				m.clampServiceResourceSelection()
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
			if m.activeScreen() == screenEntities {
				m.selectedEntity += 1
				m.clampServiceResourceSelection()
				return m, nil
			}
			if m.activeScreen() == screenValueObjects {
				m.selectedValueObject += 1
				m.clampServiceResourceSelection()
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
			if m.activeScreen() == screenPreview {
				m.openScreen(screenGenerate)
				return m, nil
			}
			if m.generationBlocked() {
				m.openScreen(screenGenerate)
				m.message = m.generationBlockMessage()
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
			} else if m.activeScreen() == screenEntities {
				m.startEntitiesEditing()
			} else if m.activeScreen() == screenValueObjects {
				m.startValueObjectsEditing()
			} else {
				m.openScreen(screenProject)
				m.startEditing()
			}
			return m, nil
		case "enter":
			if m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving {
				return m, nil
			}
			if m.guidedWorkspace && m.activeScreen() == screenGenerate {
				if m.generationBlocked() {
					m.message = m.generationBlockMessage()
					return m, nil
				}
				m.status = statusGenerating
				m.err = nil
				m.errContext = ""
				m.message = ""
				return m, m.generateCmd()
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
			if m.activeScreen() == screenEntities {
				m.startEntitiesEditing()
				return m, nil
			}
			if m.activeScreen() == screenValueObjects {
				m.startValueObjectsEditing()
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
		case "f":
			if m.activeScreen() == screenEntities {
				m.startEntitiesEditing()
				m.startFieldsEditing()
			}
			return m, nil
		case "o":
			if m.activeScreen() == screenValueObjects {
				m.startValueObjectsEditing()
				if len(m.valueObjectsEdit.valueObjects) > 0 {
					m.valueObjectsEdit.rulesOpen = true
					m.valueObjectsEdit.focusedRule = valueObjectRuleFieldType
				}
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
			if m.mode != modeWizard {
				m.openScreen(screenGenerate)
			}
			return m, nil
		}
		wasResult := m.activeScreen() == screenResult
		m.status = statusReady
		m.plan = msg.plan
		m.result = application.GenerateResult{}
		m.err = nil
		m.errContext = ""
		m.message = ""
		if wasResult {
			m.openScreen(screenPreview)
		}
		m.clampSelectedService()
		m.clampFileCursor()
		if m.mode == modeWizard && m.wizardScreen == wizardProject {
			m.enterWizardServices()
			m.wizardBackScreen = wizardProject
		} else if m.mode == modeWizard && m.wizardScreen == wizardServices {
			m.wizardServiceSelection = clampInt(m.wizardServiceSelection, 0, m.wizardServiceOptionCount()-1)
		}
		return m, nil

	case generationFinishedMsg:
		if msg.err != nil {
			m.status = statusFailed
			m.err = msg.err
			m.errContext = "Generation"
			if m.returnToWizard && m.guidedWorkspace {
				m.mode = modeWizard
				m.wizardScreen = wizardResult
			} else {
				m.openScreen(screenResult)
			}
			return m, nil
		}
		m.status = statusGenerated
		m.result = msg.result
		m.plan = msg.result.Plan
		if m.returnToWizard && m.guidedWorkspace {
			m.mode = modeWizard
			m.wizardScreen = wizardResult
			m.returnToWizard = false
		} else {
			m.openScreen(screenResult)
		}
		m.err = nil
		m.errContext = ""
		m.message = ""
		m.clampSelectedService()
		m.clampFileCursor()
		if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		}
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
		if m.mode == modeWizard && m.wizardScreen == wizardProject {
			m.enterWizardServices()
			m.wizardBackScreen = wizardProject
		} else if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		}
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
		m.message = "Services saved. Plan refreshed. Configure value objects before entities and fields."
		m.clampSelectedService()
		m.clampFileCursor()
		if m.mode == modeWizard && m.wizardScreen == wizardServices {
			m.wizardServiceSelection = clampInt(m.wizardServiceSelection, 0, m.wizardServiceOptionCount()-1)
		} else if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		}
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
		m.message = "Entities saved. Plan refreshed. Press f in the Entities editor to edit fields."
		m.clampSelectedService()
		m.clampFileCursor()
		if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		}
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
		m.message = "Fields saved. Plan refreshed. Review the generation plan."
		m.clampSelectedService()
		m.clampFileCursor()
		if m.mode == modeWizard && m.wizardScreen == wizardFields {
			m.enterWizardReview()
		}
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
		m.message = "Value objects saved. Plan refreshed. Continue with entities and fields."
		m.clampSelectedService()
		m.clampFileCursor()
		if m.mode == modeWizard && m.wizardScreen == wizardValueObjects {
			m.enterWizardEntities()
		} else if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateWizard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.postSaveRefreshFailed() {
		switch key {
		case "r":
			m.status = statusRefreshing
			m.err = nil
			m.message = ""
			return m, m.planCmd()
		case "esc":
			switch m.wizardScreen {
			case wizardReview:
				m.enterWizardFields()
				return m, nil
			case wizardValueObjects:
				m.enterWizardServices()
				return m, nil
			case wizardFields:
				m.enterWizardEntities()
				return m, nil
			case wizardEntities:
				m.enterWizardValueObjects()
				return m, nil
			case wizardServices:
				if m.wizardBackScreen == wizardProject {
					m.enterWizardProject()
				} else {
					m.enterWizardMenu()
				}
				return m, nil
			case wizardProject:
				m.enterWizardMenu()
				return m, nil
			}
			m.enterWizardMenu()
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}
	}
	if m.status == statusSaving {
		return m, nil
	}
	if m.status == statusEditing {
		switch m.edit.mode {
		case editModeServices:
			return m.updateServicesEdit(msg)
		case editModeEntities:
			return m.updateEntitiesEdit(msg)
		case editModeFields:
			return m.updateFieldsEdit(msg)
		case editModeValueObjects:
			return m.updateValueObjectsEdit(msg)
		}
		return m.updateEdit(msg)
	}

	if m.wizardScreen == wizardProject {
		if msg.String() == "up" || msg.String() == "k" {
			m.wizardProjectSelection = clampInt(m.wizardProjectSelection-1, 0, m.wizardProjectOptionCount()-1)
			return m, nil
		}
		if msg.String() == "down" || msg.String() == "j" {
			m.wizardProjectSelection = clampInt(m.wizardProjectSelection+1, 0, m.wizardProjectOptionCount()-1)
			return m, nil
		}
		if key == "enter" {
			return m.selectWizardProjectOption()
		}
		switch key {
		case "esc":
			m.enterWizardMenu()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardServices {
		switch key {
		case "up", "k":
			m.wizardServiceSelection = clampInt(m.wizardServiceSelection-1, 0, m.wizardServiceOptionCount()-1)
		case "down", "j":
			m.wizardServiceSelection = clampInt(m.wizardServiceSelection+1, 0, m.wizardServiceOptionCount()-1)
		case "enter":
			return m.selectWizardService()
		case "esc":
			if m.wizardBackScreen == wizardProject {
				m.enterWizardProject()
			} else {
				m.enterWizardMenu()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardEntities {
		switch key {
		case "up", "k":
			m.wizardEntitySelection = clampInt(m.wizardEntitySelection-1, 0, m.wizardEntityOptionCount()-1)
			m.selectedEntity = clampInt(m.wizardEntitySelection, 0, len(m.serviceEntitySummaries())-1)
		case "down", "j":
			m.wizardEntitySelection = clampInt(m.wizardEntitySelection+1, 0, m.wizardEntityOptionCount()-1)
			if m.wizardEntitySelection < len(m.serviceEntitySummaries()) {
				m.selectedEntity = m.wizardEntitySelection
			}
		case "enter":
			return m.selectWizardEntity()
		case "esc":
			m.enterWizardValueObjects()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardFields {
		switch key {
		case "up", "k":
			m.wizardFieldSelection = clampInt(m.wizardFieldSelection-1, 0, m.wizardFieldOptionCount()-1)
		case "down", "j":
			m.wizardFieldSelection = clampInt(m.wizardFieldSelection+1, 0, m.wizardFieldOptionCount()-1)
		case "enter":
			return m.selectWizardField()
		case "esc":
			m.enterWizardEntities()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardValueObjects {
		if !m.wizardValueObjectConfigured {
			switch key {
			case "up", "k":
				m.wizardValueObjectSelection = clampInt(m.wizardValueObjectSelection-1, 0, wizardValueObjectAdvanced)
			case "down", "j":
				m.wizardValueObjectSelection = clampInt(m.wizardValueObjectSelection+1, 0, wizardValueObjectAdvanced)
			case "enter":
				return m.selectWizardValueObjectOption()
			case "esc":
				m.enterWizardServices()
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		switch key {
		case "up", "k":
			m.wizardValueObjectSelection = clampInt(m.wizardValueObjectSelection-1, 0, m.wizardValueObjectOptionCount()-1)
		case "down", "j":
			m.wizardValueObjectSelection = clampInt(m.wizardValueObjectSelection+1, 0, m.wizardValueObjectOptionCount()-1)
		case "enter":
			return m.selectWizardValueObject()
		case "esc":
			m.enterWizardServices()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardReview {
		switch key {
		case "up", "k":
			m.wizardSelection = clampInt(m.wizardSelection-1, 0, 2)
		case "down", "j":
			m.wizardSelection = clampInt(m.wizardSelection+1, 0, 2)
		case "enter":
			switch m.wizardSelection {
			case 0:
				m.enterWizardWorkspace(screenGenerate)
			case 1:
				m.enterWizardWorkspace(screenPreview)
			default:
				m.enterWizardFields()
			}
		case "esc":
			m.enterWizardFields()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.wizardScreen == wizardResult {
		switch key {
		case "up", "k":
			m.wizardResultSelection = clampInt(m.wizardResultSelection-1, 0, 1)
		case "down", "j":
			m.wizardResultSelection = clampInt(m.wizardResultSelection+1, 0, 1)
		case "enter":
			if m.wizardResultSelection == 0 {
				m.enterWizardMenu()
			} else {
				m.enterAdvancedWorkspace()
			}
		case "esc":
			m.enterWizardMenu()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	switch key {
	case "up", "k":
		m.wizardSelection = clampInt(m.wizardSelection-1, 0, wizardQuit)
	case "down", "j":
		m.wizardSelection = clampInt(m.wizardSelection+1, 0, wizardQuit)
	case "enter":
		return m.selectWizardOption()
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) selectWizardOption() (tea.Model, tea.Cmd) {
	switch m.wizardSelection {
	case wizardConfigureProject:
		m.enterWizardProject()
	case wizardConfigureServices:
		m.enterWizardServices()
	case wizardReviewChanges:
		m.enterWizardReview()
	case wizardGenerateSolution:
		m.enterWizardWorkspace(screenGenerate)
	case wizardAdvancedWorkspace:
		m.enterAdvancedWorkspace()
	case wizardQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) selectWizardProjectOption() (tea.Model, tea.Cmd) {
	switch {
	case m.wizardProjectSelection < len(m.targetFrameworkSuggestions):
		m.edit.targetFramework = newTextField(m.targetFrameworkSuggestions[m.wizardProjectSelection])
		m.message = fmt.Sprintf("Draft target framework set to %s. Continue to services to save.", m.edit.targetFramework.string())
	case m.wizardProjectSelection == m.wizardProjectEditOption():
		name, description, targetFramework := m.edit.name, m.edit.description, m.edit.targetFramework
		m.startEditing()
		m.edit.name = name
		m.edit.description = description
		m.edit.targetFramework = targetFramework
	case m.wizardProjectSelection == m.wizardProjectContinueOption():
		m.status = statusSaving
		m.err = nil
		m.errContext = ""
		m.message = ""
		return m, m.saveSettingsCmd()
	}
	return m, nil
}

func (m Model) selectWizardService() (tea.Model, tea.Cmd) {
	serviceCount := len(m.serviceSummaries())
	switch {
	case m.wizardServiceSelection < serviceCount:
		m.selectedService = m.wizardServiceSelection
		m.clampSelectedService()
		m.enterWizardValueObjects()
	case m.wizardServiceSelection == m.wizardServiceAddOption():
		m.startServicesEditing()
		m.servicesEdit.services = append(m.servicesEdit.services, newTextField(m.nextServicePlaceholder()))
		m.servicesEdit.original = append(m.servicesEdit.original, "")
		m.servicesEdit.selected = len(m.servicesEdit.services) - 1
		m.servicesEdit.renaming = true
	case m.wizardServiceSelection == m.wizardServiceEditOption():
		m.startServicesEditing()
	case m.wizardServiceSelection == m.wizardServiceAdvancedOption():
		m.enterAdvancedWorkspace()
		m.openScreen(screenServices)
	}
	return m, nil
}

func (m Model) selectWizardEntity() (tea.Model, tea.Cmd) {
	entityCount := len(m.serviceEntitySummaries())
	switch {
	case m.wizardEntitySelection < entityCount:
		m.selectedEntity = m.wizardEntitySelection
		m.enterWizardFields()
	case m.wizardEntitySelection == m.wizardEntityAddOption():
		m.startEntitiesEditing()
		m.entitiesEdit.entities = append(m.entitiesEdit.entities, newTextField(m.nextEntityPlaceholder()))
		m.entitiesEdit.original = append(m.entitiesEdit.original, "")
		m.entitiesEdit.selected = len(m.entitiesEdit.entities) - 1
		m.selectedEntity = m.entitiesEdit.selected
		m.entitiesEdit.renaming = true
	case m.wizardEntitySelection == m.wizardEntityEditOption():
		m.startEntitiesEditing()
	case m.wizardEntitySelection == m.wizardEntityAdvancedOption():
		m.enterAdvancedWorkspace()
		m.openScreen(screenEntities)
	}
	return m, nil
}

func (m Model) selectWizardField() (tea.Model, tea.Cmd) {
	fieldCount := len(m.selectedEntitySummary().Fields)
	switch {
	case m.wizardFieldSelection < fieldCount:
		m.startWizardFieldsEditing()
		m.fieldsEdit.selected = m.wizardFieldSelection
	case m.wizardFieldSelection == m.wizardFieldContinueOption():
		m.enterWizardReview()
	case m.wizardFieldSelection == m.wizardFieldAddOption():
		m.startWizardFieldsEditing()
		m.fieldsEdit.fields = append(m.fieldsEdit.fields, fieldEditItem{name: newTextField(m.nextFieldPlaceholder()), typeName: newTextField("string")})
		m.fieldsEdit.selected = len(m.fieldsEdit.fields) - 1
	case m.wizardFieldSelection == m.wizardFieldEditOption():
		m.startWizardFieldsEditing()
	case m.wizardFieldSelection == m.wizardFieldAdvancedOption():
		m.enterAdvancedWorkspace()
		m.openScreen(screenEntities)
	}
	return m, nil
}

func (m Model) selectWizardValueObjectOption() (tea.Model, tea.Cmd) {
	switch m.wizardValueObjectSelection {
	case wizardValueObjectConfigure:
		m.wizardValueObjectConfigured = true
		m.wizardValueObjectSelection = 0
	case wizardValueObjectSkip:
		m.enterWizardEntities()
	case wizardValueObjectAdvanced:
		m.enterAdvancedWorkspace()
		m.openScreen(screenValueObjects)
	}
	return m, nil
}

func (m Model) selectWizardValueObject() (tea.Model, tea.Cmd) {
	valueObjectCount := len(m.serviceValueObjectSummaries())
	switch {
	case m.wizardValueObjectSelection < valueObjectCount:
		m.selectedValueObject = m.wizardValueObjectSelection
		m.startValueObjectsEditing()
	case m.wizardValueObjectSelection == m.wizardValueObjectAddOption():
		m.startValueObjectsEditing()
		if valueObjectCount > 0 {
			m.valueObjectsEdit.valueObjects = append(m.valueObjectsEdit.valueObjects, newValueObjectEditItem("", m.nextValueObjectPlaceholder()))
		}
		m.valueObjectsEdit.selected = len(m.valueObjectsEdit.valueObjects) - 1
		m.valueObjectsEdit.renaming = true
	case m.wizardValueObjectSelection == m.wizardValueObjectEditOption():
		m.startValueObjectsEditing()
	case m.wizardValueObjectSelection == m.wizardValueObjectReviewOption():
		m.enterWizardEntities()
	case m.wizardValueObjectSelection == m.wizardValueObjectAdvancedOption():
		m.enterAdvancedWorkspace()
		m.openScreen(screenValueObjects)
	}
	return m, nil
}

func (m *Model) enterWizardMenu() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardMenu
	m.wizardBackScreen = wizardMenu
	m.helpOpen = false
}

func (m *Model) enterWizardProject() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardProject
	m.wizardBackScreen = wizardMenu
	m.helpOpen = false
	m.initializeProjectDraft()
	m.wizardProjectSelection = m.wizardProjectCurrentSelection()
	if !m.postSaveRefreshFailed() {
		m.err = nil
		m.errContext = ""
		m.message = ""
	}
}

func (m *Model) enterWizardServices() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardServices
	m.wizardBackScreen = wizardMenu
	m.wizardServiceSelection = clampInt(m.selectedService, 0, m.wizardServiceOptionCount()-1)
	m.helpOpen = false
}

func (m *Model) enterWizardEntities() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardEntities
	m.wizardBackScreen = wizardValueObjects
	m.clampSelectedService()
	entities := m.serviceEntitySummaries()
	m.wizardEntitySelection = clampInt(m.selectedEntity, 0, len(entities)+2)
	if len(entities) > 0 {
		m.selectedEntity = clampInt(m.wizardEntitySelection, 0, len(entities)-1)
	}
	m.serviceContext = serviceResourceEntities
	m.helpOpen = false
}

func (m *Model) enterWizardFields() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardFields
	m.wizardBackScreen = wizardEntities
	m.entitiesEdit.serviceName = m.selectedServiceSummary().Name
	m.entitiesEdit.returnStatus = m.status
	m.selectedEntity = clampInt(m.selectedEntity, 0, len(m.serviceEntitySummaries())-1)
	m.wizardFieldSelection = 0
	m.serviceContext = serviceResourceEntities
	m.helpOpen = false
}

func (m *Model) enterWizardValueObjects() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardValueObjects
	m.wizardBackScreen = wizardServices
	m.wizardValueObjectConfigured = false
	m.wizardValueObjectSelection = 0
	m.clampSelectedService()
	m.serviceContext = serviceResourceValueObjects
	m.helpOpen = false
}

func (m *Model) enterWizardReview() {
	m.mode = modeWizard
	m.returnToWizard = false
	m.guidedWorkspace = false
	m.wizardScreen = wizardReview
	m.wizardBackScreen = wizardFields
	m.wizardSelection = 0
	m.helpOpen = false
}

func (m *Model) startWizardFieldsEditing() {
	m.status = statusEditing
	m.entitiesEdit.returnStatus = statusReady
	m.entitiesEdit.serviceName = m.selectedServiceSummary().Name
	m.startFieldsEditing()
}

func (m Model) wizardServiceAddOption() int {
	return len(m.serviceSummaries())
}

func (m Model) wizardServiceEditOption() int {
	return m.wizardServiceAddOption() + 1
}

func (m Model) wizardServiceAdvancedOption() int {
	return m.wizardServiceAddOption() + 2
}

func (m Model) wizardServiceOptionCount() int {
	return m.wizardServiceAdvancedOption() + 1
}

func (m Model) wizardProjectEditOption() int {
	return len(m.targetFrameworkSuggestions)
}

func (m Model) wizardProjectContinueOption() int {
	return m.wizardProjectEditOption() + 1
}

func (m Model) wizardProjectOptionCount() int {
	return m.wizardProjectContinueOption() + 1
}

func (m Model) wizardProjectCurrentSelection() int {
	current := m.edit.targetFramework.string()
	for index, suggestion := range m.targetFrameworkSuggestions {
		if suggestion == current {
			return index
		}
	}
	return m.wizardProjectEditOption()
}

func (m Model) wizardEntityAddOption() int {
	return len(m.serviceEntitySummaries())
}

func (m Model) wizardEntityEditOption() int {
	return m.wizardEntityAddOption() + 1
}

func (m Model) wizardEntityAdvancedOption() int {
	return m.wizardEntityAddOption() + 2
}

func (m Model) wizardEntityOptionCount() int {
	return m.wizardEntityAdvancedOption() + 1
}

func (m Model) wizardFieldAddOption() int {
	return m.wizardFieldContinueOption() + 1
}

func (m Model) wizardFieldContinueOption() int {
	return len(m.selectedEntitySummary().Fields)
}

func (m Model) wizardFieldEditOption() int {
	return m.wizardFieldAddOption() + 1
}

func (m Model) wizardFieldAdvancedOption() int {
	return m.wizardFieldAddOption() + 2
}

func (m Model) wizardFieldOptionCount() int {
	return m.wizardFieldAdvancedOption() + 1
}

func (m Model) wizardValueObjectAddOption() int {
	return len(m.serviceValueObjectSummaries())
}

func (m Model) wizardValueObjectEditOption() int {
	return m.wizardValueObjectAddOption() + 1
}

func (m Model) wizardValueObjectReviewOption() int {
	return m.wizardValueObjectEditOption() + 1
}

func (m Model) wizardValueObjectAdvancedOption() int {
	return m.wizardValueObjectReviewOption() + 1
}

func (m Model) wizardValueObjectOptionCount() int {
	return m.wizardValueObjectAdvancedOption() + 1
}

func (m *Model) enterWizardWorkspace(screen workspaceScreen) {
	m.wizardBackScreen = m.wizardScreen
	m.mode = modeWorkspace
	m.returnToWizard = true
	m.guidedWorkspace = true
	m.openScreen(screen)
}

func (m *Model) enterGuidedWizardBack() {
	switch m.wizardBackScreen {
	case wizardReview:
		m.enterWizardReview()
	case wizardValueObjects:
		m.enterWizardValueObjects()
	default:
		m.enterWizardMenu()
	}
}

func (m *Model) enterWizardServicesWorkspace() {
	m.wizardBackScreen = m.wizardScreen
	m.mode = modeWorkspace
	m.returnToWizard = true
	m.guidedWorkspace = true
	m.screen = screenServices
	m.selectedScreen = screenServices
	m.currentStep = stepForScreen(screenServices)
}

func (m *Model) enterAdvancedWorkspace() {
	m.mode = modeWorkspace
	m.returnToWizard = true
	m.guidedWorkspace = false
	m.openScreen(screenOverview)
}

func (m Model) busy() bool {
	return m.status == statusRefreshing || m.status == statusGenerating || m.status == statusSaving
}

func (m Model) generationBlocked() bool {
	return m.postSaveRefreshFailed() || ((m.plan.ForceRequired || m.plan.Readiness.OutputForceRequired) && !m.request.Force)
}

func (m Model) generationBlockMessage() string {
	if m.postSaveRefreshFailed() {
		return "Generation is locked until the plan refresh succeeds."
	}
	return "Generation is locked until --force is confirmed for this existing output."
}

func (m *Model) moveStep(delta int) {
	next := int(m.currentStep) + delta
	if next < 0 {
		next = 0
	}
	if next >= int(stepCount) {
		next = int(stepCount) - 1
	}
	m.openScreen(screenForStep(tuiStep(next)))
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
	case stepEntities:
		return screenEntities
	case stepValueObjects:
		return screenValueObjects
	case stepPreview:
		return screenPreview
	case stepGenerate:
		return screenGenerate
	case stepResult:
		return screenResult
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
	case screenEntities:
		return stepEntities
	case screenValueObjects:
		return stepValueObjects
	case screenPreview:
		return stepPreview
	case screenGenerate:
		return stepGenerate
	case screenResult:
		return stepResult
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
	case screenEntities:
		return "Entities"
	case screenValueObjects:
		return "Value Objects"
	case screenPreview:
		return "Preview"
	case screenGenerate:
		return "Generate"
	case screenResult:
		return "Result"
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
	switch screen {
	case screenServices:
		m.serviceContext = serviceResourceServices
	case screenEntities:
		m.serviceContext = serviceResourceEntities
	case screenValueObjects:
		m.serviceContext = serviceResourceValueObjects
	}
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
	case stepResult:
		return "Result"
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
	m.initializeProjectDraft()
	m.edit.returnStatus = returnStatus
}

func (m *Model) initializeProjectDraft() {
	m.edit = editState{
		mode:            editModeProject,
		name:            newTextField(m.plan.Config.SolutionName),
		description:     newTextField(m.plan.Config.SolutionDescription),
		targetFramework: newTextField(m.plan.Config.TargetFramework),
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
	m.openScreen(screenEntities)
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
	if m.mode == modeWizard {
		m.wizardScreen = wizardFields
		m.wizardBackScreen = wizardEntities
	}
}

func (m *Model) startValueObjectsEditing() {
	m.openScreen(screenValueObjects)
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
		m.openScreen(screenServices)
		m.serviceContext = serviceResourceEntities
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
		if m.mode == modeWizard && m.wizardScreen == wizardValueObjects {
			m.enterWizardServices()
			return m, nil
		}
		m.openScreen(screenServices)
		m.serviceContext = serviceResourceValueObjects
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
		m.valueObjectsEdit.rulesOpen = false
		m.valueObjectsEdit.editingRule = false
		if m.mode == modeWizard && m.wizardScreen == wizardValueObjects {
			m.enterWizardServices()
			return m, nil
		}
		m.openScreen(screenValueObjects)
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
		if m.mode == modeWizard && m.wizardScreen == wizardProject {
			m.enterWizardMenu()
		} else if m.returnToWizard && m.guidedWorkspace {
			m.enterWizardMenu()
		} else {
			m.openScreen(screenProject)
		}
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
	if m.mode == modeWizard {
		return m.wizardView()
	}
	if m.guidedWorkspace {
		return m.guidedWorkspaceView()
	}
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

func (m Model) wizardView() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s - %s\n", appTitleStyle.Render("Microgen"), m.statusBadge())
	if m.wizardScreen == wizardResult {
		fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Wizard / Review > Generate > Result  |  Progress: complete"))
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Generation complete"))
		if m.status == statusFailed {
			outputDir := m.outputDirectory()
			fmt.Fprintf(&builder, "%s %s failed: %v\n", dangerStyle.Render("FAILED"), m.errContext, m.err)
			fmt.Fprintf(&builder, "Output directory: %s\n", outputDir)
			fmt.Fprintln(&builder, "Impact: no generated files were published.")
			fmt.Fprintln(&builder, dangerStyle.Render("No generated result was published."))
			fmt.Fprintln(&builder, "Build/test: unavailable until generation succeeds.")
			fmt.Fprintln(&builder, "Check the error, then return to Generate to retry safely.")
		} else {
			outputDir := m.result.OutputDir
			if outputDir == "" {
				outputDir = m.outputDirectory()
			}
			fmt.Fprintf(&builder, "%s %d files written to %s.\n", successStyle.Render("Generated"), m.result.Plan.FileCount, outputDir)
			fmt.Fprintf(&builder, "Output directory: %s\n", outputDir)
			fmt.Fprintf(&builder, "Files: %d | Impact: %s | deleted=%d\n", m.result.Plan.FileCount, postGenerateImpactSummary(m.result.Plan), len(m.result.Plan.DeletedFiles))
			fmt.Fprintf(&builder, "Build/test: cd %s && dotnet build && dotnet test\n", outputDir)
			if m.result.Warning != "" {
				fmt.Fprintf(&builder, "%s %s\n", warningStyle.Render("Warning:"), m.result.Warning)
			}
		}
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardResultOption(0, "Back to menu"))
		fmt.Fprintln(&builder, m.wizardResultOption(1, "Advanced workspace"))
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, dimStyle.Render("Footer: up/down or j/k move | enter select | esc menu | q quit"))
		return strings.TrimRight(builder.String(), "\n")
	}
	if m.wizardScreen == wizardProject {
		return m.wizardProjectView()
	}
	if m.wizardScreen == wizardServices {
		return m.wizardServicesView()
	}
	if m.wizardScreen == wizardEntities {
		return m.wizardEntitiesView()
	}
	if m.wizardScreen == wizardFields {
		return m.wizardFieldsView()
	}
	if m.wizardScreen == wizardValueObjects {
		return m.wizardValueObjectsView()
	}
	if m.wizardScreen == wizardReview {
		return m.wizardReviewView()
	}

	fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Wizard / Menu  |  Progress: 0/6"))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("What would you like to configure?"))
	fmt.Fprintln(&builder)
	for option := 0; option <= wizardQuit; option++ {
		fmt.Fprintln(&builder, m.wizardOption(option, wizardOptionLabel(option)))
	}
	if m.message != "" {
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, dimStyle.Render("Footer: up/down or j/k move | enter select | q quit"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardProjectView() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Wizard / Project > Services  |  Progress: 1/6"))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Set up your project"))
	fmt.Fprintln(&builder)
	if m.status == statusEditing || m.status == statusSaving {
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Edit solution settings"))
		m.renderSettingsEditor(&builder)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("tab/down or shift+tab/up move fields | enter save and continue | esc back to menu"))
		return strings.TrimRight(builder.String(), "\n")
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Solution name:"), truncateWizardText(m.edit.name.string(), m.wizardContentWidth()-16))
	fmt.Fprintf(&builder, "%s %s\n", sectionTitleStyle.Render("Target framework:"), m.edit.targetFramework.string())
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Available target frameworks"))
	if len(m.targetFrameworkSuggestions) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No framework suggestions available."))
	} else {
		for option := 0; option < len(m.targetFrameworkSuggestions); option++ {
			fmt.Fprintln(&builder, m.wizardProjectOption(option))
		}
	}
	if m.wizardProjectCurrentSelection() == m.wizardProjectEditOption() {
		fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Current framework:"), m.edit.targetFramework.string())
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardProjectOption(m.wizardProjectEditOption()))
	fmt.Fprintln(&builder, m.wizardProjectOption(m.wizardProjectContinueOption()))
	if m.err != nil {
		fmt.Fprintf(&builder, "%s %v\n", dangerStyle.Render("Save failed:"), m.err)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, dangerStyle.Render("The saved project has a stale plan."))
		fmt.Fprintln(&builder, "Press r to retry the refresh before continuing.")
	}
	if m.message != "" {
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	footer := "up/down or j/k select | enter choose/open/save | esc back to menu"
	if m.postSaveRefreshFailed() {
		footer += " | r retry refresh"
	}
	fmt.Fprintln(&builder, m.wizardFooter(footer))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardServicesView() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Wizard / Project / Services  |  Progress: 2/6"))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Which service should we configure?"))
	fmt.Fprintln(&builder)
	if m.status == statusEditing || m.status == statusSaving {
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Edit services"))
		m.renderServicesEditor(&builder)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter confirm/save | esc cancel"))
		return strings.TrimRight(builder.String(), "\n")
	}
	for option := 0; option < m.wizardServiceOptionCount(); option++ {
		fmt.Fprintln(&builder, m.wizardServiceOption(option))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardServiceDetail())
	if m.err != nil {
		fmt.Fprintf(&builder, "%s %v\n", dangerStyle.Render("Operation failed:"), m.err)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, dangerStyle.Render("The saved services have a stale plan."))
		fmt.Fprintln(&builder, "Press r to retry the refresh before continuing.")
	}
	if m.message != "" {
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	footer := "up/down or j/k select | enter configure value objects | esc back"
	if m.postSaveRefreshFailed() {
		footer += " | r retry refresh"
	}
	fmt.Fprintln(&builder, m.wizardFooter(footer))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardEntitiesView() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Project > Services > Value Objects > Entities  |  Progress: 4/6"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Selected service:"), truncateWizardText(m.selectedServiceSummary().Name, m.wizardContentWidth()-18))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Which entity should we configure?"))
	fmt.Fprintln(&builder)
	if (m.status == statusEditing || m.status == statusSaving) && m.edit.mode == editModeEntities {
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Edit entities"))
		m.renderEntitiesEditor(&builder)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter confirm/save | esc back"))
		return strings.TrimRight(builder.String(), "\n")
	}
	for option := 0; option < m.wizardEntityOptionCount(); option++ {
		fmt.Fprintln(&builder, m.wizardEntityOption(option))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardEntityDetail())
	if m.err != nil {
		fmt.Fprintf(&builder, "%s %v\n", dangerStyle.Render("Operation failed:"), m.err)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, dangerStyle.Render("The saved entities have a stale plan."))
		fmt.Fprintln(&builder, "Press r to retry the refresh before continuing.")
	}
	if m.message != "" {
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	footer := "up/down or j/k select | enter open | esc back to value objects"
	if m.postSaveRefreshFailed() {
		footer += " | r retry refresh"
	}
	fmt.Fprintln(&builder, m.wizardFooter(footer))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardFieldsView() string {
	var builder strings.Builder
	service := truncateWizardText(m.selectedServiceSummary().Name, m.wizardContentWidth()-10)
	entity := truncateWizardText(m.selectedEntitySummary().Name, m.wizardContentWidth()-10)
	fmt.Fprintf(&builder, "%s\n", dimStyle.Render(fmt.Sprintf("Breadcrumb: Project > Services > Value Objects > Entities > Fields  |  Progress: 5/6  |  %s/%s", service, entity)))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Which field should we configure?"))
	fmt.Fprintln(&builder)
	if (m.status == statusEditing || m.status == statusSaving) && m.edit.mode == editModeFields {
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Edit fields"))
		m.renderFieldsEditor(&builder)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter confirm/save | esc back"))
		return strings.TrimRight(builder.String(), "\n")
	}
	for option := 0; option < m.wizardFieldOptionCount(); option++ {
		fmt.Fprintln(&builder, m.wizardFieldOption(option))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardFieldDetail())
	if m.err != nil {
		fmt.Fprintf(&builder, "%s %v\n", dangerStyle.Render("Operation failed:"), m.err)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, dangerStyle.Render("The saved fields have a stale plan."))
		fmt.Fprintln(&builder, "Press r to retry the refresh before continuing.")
	}
	if m.message != "" {
		fmt.Fprintln(&builder, successStyle.Render(m.message))
	}
	fmt.Fprintln(&builder)
	footer := "up/down or j/k select | enter open | esc back to entities"
	if m.postSaveRefreshFailed() {
		footer += " | r retry refresh"
	}
	fmt.Fprintln(&builder, m.wizardFooter(footer))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardValueObjectsView() string {
	var builder strings.Builder
	service := truncateWizardText(m.selectedServiceSummary().Name, m.wizardContentWidth()-10)
	fmt.Fprintf(&builder, "%s\n", dimStyle.Render(fmt.Sprintf("Breadcrumb: Project > Services > Value Objects  |  Progress: 3/6  |  %s", service)))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Would you like to configure value objects before entities and fields?"))
	fmt.Fprintln(&builder)
	if (m.status == statusEditing || m.status == statusSaving) && m.edit.mode == editModeValueObjects {
		fmt.Fprintln(&builder, sectionTitleStyle.Render("Edit value objects"))
		m.renderValueObjectsEditor(&builder)
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter save | o rules | esc back to services"))
		return strings.TrimRight(builder.String(), "\n")
	}
	if !m.wizardValueObjectConfigured {
		for option, label := range []string{"Configure value objects", "Skip to entities", "Advanced configuration"} {
			fmt.Fprintln(&builder, m.wizardOptionAt(option, m.wizardValueObjectSelection, label))
		}
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardValueObjectDetail())
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter continue | esc back to services"))
		return strings.TrimRight(builder.String(), "\n")
	}
	for option := 0; option < m.wizardValueObjectOptionCount(); option++ {
		fmt.Fprintln(&builder, m.wizardValueObjectOption(option))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardValueObjectDetail())
	if m.err != nil {
		fmt.Fprintf(&builder, "%s %v\n", dangerStyle.Render("Operation failed:"), m.err)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, dangerStyle.Render("The saved value objects have a stale plan."))
		fmt.Fprintln(&builder, "Press r to retry the refresh before continuing.")
	}
	fmt.Fprintln(&builder)
	footer := "up/down or j/k select | enter open | esc back to services"
	if m.postSaveRefreshFailed() {
		footer += " | r retry refresh"
	}
	fmt.Fprintln(&builder, m.wizardFooter(footer))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardReviewView() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, dimStyle.Render("Breadcrumb: Project > Services > Value Objects > Entities > Fields > Review  |  Progress: 6/6"))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardPromptStyle().Render("Review your generation plan"))
	fmt.Fprintln(&builder)
	m.renderWizardReviewChecklist(&builder)
	fmt.Fprintln(&builder)
	for option, label := range []string{"Generate solution", "Inspect advanced preview", "Back to fields"} {
		fmt.Fprintln(&builder, m.wizardOptionAt(option, m.wizardSelection, label))
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.wizardFooter("up/down or j/k select | enter open | esc back to fields"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardServiceOption(option int) string {
	label := ""
	services := m.serviceSummaries()
	switch {
	case option < len(services):
		label = truncateWizardText(services[option].Name, m.wizardContentWidth()-4)
	case option == m.wizardServiceAddOption():
		label = "Add service"
	case option == m.wizardServiceEditOption():
		label = "Edit services"
	default:
		label = "Advanced configuration"
	}
	return m.wizardOptionAt(option, m.wizardServiceSelection, label)
}

func (m Model) wizardProjectOption(option int) string {
	label := ""
	switch {
	case option < len(m.targetFrameworkSuggestions):
		label = m.targetFrameworkSuggestions[option]
		if m.edit.targetFramework.string() == label {
			label += " (current)"
		}
	case option == m.wizardProjectEditOption():
		label = "Edit solution name and description"
	default:
		label = "Continue to services"
	}
	return m.wizardOptionAt(option, m.wizardProjectSelection, label)
}

func (m Model) wizardEntityOption(option int) string {
	entities := m.serviceEntitySummaries()
	label := ""
	switch {
	case option < len(entities):
		label = fmt.Sprintf("%s (%d fields)", truncateWizardText(entities[option].Name, maxInt(m.wizardContentWidth()-18, 4)), len(entities[option].Fields))
	case option == m.wizardEntityAddOption():
		label = "Add entity"
	case option == m.wizardEntityEditOption():
		label = "Edit entities"
	default:
		label = "Advanced configuration"
	}
	return m.wizardOptionAt(option, m.wizardEntitySelection, label)
}

func (m Model) wizardFieldOption(option int) string {
	fields := m.selectedEntitySummary().Fields
	label := ""
	switch {
	case option < len(fields):
		label = fmt.Sprintf("%s: %s", truncateWizardText(fields[option].Name, m.wizardContentWidth()/2), truncateWizardText(fields[option].Type, m.wizardContentWidth()/2))
	case option == m.wizardFieldContinueOption():
		label = "Continue to review"
	case option == m.wizardFieldAddOption():
		label = "Add field"
	case option == m.wizardFieldEditOption():
		label = "Edit fields"
	default:
		label = "Advanced configuration"
	}
	return m.wizardOptionAt(option, m.wizardFieldSelection, label)
}

func (m Model) wizardValueObjectOption(option int) string {
	valueObjects := m.serviceValueObjectSummaries()
	label := ""
	switch {
	case option < len(valueObjects):
		valueObject := valueObjects[option]
		label = fmt.Sprintf("%s: %s", truncateWizardText(valueObject.Name, m.wizardContentWidth()/2), truncateWizardText(valueObject.Type, m.wizardContentWidth()/2))
	case option == m.wizardValueObjectAddOption():
		label = "Add value object"
	case option == m.wizardValueObjectEditOption():
		label = "Edit value objects"
	case option == m.wizardValueObjectReviewOption():
		label = "Continue to review"
	default:
		label = "Advanced configuration"
	}
	return m.wizardOptionAt(option, m.wizardValueObjectSelection, label)
}

func (m Model) wizardOptionAt(option, selected int, label string) string {
	if option == selected {
		return m.wizardSelectedStyle().Render("> " + label)
	}
	return m.wizardTextStyle().Render("  " + label)
}

func (m Model) wizardServiceDetail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Selected service"))
	services := m.serviceSummaries()
	if m.wizardServiceSelection >= len(services) {
		switch m.wizardServiceSelection {
		case m.wizardServiceAddOption():
			fmt.Fprintln(&builder, "Create a service with a default Id Guid entity.")
		case m.wizardServiceEditOption():
			fmt.Fprintln(&builder, "Rename or remove services using the existing editor.")
		default:
			fmt.Fprintln(&builder, "Open the full route workspace for entities, fields, and value objects.")
		}
		return strings.TrimRight(builder.String(), "\n")
	}
	service := services[m.wizardServiceSelection]
	fieldCount := 0
	for _, entity := range service.Entities {
		fieldCount += len(entity.Fields)
	}
	fmt.Fprintf(&builder, "%s\n", truncateWizardText(service.Name, m.wizardContentWidth()))
	fmt.Fprintf(&builder, "Entities: %d | Fields: %d | Value objects: %d", len(service.Entities), fieldCount, len(service.ValueObjects))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardEntityDetail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Selected entity"))
	entities := m.serviceEntitySummaries()
	if m.wizardEntitySelection >= len(entities) {
		switch m.wizardEntitySelection {
		case m.wizardEntityAddOption():
			fmt.Fprintln(&builder, "Create an entity with a default Id Guid field.")
		case m.wizardEntityEditOption():
			fmt.Fprintln(&builder, "Rename or remove entities using the existing editor.")
		default:
			fmt.Fprintln(&builder, "Open the Advanced workspace for the full entity and field routes.")
		}
		return strings.TrimRight(builder.String(), "\n")
	}
	entity := entities[m.wizardEntitySelection]
	fmt.Fprintf(&builder, "%s\n", truncateWizardText(entity.Name, m.wizardContentWidth()))
	fmt.Fprintf(&builder, "Field count: %d\n", len(entity.Fields))
	if len(entity.Fields) == 0 {
		fmt.Fprintln(&builder, "Fields: none configured")
	} else {
		fmt.Fprintln(&builder, "Fields:")
		for _, field := range entity.Fields {
			fmt.Fprintf(&builder, "  %s: %s\n", truncateWizardText(field.Name, m.wizardContentWidth()/2), truncateWizardText(field.Type, m.wizardContentWidth()/2))
		}
	}
	references := m.entityValueObjectReferences(entity)
	if len(references) == 0 {
		fmt.Fprintln(&builder, "Referenced value objects: none")
	} else {
		fmt.Fprintf(&builder, "Referenced value objects: %s", truncateWizardText(strings.Join(references, ", "), m.wizardContentWidth()-28))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardFieldDetail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Selected field"))
	fields := m.selectedEntitySummary().Fields
	if m.wizardFieldSelection >= len(fields) {
		switch m.wizardFieldSelection {
		case m.wizardFieldAddOption():
			fmt.Fprintln(&builder, "Add a string field to the selected entity.")
		case m.wizardFieldEditOption():
			fmt.Fprintln(&builder, "Open the existing field editor for this entity.")
		default:
			fmt.Fprintln(&builder, "Open the Advanced workspace for fields and Value Objects.")
		}
		return strings.TrimRight(builder.String(), "\n")
	}
	field := fields[m.wizardFieldSelection]
	fmt.Fprintf(&builder, "Name: %s\n", truncateWizardText(field.Name, m.wizardContentWidth()-6))
	fmt.Fprintf(&builder, "Type: %s\n", truncateWizardText(field.Type, m.wizardContentWidth()-6))
	if valueObject, ok := m.wizardValueObject(field.Type); ok {
		rules := valueObject.RulesLabel
		if rules == "" {
			rules = rulesLabelForEdit(valueObject.Type, valueObjectRuleEditStateFromSummary(valueObject.Validations))
		}
		fmt.Fprintf(&builder, "Value object: %s | Rules: %s", truncateWizardText(valueObject.Name, m.wizardContentWidth()/3), truncateWizardText(rules, m.wizardContentWidth()/2))
	} else {
		fmt.Fprintln(&builder, "Hint: scalar field")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) wizardValueObjectDetail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Selected value object"))
	valueObjects := m.serviceValueObjectSummaries()
	if !m.wizardValueObjectConfigured {
		fmt.Fprintf(&builder, "Current value objects: %d\n", len(valueObjects))
		fmt.Fprintln(&builder, "Configure them now or continue directly to entities.")
		return strings.TrimRight(builder.String(), "\n")
	}
	if m.wizardValueObjectSelection >= len(valueObjects) {
		switch m.wizardValueObjectSelection {
		case m.wizardValueObjectAddOption():
			fmt.Fprintln(&builder, "Create a string-backed value object with conservative defaults.")
		case m.wizardValueObjectEditOption():
			fmt.Fprintln(&builder, "Open the existing value-object editor for this service.")
		case m.wizardValueObjectReviewOption():
			fmt.Fprintln(&builder, "Continue to entities and fields.")
		default:
			fmt.Fprintln(&builder, "Open the Advanced workspace for the full value-object route and rules editor.")
		}
		return strings.TrimRight(builder.String(), "\n")
	}
	valueObject := valueObjects[m.wizardValueObjectSelection]
	rules := valueObject.RulesLabel
	if rules == "" {
		rules = rulesLabelForEdit(valueObject.Type, valueObjectRuleEditStateFromSummary(valueObject.Validations))
	}
	fmt.Fprintf(&builder, "Name: %s\n", truncateWizardText(valueObject.Name, m.wizardContentWidth()-6))
	fmt.Fprintf(&builder, "Type: %s\n", truncateWizardText(valueObject.Type, m.wizardContentWidth()-6))
	fmt.Fprintf(&builder, "Rules: %s", truncateWizardText(rules, m.wizardContentWidth()-7))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) renderWizardReviewChecklist(builder *strings.Builder) {
	services, entities, fields, valueObjects := m.wizardPlanCounts()
	created, replaced, unchanged := plannedFileCounts(m.plan)
	forceRequired := m.plan.ForceRequired || m.plan.Readiness.OutputForceRequired
	forceState := "not required"
	if forceRequired {
		forceState = "required"
		if m.request.Force {
			forceState = "confirmed"
		}
	}
	projectState := "not configured"
	if m.plan.Config.SolutionName != "" || m.plan.Readiness.ProjectPresent {
		projectState = "ready"
	}
	lineWidth := maxInt(m.wizardContentWidth(), 4)
	fmt.Fprintf(builder, "%s\n", truncateWizardText(fmt.Sprintf("[ready] Solution: %s", m.plan.Config.SolutionName), lineWidth))
	fmt.Fprintf(builder, "%s\n", truncateWizardText(fmt.Sprintf("[ready] Services: %d | Entities: %d | Fields: %d | Value objects: %d", services, entities, fields, valueObjects), lineWidth))
	fmt.Fprintf(builder, "[ready] Project readiness: %s\n", projectState)
	fmt.Fprintf(builder, "%s\n", truncateWizardText(fmt.Sprintf("[ready] Output directory: %s", m.outputDirectory()), lineWidth))
	fmt.Fprintf(builder, "%s\n", truncateWizardText(fmt.Sprintf("[ready] Changes: created=%d | replaced=%d | unchanged=%d | deleted=%d", created, replaced, unchanged, len(m.plan.DeletedFiles)), lineWidth))
	if forceRequired && !m.request.Force {
		fmt.Fprintf(builder, "%s Force: %s\n", dangerStyle.Render("[blocked]"), forceState)
	} else {
		fmt.Fprintf(builder, "[ready] Force: %s\n", forceState)
	}
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(builder, dangerStyle.Render("[blocked] Plan refresh required before generation"))
	}
	for _, hint := range m.plan.Readiness.Hints {
		fmt.Fprintf(builder, "%s %s\n", labelStyle.Render("Hint:"), truncateWizardText(hint, m.wizardContentWidth()-6))
	}
	if len(m.plan.Readiness.Hints) == 0 && !m.postSaveRefreshFailed() {
		fmt.Fprintln(builder, "Hint: Plan is ready for generation.")
	}
}

func (m Model) wizardPlanCounts() (services, entities, fields, valueObjects int) {
	for _, service := range m.serviceSummaries() {
		services++
		serviceEntities := service.Entities
		if len(serviceEntities) == 0 {
			entities += len(service.EntityNames)
		} else {
			entities += len(serviceEntities)
			for _, entity := range serviceEntities {
				fields += len(entity.Fields)
			}
		}
		valueObjects += len(m.serviceValueObjects(service))
	}
	readiness := m.plan.Readiness
	if services == 0 {
		services = readiness.ServiceCount
		if services == 0 {
			services = m.plan.Config.ServiceCount
		}
	}
	if entities == 0 {
		entities = readiness.EntityCount
		if entities == 0 {
			entities = m.plan.Config.EntityCount
		}
	}
	if fields == 0 {
		fields = readiness.FieldCount
	}
	if valueObjects == 0 {
		valueObjects = readiness.ValueObjectCount
		if valueObjects == 0 {
			valueObjects = m.plan.Config.ValueObjectCount
		}
	}
	return services, entities, fields, valueObjects
}

func (m Model) serviceValueObjects(service application.ServiceSummary) []application.ValueObjectSummary {
	if len(service.ValueObjects) > 0 {
		return service.ValueObjects
	}
	valueObjects := make([]application.ValueObjectSummary, 0, len(service.ValueObjectNames))
	for _, name := range service.ValueObjectNames {
		valueObjects = append(valueObjects, application.ValueObjectSummary{Name: name})
	}
	return valueObjects
}

func (m Model) wizardValueObject(fieldType string) (application.ValueObjectSummary, bool) {
	for _, valueObject := range m.serviceValueObjectSummaries() {
		if valueObject.Name == fieldType {
			return valueObject, true
		}
	}
	return application.ValueObjectSummary{}, false
}

func (m Model) wizardFooter(controls string) string {
	return dimStyle.Render("Footer: " + controls)
}

func (m Model) wizardContentWidth() int {
	width := m.windowWidth
	if width <= 0 {
		return 80
	}
	if width < 20 {
		return 20
	}
	return width - 2
}

func truncateWizardText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func (m Model) wizardOption(option int, label string) string {
	if option == m.wizardSelection {
		return m.wizardSelectedStyle().Render("> " + label)
	}
	return m.wizardTextStyle().Render("  " + label)
}

func (m Model) wizardResultOption(option int, label string) string {
	if option == m.wizardResultSelection {
		return m.wizardSelectedStyle().Render("> " + label)
	}
	return m.wizardTextStyle().Render("  " + label)
}

func (m Model) wizardTextStyle() lipgloss.Style {
	if m.windowWidth <= 0 {
		return dimStyle
	}
	width := m.windowWidth - 2
	if width < 1 {
		width = 1
	}
	return dimStyle.Width(width)
}

func (m Model) wizardSelectedStyle() lipgloss.Style {
	style := selectedRowStyle
	if m.windowWidth > 0 {
		width := m.windowWidth - 2
		if width < 1 {
			width = 1
		}
		style = style.Width(width)
	}
	return style
}

func (m Model) wizardPromptStyle() lipgloss.Style {
	style := sectionTitleStyle
	if m.windowWidth > 0 {
		width := m.windowWidth - 2
		if width < 1 {
			width = 1
		}
		style = style.Width(width)
	}
	return style
}

func wizardOptionLabel(option int) string {
	switch option {
	case wizardConfigureProject:
		return "Configure project"
	case wizardConfigureServices:
		return "Configure services, value objects, entities, and fields"
	case wizardReviewChanges:
		return "Review generated changes"
	case wizardGenerateSolution:
		return "Generate solution"
	case wizardAdvancedWorkspace:
		return "Advanced workspace"
	default:
		return "Quit"
	}
}

func (m Model) guidedWorkspaceView() string {
	m.layout = layoutNarrow
	var builder strings.Builder
	fmt.Fprintln(&builder, m.guidedWorkspaceHeader())
	fmt.Fprintln(&builder, m.workspaceContent())
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, m.guidedWorkspaceFooter())
	return builder.String()
}

func (m Model) guidedWorkspaceHeader() string {
	return fmt.Sprintf("%s - %s\n%s", appTitleStyle.Render("Microgen"), m.statusBadge(), dimStyle.Render("Breadcrumb: Wizard / "+m.activeScreen().label()))
}

func (m Model) guidedWorkspaceFooter() string {
	var controls string
	switch m.activeScreen() {
	case screenProject:
		controls = "esc back | tab/up/down fields | enter save"
	case screenServices:
		controls = "esc back | up/down select | enter open | e edit"
	case screenEntities:
		controls = "esc back | up/down select | enter edit | f fields"
	case screenValueObjects:
		controls = "esc back | up/down select | enter edit | o rules"
	case screenPreview:
		controls = "esc back | arrows/j/k files | r refresh | g generate"
	case screenGenerate:
		controls = "esc back | enter or g confirm generation | r refresh"
	case screenResult:
		controls = "esc back to menu | enter select | q quit"
	default:
		controls = "esc back | q quit"
	}
	return dimStyle.Render("Footer: " + controls)
}

func (m Model) workspaceHeader() string {
	project := m.plan.Config.SolutionName
	if project == "" {
		project = "Unconfigured project"
	}
	workspace := "Workspace"
	if m.returnToWizard {
		workspace = "Advanced workspace | esc back to wizard"
	}
	return fmt.Sprintf("%s - %s  %s  %s %s\n%s", appTitleStyle.Render("Microgen"), m.statusBadge(), dimStyle.Render(workspace), labelStyle.Render("Current project:"), project, m.primaryActionStyle().Render("Primary: "+m.primaryAction()))
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
	lines := []string{"Navigation "}
	for screen := screenOverview; screen < screenCount; screen++ {
		plainLabel := fmt.Sprintf("%d:%s", screen+1, screen.label())
		label := plainLabel
		if screen == m.activeScreen() {
			label = readyStyle.Render("[" + label + "]")
		}
		separator := ""
		if lines[len(lines)-1] != "Navigation " {
			separator = "  "
		}
		candidate := lines[len(lines)-1] + separator + label
		if m.windowWidth > 0 && lipgloss.Width(candidate) > m.windowWidth && lines[len(lines)-1] != "Navigation " {
			lines = append(lines, strings.Repeat(" ", len("Navigation "))+label)
			continue
		}
		lines[len(lines)-1] = candidate
	}
	return dimStyle.Render(strings.Join(lines, "\n"))
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
	case screenEntities:
		return m.entitiesWorkspace()
	case screenValueObjects:
		return m.valueObjectsWorkspace()
	case screenPreview:
		return m.previewWorkspace()
	case screenGenerate:
		return m.generateStepCard()
	case screenResult:
		return m.resultWorkspace()
	default:
		return m.overviewCard()
	}
}

func (m Model) helpOverlay() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, cardStyle.Render(strings.Join([]string{
		sectionTitleStyle.Render("Help"),
		"Global: 1-7 open screens | up/down select route/resource | enter open | h/l or left/right switch",
		"Global: ? close help | esc back | q/ctrl+c quit",
		"Overview: r refresh | g generate",
		"Project: e edit settings | r refresh",
		"Services: tab or left/right switch Services/Entities/Value Objects | up/down select | enter open",
		"Entities: up/down select | enter/e edit | f fields | a/r/d in editor",
		"Value Objects: up/down select | enter/e edit | o rules | a/r/d in editor",
		"Services actions: e service list | enter Entities/Value Objects context | v value objects",
		"Preview: arrows/k/j inspect files | a filter | r refresh | g continue",
		"Generate: g confirm generation | r refresh",
		"Result: esc back to Generate | g retry after failure | r refresh",
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

	fmt.Fprintln(&builder, m.resourceColumns(m.servicesResourceList(), m.serviceDetail()))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) resourceColumns(list, detail string) string {
	listCard := cardStyle.Render(list)
	detailCard := cardStyle.Render(detail)
	if m.layout != layoutWide {
		return listCard + "\n\n" + detailCard
	}
	available := m.windowWidth - 30
	if available < 60 {
		available = 60
	}
	listWidth := available / 3
	detailWidth := available - listWidth - 2
	listCard = cardStyle.Width(listWidth).Render(list)
	detailCard = cardStyle.Width(detailWidth).Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, listCard, "  ", detailCard)
}

func (m Model) entitiesWorkspace() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Entities"))
	fmt.Fprintf(&builder, "%s Services > Entities\n", dimStyle.Render("Breadcrumb:"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Selected service:"), m.selectedServiceSummary().Name)
	if (m.status == statusEditing && m.edit.mode == editModeEntities) || (m.status == statusSaving && m.edit.mode == editModeEntities) || (m.status == statusEditing && m.edit.mode == editModeFields) || (m.status == statusSaving && m.edit.mode == editModeFields) {
		if m.edit.mode == editModeFields {
			m.renderFieldsEditor(&builder)
		} else {
			m.renderEntitiesEditor(&builder)
		}
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	fmt.Fprintln(&builder, m.resourceColumns(m.entitiesResourceList(), m.entityDetail()))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) valueObjectsWorkspace() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Value Objects"))
	fmt.Fprintf(&builder, "%s Services > Value Objects\n", dimStyle.Render("Breadcrumb:"))
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Selected service:"), m.selectedServiceSummary().Name)
	if (m.status == statusEditing && m.edit.mode == editModeValueObjects) || (m.status == statusSaving && m.edit.mode == editModeValueObjects) {
		m.renderValueObjectsEditor(&builder)
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	fmt.Fprintln(&builder, m.resourceColumns(m.valueObjectsResourceList(), m.valueObjectDetail()))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) entitiesResourceList() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Entity list"))
	entities := m.serviceEntitySummaries()
	if len(entities) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No entities configured."))
		return strings.TrimRight(builder.String(), "\n")
	}
	for index, entity := range entities {
		row := fmt.Sprintf("  %s (%d fields)", entity.Name, len(entity.Fields))
		if index == m.selectedEntity {
			row = selectedRowStyle.Render(fmt.Sprintf("> %s (%d fields)", entity.Name, len(entity.Fields)))
		}
		fmt.Fprintln(&builder, row)
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, dimStyle.Render("up/down select | enter/e edit"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) entityDetail() string {
	entities := m.serviceEntitySummaries()
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Entity detail"))
	if len(entities) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No entity selected."))
		return strings.TrimRight(builder.String(), "\n")
	}
	entity := entities[clampInt(m.selectedEntity, 0, len(entities)-1)]
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Entity:"), entity.Name)
	fmt.Fprintf(&builder, "%s %d\n", labelStyle.Render("Field count:"), len(entity.Fields))
	fmt.Fprintln(&builder, labelStyle.Render("Fields"))
	if len(entity.Fields) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("  No fields configured."))
	} else {
		for _, field := range entity.Fields {
			fmt.Fprintf(&builder, "  %s: %s\n", field.Name, field.Type)
		}
	}
	fmt.Fprintln(&builder, labelStyle.Render("Referenced value objects"))
	references := m.entityValueObjectReferences(entity)
	if len(references) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("  None"))
	} else {
		for _, reference := range references {
			fmt.Fprintf(&builder, "  %s\n", reference)
		}
	}
	fmt.Fprintln(&builder)
	if m.busy() || m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, busyStyle.Render("Editing is paused until the current operation finishes or the stale plan is refreshed."))
	} else {
		fmt.Fprintln(&builder, successStyle.Render("Enter/e edit entity | f edit fields"))
		fmt.Fprintln(&builder, dimStyle.Render("a/r/d add, rename, or delete in editor"))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) valueObjectsResourceList() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Value object list"))
	valueObjects := m.serviceValueObjectSummaries()
	if len(valueObjects) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No value objects configured."))
		return strings.TrimRight(builder.String(), "\n")
	}
	for index, valueObject := range valueObjects {
		row := fmt.Sprintf("  %s: %s", valueObject.Name, valueObject.Type)
		if index == m.selectedValueObject {
			row = selectedRowStyle.Render(fmt.Sprintf("> %s: %s", valueObject.Name, valueObject.Type))
		}
		fmt.Fprintln(&builder, row)
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, dimStyle.Render("up/down select | enter/e edit"))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) valueObjectDetail() string {
	valueObjects := m.serviceValueObjectSummaries()
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Value object detail"))
	if len(valueObjects) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("No value object selected."))
		return strings.TrimRight(builder.String(), "\n")
	}
	valueObject := valueObjects[clampInt(m.selectedValueObject, 0, len(valueObjects)-1)]
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Value object:"), valueObject.Name)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Type:"), valueObject.Type)
	rules := valueObject.RulesLabel
	if rules == "" {
		rules = rulesLabelForEdit(valueObject.Type, valueObjectRuleEditStateFromSummary(valueObject.Validations))
	}
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Validation rules:"), rules)
	fmt.Fprintln(&builder, labelStyle.Render("Referencing fields"))
	references := m.valueObjectFieldReferences(valueObject.Name)
	if len(references) == 0 {
		fmt.Fprintln(&builder, dimStyle.Render("  None"))
	} else {
		for _, reference := range references {
			fmt.Fprintf(&builder, "  %s\n", reference)
		}
	}
	fmt.Fprintln(&builder)
	if m.busy() || m.postSaveRefreshFailed() {
		fmt.Fprintln(&builder, busyStyle.Render("Editing is paused until the current operation finishes or the stale plan is refreshed."))
	} else {
		fmt.Fprintln(&builder, successStyle.Render("Enter/e edit value object | o edit rules"))
		fmt.Fprintln(&builder, dimStyle.Render("a/r/d add, rename, or delete in editor"))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) entityValueObjectReferences(entity application.EntitySummary) []string {
	service := m.selectedServiceSummary()
	valueObjectNames := make(map[string]bool)
	for _, valueObject := range m.serviceValueObjectSummaries() {
		valueObjectNames[valueObject.Name] = true
	}
	seen := make(map[string]bool)
	references := make([]string, 0)
	for _, field := range entity.Fields {
		if valueObjectNames[field.Type] && !seen[field.Type] {
			seen[field.Type] = true
			references = append(references, field.Type)
		}
	}
	for _, reference := range service.ValueObjectReferences {
		if reference.EntityName == entity.Name && !seen[reference.ValueObjectName] {
			seen[reference.ValueObjectName] = true
			references = append(references, reference.ValueObjectName)
		}
	}
	sort.Strings(references)
	return references
}

func (m Model) valueObjectFieldReferences(valueObjectName string) []string {
	service := m.selectedServiceSummary()
	seen := make(map[string]bool)
	references := make([]string, 0)
	for _, reference := range service.ValueObjectReferences {
		if reference.ValueObjectName == valueObjectName {
			label := fmt.Sprintf("%s.%s", reference.EntityName, reference.FieldName)
			if !seen[label] {
				seen[label] = true
				references = append(references, label)
			}
		}
	}
	for _, entity := range m.serviceEntitySummaries() {
		for _, field := range entity.Fields {
			if field.Type == valueObjectName {
				label := fmt.Sprintf("%s.%s", entity.Name, field.Name)
				if !seen[label] {
					seen[label] = true
					references = append(references, label)
				}
			}
		}
	}
	sort.Strings(references)
	return references
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
	m.renderGenerationChecklist(&builder)
	switch m.status {
	case statusGenerating:
		fmt.Fprintln(&builder, busyStyle.Render("Generating files. Please wait; exit is available after generation finishes."))
	case statusGenerated:
		m.renderPostGenerateSummary(&builder)
		fmt.Fprintln(&builder, dimStyle.Render("Result is available on the Result route."))
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
		fmt.Fprintf(&builder, "%s Generate %d planned file(s) into %s.\n", dimStyle.Render("Plan:"), m.plan.FileCount, m.outputDirectory())
		fmt.Fprintf(&builder, "%s Review the Preview step before confirming writes.\n", labelStyle.Render("Before"))
		if m.generationBlocked() {
			fmt.Fprintln(&builder, dangerStyle.Render(m.generationBlockMessage()))
			fmt.Fprintln(&builder, dimStyle.Render("Resolve the checklist item before confirming generation."))
		} else {
			fmt.Fprintf(&builder, "%s Confirm writing %d planned file(s) into %s.\n", successStyle.Render("g"), m.plan.FileCount, m.outputDirectory())
			fmt.Fprintf(&builder, "%s Preview reviewed. Press g to confirm the write.\n", labelStyle.Render("Ready"))
		}
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) renderGenerationChecklist(builder *strings.Builder) {
	if m.postSaveRefreshFailed() {
		fmt.Fprintln(builder, dangerStyle.Render("Readiness is stale. Saved settings need a successful plan refresh before generation."))
		fmt.Fprintln(builder, dangerStyle.Render("[blocked] Plan refresh required"))
		return
	}
	readiness := m.plan.Readiness
	project := "not ready"
	if readiness.ProjectPresent || m.plan.Config.SolutionName != "" {
		project = "ready"
	}
	force := "not required"
	if m.plan.ForceRequired || readiness.OutputForceRequired {
		force = "required"
		if m.request.Force {
			force = "confirmed"
		}
	}
	projectReady := "no"
	if readiness.ProjectPresent || m.plan.Config.SolutionName != "" {
		projectReady = "yes"
	}
	forceRequired := "no"
	if m.plan.ForceRequired || readiness.OutputForceRequired {
		forceRequired = "yes"
	}
	fmt.Fprintf(builder, "%s project=%s, services=%d, entities=%d, fields=%d, value objects=%d, force required=%s\n", labelStyle.Render("Readiness"), projectReady, readiness.ServiceCount, readiness.EntityCount, readiness.FieldCount, readiness.ValueObjectCount, forceRequired)
	fmt.Fprintln(builder, sectionTitleStyle.Render("Readiness checklist"))
	fmt.Fprintf(builder, "[ready] Project configured: %s\n", project)
	fmt.Fprintf(builder, "[ready] Resources: services=%d, entities=%d, fields=%d, value objects=%d\n", readiness.ServiceCount, readiness.EntityCount, readiness.FieldCount, readiness.ValueObjectCount)
	fmt.Fprintf(builder, "[ready] Output directory: %s\n", m.outputDirectory())
	if force == "required" {
		fmt.Fprintf(builder, "%s Force confirmation: %s\n", dangerStyle.Render("[blocked]"), force)
	} else {
		fmt.Fprintf(builder, "[ready] Force confirmation: %s\n", force)
	}
	if len(readiness.Hints) > 0 {
		for _, hint := range readiness.Hints {
			fmt.Fprintf(builder, "%s %s\n", labelStyle.Render("Next"), hint)
		}
	}
	fmt.Fprintln(builder)
}

func (m Model) renderPostGenerateSummary(builder *strings.Builder) {
	plan := m.result.Plan
	outputDir := m.result.OutputDir
	if outputDir == "" {
		outputDir = m.outputDirectory()
	}
	fmt.Fprintf(builder, "%s %d files written to %s.\n", successStyle.Render("Generated"), plan.FileCount, outputDir)
	fmt.Fprintf(builder, "%s %s\n", labelStyle.Render("Impact"), postGenerateImpactSummary(plan))
	if len(plan.DeletedFiles) > 0 {
		fmt.Fprintf(builder, "%s deleted %d previous generated file(s)\n", dangerStyle.Render("Cleanup"), len(plan.DeletedFiles))
	}
	fmt.Fprintf(builder, "%s cd %s && dotnet build\n", labelStyle.Render("Next"), outputDir)
	if m.result.Warning != "" {
		fmt.Fprintf(builder, "%s %s\n", warningStyle.Render("WARNING"), warningStyle.Render(m.result.Warning))
	}
}

func (m Model) resultWorkspace() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("Result"))
	if m.status == statusFailed {
		context := m.errContext
		if context == "" {
			context = "Generation"
		}
		fmt.Fprintf(&builder, "%s %s failed: %v\n", dangerStyle.Render("FAILED"), context, m.err)
		fmt.Fprintln(&builder, dangerStyle.Render("g Retry generation, or r refresh the plan first. esc back to Generate."))
		fmt.Fprintln(&builder, dimStyle.Render("r Refresh the plan before retrying if the output or config changed."))
		return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
	}
	m.renderPostGenerateSummary(&builder)
	if len(m.result.Plan.DeletedFiles) > 0 {
		fmt.Fprintln(&builder, labelStyle.Render("Deleted files"))
		for _, path := range m.result.Plan.DeletedFiles {
			fmt.Fprintf(&builder, "  %s\n", path)
		}
	}
	return cardStyle.Render(strings.TrimRight(builder.String(), "\n"))
}

func (m Model) previewWorkspace() string {
	var summary strings.Builder
	fmt.Fprintln(&summary, sectionTitleStyle.Render("Output Preview"))
	fmt.Fprintf(&summary, "%s %s\n", labelStyle.Render("Directory"), m.outputDirectory())
	fmt.Fprintf(&summary, "%s %s\n", labelStyle.Render("Write mode"), m.writeMode())
	readinessLabel := "ready"
	if m.postSaveRefreshFailed() {
		readinessLabel = "stale"
	} else if m.generationBlocked() {
		readinessLabel = "action required"
	}
	fmt.Fprintf(&summary, "%s %s\n", labelStyle.Render("Readiness"), readinessLabel)
	forceRequired := m.plan.ForceRequired || m.plan.Readiness.OutputForceRequired
	forceUsed := m.plan.ForceUsed || m.request.Force
	forceStyle := dimStyle
	if forceRequired && !forceUsed {
		forceStyle = dangerStyle
	} else if forceRequired {
		forceStyle = warningStyle
	}
	fmt.Fprintf(&summary, "%s %s\n", labelStyle.Render("Force"), forceStyle.Render(fmt.Sprintf("required=%s, used=%s", yesNo(forceRequired), yesNo(forceUsed))))
	fmt.Fprintf(&summary, "%s %d planned\n", labelStyle.Render("Files"), m.plan.FileCount)
	fmt.Fprintf(&summary, "%s %s\n", labelStyle.Render("Impact"), m.impactSummary())
	created, replaced, unchanged := plannedFileCounts(m.plan)
	fmt.Fprintf(&summary, "%s created=%d, replaced=%d, unchanged=%d, deleted=%d\n", labelStyle.Render("Change counts"), created, replaced, unchanged, len(m.plan.DeletedFiles))
	if m.plan.FileCount > 0 && m.impactSummary() == "unchanged only" {
		fmt.Fprintln(&summary, successStyle.Render("No generated file content changes detected."))
	}
	if m.plan.ExtraFileCount > 0 || len(m.plan.DeletedFiles) > 0 {
		fmt.Fprintf(&summary, "%s replacement removes %d previous generated file(s)\n", dangerStyle.Render("DANGER"), len(m.plan.DeletedFiles))
		fmt.Fprintf(&summary, "%s\n", dangerStyle.Render(deletedFilePreview(m.plan.DeletedFiles)))
	}

	files := m.plannedFilesCard()
	detail := m.plannedFileDetail()
	if m.layout == layoutWide {
		available := m.windowWidth - 30
		if available < 72 {
			available = 72
		}
		leftWidth := available / 3
		rightWidth := available - leftWidth - 2
		left := cardStyle.Width(leftWidth).Render(summary.String())
		right := cardStyle.Width(rightWidth).Render(files + "\n\n" + detail)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	}
	return strings.Join([]string{summary.String(), files, detail}, "\n\n")
}

func (m Model) plannedFileDetail() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, sectionTitleStyle.Render("File detail"))
	indices := m.filteredFileIndices()
	position := m.selectedFilteredPosition(indices)
	if position < 0 || position >= len(indices) {
		fmt.Fprintln(&builder, dimStyle.Render("No file selected."))
		return strings.TrimRight(builder.String(), "\n")
	}
	file := m.plan.Files[indices[position]]
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Path"), file.Path)
	fmt.Fprintf(&builder, "%s %s\n", labelStyle.Render("Action"), actionBadge(file.Action))
	fmt.Fprintf(&builder, "%s %d of %d\n", labelStyle.Render("Position"), position+1, len(indices))
	return strings.TrimRight(builder.String(), "\n")
}

func (m Model) outputDirectory() string {
	if m.plan.OutputDir != "" {
		return m.plan.OutputDir
	}
	if m.result.OutputDir != "" {
		return m.result.OutputDir
	}
	return m.request.OutputDir
}

func (m Model) writeMode() string {
	if m.plan.OutputAction != "" {
		return m.plan.OutputAction
	}
	return "create"
}

func plannedFileCounts(plan application.GenerationPlan) (created, replaced, unchanged int) {
	for _, file := range plan.Files {
		switch file.Action {
		case "create":
			created++
		case "replace":
			replaced++
		case "unchanged":
			unchanged++
		}
	}
	return created, replaced, unchanged
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
	case screenEntities:
		fmt.Fprintln(&builder, "Entities: up/down select; enter/e edit; f fields; a/r/d in editor; r refresh.")
	case screenValueObjects:
		fmt.Fprintln(&builder, "Value Objects: up/down select; enter/e edit; o rules; a/r/d in editor; r refresh.")
	case screenPreview:
		fmt.Fprintln(&builder, readyHelp)
	case screenGenerate:
		fmt.Fprintln(&builder, "Generate: g confirm generation, r refresh.")
	case screenResult:
		if m.status == statusFailed {
			fmt.Fprintln(&builder, "Result: g retry generation, esc back to Generate, r refresh.")
			fmt.Fprintln(&builder, "Generate: g generate, r refresh.")
		} else {
			fmt.Fprintln(&builder, generatedHelp)
			fmt.Fprintln(&builder, "Result: esc back to Generate.")
		}
	default:
		fmt.Fprintln(&builder, "Overview: r refresh, 2 Project, 3 Services, 4 Entities, 5 Value Objects, 6 Preview, 7 Generate.")
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
	fmt.Fprintln(builder, "Use the Services, Value Objects, Entities, and Fields routes for resource editing.")
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
	fmt.Fprintln(builder, "Press enter from Services to open Entities; use the Value Objects route for value objects.")
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
	fmt.Fprintln(builder, dimStyle.Render("Breadcrumb: Services > Entities > Fields"))
	fmt.Fprintf(builder, "Editing fields for %s/%s\n", m.fieldsEdit.serviceName, m.fieldsEdit.entityName)
	fmt.Fprintln(builder, "Field details are name and type. Value Object Rules are edited from the Value Objects route.")
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
	fmt.Fprintln(builder, dimStyle.Render("Breadcrumb: Services > Value Objects > Rules"))
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
		parts = append(parts, fmt.Sprintf("%s=%d", item.label, counts[item.action]))
		delete(counts, item.action)
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
