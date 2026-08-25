package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
	"github.com/rivo/tview"
)

type tviewScreen int

const (
	tviewScreenOverview tviewScreen = iota
	tviewScreenProject
	tviewScreenServices
	tviewScreenEntities
	tviewScreenValueObjects
	tviewScreenPreview
	tviewScreenGenerate
	tviewScreenResult
)

type tviewFocusPanel int

const (
	tviewFocusSidebar tviewFocusPanel = iota
	tviewFocusDetails
	tviewFocusFiles
	tviewFocusShortcuts
)

type tviewRoute struct {
	screen tviewScreen
	label  string
}

var tviewRoutes = []tviewRoute{
	{tviewScreenOverview, "Overview"},
	{tviewScreenProject, "Project"},
	{tviewScreenServices, "Services"},
	{tviewScreenEntities, "Entities"},
	{tviewScreenValueObjects, "Value Objects"},
	{tviewScreenPreview, "Preview"},
	{tviewScreenGenerate, "Generate"},
	{tviewScreenResult, "Result"},
}

type tviewUI struct {
	app                        *tview.Application
	root                       *tview.Pages
	dashboard                  tview.Primitive
	sidebar                    *tview.Table
	detail                     *tview.TextView
	files                      *tview.Table
	footer                     *tview.TextView
	plan                       application.GenerationPlan
	request                    application.GenerateRequest
	planFunc                   PlanFunc
	generate                   GenerateFunc
	updateSettings             UpdateSettingsFunc
	updateServices             UpdateServicesFunc
	updateEntities             UpdateEntitiesFunc
	updateFields               UpdateFieldsFunc
	updateValueObjects         UpdateValueObjectsFunc
	targetFrameworkSuggestions []string
	result                     application.GenerateResult
	err                        error
	message                    string
	screen                     tviewScreen
	focus                      tviewFocusPanel
	editOpen                   bool
	planStale                  bool
}

const (
	tviewEditModalPage       = "edit-modal"
	tviewEditModalInputWidth = 32
)

type tviewManagerRow struct {
	original    string
	name        string
	typeName    string
	deleted     bool
	validations application.ValidationRuleSettings
}

type tviewServicesEditState struct {
	rows []tviewManagerRow
}

type tviewEntitiesEditState struct {
	serviceName string
	rows        []tviewManagerRow
}

type tviewFieldsEditState struct {
	serviceName string
	entityName  string
	rows        []tviewManagerRow
}

type tviewValueObjectsEditState struct {
	serviceName string
	rows        []tviewManagerRow
}

type tviewValueObjectRulesEditState struct {
	serviceName string
	valueObject application.ValueObjectSummary
	rows        []tviewManagerRow
	validations application.ValidationRuleSettings
}

var runTViewApplication = func(app *tview.Application, root tview.Primitive) error {
	return app.SetRoot(root, true).EnableMouse(true).Run()
}

func newTViewUI(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc, callbacks ...any) *tviewUI {
	ui := &tviewUI{
		app:      tview.NewApplication(),
		plan:     plan,
		request:  request,
		planFunc: planFunc,
		generate: generate,
		screen:   tviewScreenOverview,
	}
	if len(callbacks) > 0 {
		ui.updateSettings = tviewUpdateSettingsCallback(callbacks[0])
	}
	if len(callbacks) > 1 {
		ui.updateServices = tviewUpdateServicesCallback(callbacks[1])
	}
	if len(callbacks) > 2 {
		ui.updateEntities = tviewUpdateEntitiesCallback(callbacks[2])
	}
	if len(callbacks) > 3 {
		ui.updateFields = tviewUpdateFieldsCallback(callbacks[3])
	}
	if len(callbacks) > 4 {
		ui.updateValueObjects = tviewUpdateValueObjectsCallback(callbacks[4])
	}
	if len(callbacks) > 5 {
		ui.targetFrameworkSuggestions, _ = callbacks[5].([]string)
	}
	ui.root = ui.build()
	ui.setFocusedPanel(tviewFocusSidebar)
	ui.refresh()
	return ui
}

func tviewUpdateSettingsCallback(callback any) UpdateSettingsFunc {
	switch fn := callback.(type) {
	case UpdateSettingsFunc:
		return fn
	case func(application.GenerateRequest, application.SolutionSettings) (application.UpdateSolutionSettingsResult, error):
		return fn
	default:
		return nil
	}
}

func tviewUpdateServicesCallback(callback any) UpdateServicesFunc {
	switch fn := callback.(type) {
	case UpdateServicesFunc:
		return fn
	case func(application.GenerateRequest, application.ServiceSettings) (application.UpdateServiceSettingsResult, error):
		return fn
	default:
		return nil
	}
}

func tviewUpdateEntitiesCallback(callback any) UpdateEntitiesFunc {
	switch fn := callback.(type) {
	case UpdateEntitiesFunc:
		return fn
	case func(application.GenerateRequest, application.EntitySettings) (application.UpdateEntitySettingsResult, error):
		return fn
	default:
		return nil
	}
}

func tviewUpdateFieldsCallback(callback any) UpdateFieldsFunc {
	switch fn := callback.(type) {
	case UpdateFieldsFunc:
		return fn
	case func(application.GenerateRequest, application.FieldSettings) (application.UpdateFieldSettingsResult, error):
		return fn
	default:
		return nil
	}
}

func tviewUpdateValueObjectsCallback(callback any) UpdateValueObjectsFunc {
	switch fn := callback.(type) {
	case UpdateValueObjectsFunc:
		return fn
	case func(application.GenerateRequest, application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error):
		return fn
	default:
		return nil
	}
}

func (ui *tviewUI) build() *tview.Pages {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDarkSlateGray
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorDarkSlateGray
	tview.Styles.BorderColor = tcell.ColorDarkCyan
	tview.Styles.TitleColor = tcell.ColorLightCyan
	tview.Styles.GraphicsColor = tcell.ColorDarkCyan
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorGray
	tview.Styles.TertiaryTextColor = tcell.ColorDarkCyan
	tview.Styles.InverseTextColor = tcell.ColorBlack

	ui.sidebar = tview.NewTable().SetSelectable(true, false)
	ui.sidebar.SetBorder(true).SetTitle(" microgen ").SetTitleAlign(tview.AlignLeft)
	ui.sidebar.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal).Bold(true))
	ui.sidebar.SetSelectedFunc(func(row, _ int) {
		if row >= 0 && row < len(tviewRoutes) {
			ui.open(tviewRoutes[row].screen)
		}
	})

	ui.detail = tview.NewTextView().SetDynamicColors(true).SetWordWrap(true).SetScrollable(true)
	ui.detail.SetBorder(true).SetTitle(" Details ").SetTitleAlign(tview.AlignLeft)

	ui.files = tview.NewTable().SetBorders(false).SetSelectable(true, false).SetFixed(1, 0)
	ui.files.SetBorder(true).SetTitle(" Planned Files ").SetTitleAlign(tview.AlignLeft)
	ui.files.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal))

	ui.footer = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	ui.footer.SetBorder(true).SetTitle(" Shortcuts ").SetTitleAlign(tview.AlignLeft)

	main := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.detail, 0, 2, false).
		AddItem(ui.files, 0, 3, false).
		AddItem(ui.footer, 3, 0, false)

	ui.dashboard = tview.NewFlex().
		AddItem(ui.sidebar, 24, 0, true).
		AddItem(main, 0, 1, false)

	ui.app.SetInputCapture(ui.handleKey)
	return tview.NewPages().AddPage("dashboard", ui.dashboard, true, true)
}

func (ui *tviewUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.editOpen {
		if event.Key() == tcell.KeyCtrlC {
			ui.app.Stop()
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			ui.closeEdit()
			return nil
		}
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		ui.app.Stop()
		return nil
	case tcell.KeyTAB:
		ui.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		ui.cycleFocus(-1)
		return nil
	case tcell.KeyEscape:
		ui.open(tviewScreenOverview)
		return nil
	case tcell.KeyUp:
		ui.moveFocused(-1)
		return nil
	case tcell.KeyDown:
		ui.moveFocused(1)
		return nil
	}
	switch strings.ToLower(string(event.Rune())) {
	case "q":
		ui.app.Stop()
		return nil
	case "j":
		ui.moveFocused(1)
		return nil
	case "k":
		ui.moveFocused(-1)
		return nil
	case "r":
		ui.refreshPlan()
		return nil
	case "g":
		if ui.screen != tviewScreenGenerate {
			ui.open(tviewScreenGenerate)
			return nil
		}
		ui.generateFiles()
		return nil
	case "e":
		ui.startEdit()
		return nil
	}
	return event
}

func (ui *tviewUI) cycleFocus(delta int) {
	next := int(ui.focus) + delta
	if next < 0 {
		next = int(tviewFocusShortcuts)
	}
	if next > int(tviewFocusShortcuts) {
		next = 0
	}
	ui.setFocusedPanel(tviewFocusPanel(next))
}

func (ui *tviewUI) setFocusedPanel(panel tviewFocusPanel) {
	ui.focus = panel
	ui.sidebar.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.detail.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.files.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.footer.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	switch panel {
	case tviewFocusDetails:
		ui.detail.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.detail)
	case tviewFocusFiles:
		ui.files.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.files)
	case tviewFocusShortcuts:
		ui.footer.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.footer)
	default:
		ui.sidebar.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.sidebar)
	}
}

func (ui *tviewUI) moveFocused(delta int) {
	switch ui.focus {
	case tviewFocusFiles:
		row, _ := ui.files.GetSelection()
		maxRow := len(ui.visibleFiles())
		if maxRow < 1 {
			maxRow = 1
		}
		row += delta
		if row < 1 {
			row = maxRow
		}
		if row > maxRow {
			row = 1
		}
		ui.files.Select(row, 0)
	default:
		ui.move(delta)
	}
}

func (ui *tviewUI) move(delta int) {
	next := int(ui.screen) + delta
	if next < 0 {
		next = len(tviewRoutes) - 1
	}
	if next >= len(tviewRoutes) {
		next = 0
	}
	ui.open(tviewRoutes[next].screen)
}

func (ui *tviewUI) open(screen tviewScreen) {
	ui.screen = screen
	ui.refresh()
}

func (ui *tviewUI) startEdit() {
	switch ui.screen {
	case tviewScreenProject:
		ui.openProjectEdit()
	case tviewScreenServices:
		ui.openServicesEdit()
	case tviewScreenEntities:
		ui.openEntitiesEdit()
	case tviewScreenValueObjects:
		ui.openValueObjectsEdit()
	default:
		ui.message = "Open Project, Services, Entities, or Value Objects before editing."
		ui.refresh()
	}
}

func (ui *tviewUI) openEntitiesEdit() {
	service, ok := firstService(ui.plan.Config.Services)
	if !ok {
		ui.closeEditWithMessage("No services are available to edit entities.")
		return
	}
	ui.openEntitiesEditForService(service.Name)
}

func (ui *tviewUI) openEntitiesEditForService(serviceName string) {
	service, ok := serviceSummaryByName(ui.plan.Config.Services, serviceName)
	if !ok {
		service, ok = firstService(ui.plan.Config.Services)
		if !ok {
			ui.closeEditWithMessage("No services are available to edit entities.")
			return
		}
	}
	ui.showEntitiesManager(tviewEntitiesStateFromService(service))
}

func (ui *tviewUI) openEntitiesEditForServiceAndEntity(serviceName string, selectedEntityIndex int) {
	service, ok := serviceSummaryByName(ui.plan.Config.Services, serviceName)
	if !ok {
		ui.openEntitiesEditForService(serviceName)
		return
	}
	state := tviewEntitiesStateFromService(service)
	if selectedEntityIndex >= 0 && selectedEntityIndex < len(state.rows) {
		ui.showFieldsManager(tviewFieldsStateFromEntity(service.Name, service.Entities[selectedEntityIndex]))
		return
	}
	ui.showEntitiesManager(state)
}

func (ui *tviewUI) saveEntitiesEdit(state *tviewEntitiesEditState) {
	if ui.updateEntities == nil {
		ui.closeEditWithMessage("Entity editing is not available.")
		return
	}
	settings := application.EntitySettings{ServiceName: state.serviceName, Entities: make([]application.EntityNameSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		settings.Entities = append(settings.Entities, application.EntityNameSetting{OriginalName: row.original, Name: strings.TrimSpace(row.name)})
	}
	result, err := ui.updateEntities(ui.request, settings)
	ui.applyEntitiesSaveResult(result, err)
}

func (ui *tviewUI) saveFieldsEdit(state *tviewFieldsEditState) {
	if ui.updateFields == nil {
		ui.closeEditWithMessage("Field editing is not available.")
		return
	}
	result, err := ui.saveFieldsState(state)
	ui.applyFieldsSaveResult(result, err)
}

func (ui *tviewUI) saveFieldsState(state *tviewFieldsEditState) (application.UpdateFieldSettingsResult, error) {
	settings := application.FieldSettings{ServiceName: state.serviceName, EntityName: state.entityName, Fields: make([]application.FieldSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		settings.Fields = append(settings.Fields, application.FieldSetting{OriginalName: row.original, Name: strings.TrimSpace(row.name), Type: trimmedDefault(row.typeName, "string")})
	}
	return ui.updateFields(ui.request, settings)
}

func (ui *tviewUI) applyEntitiesSaveResult(result application.UpdateEntitySettingsResult, err error) {
	if err != nil {
		ui.closeEditWithError("Entities save failed.", err)
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.markPlanStale("Entities saved, but plan refresh failed.", result.PlanError)
		ui.closeEdit()
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.closeEdit()
	ui.refreshPlanAfterSave("Entities saved.")
}

func (ui *tviewUI) applyFieldsSaveResult(result application.UpdateFieldSettingsResult, err error) {
	if err != nil {
		ui.markPlanStale("Entities saved, but fields save failed.", err)
		ui.closeEdit()
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.markPlanStale("Fields saved, but plan refresh failed.", result.PlanError)
		ui.closeEdit()
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.closeEdit()
	ui.refreshPlanAfterSave("Entities and fields saved.")
}

func (ui *tviewUI) openValueObjectsEdit() {
	service, ok := firstService(ui.plan.Config.Services)
	if !ok {
		ui.closeEditWithMessage("No services are available to edit value objects.")
		return
	}
	ui.openValueObjectsEditForService(service.Name)
}

func (ui *tviewUI) openValueObjectsEditForService(serviceName string) {
	service, ok := serviceSummaryByName(ui.plan.Config.Services, serviceName)
	if !ok {
		service, ok = firstService(ui.plan.Config.Services)
		if !ok {
			ui.closeEditWithMessage("No services are available to edit value objects.")
			return
		}
	}
	ui.showValueObjectsManager(tviewValueObjectsStateFromService(service))
}

func (ui *tviewUI) saveValueObjectsEdit(state *tviewValueObjectsEditState) {
	if ui.updateValueObjects == nil {
		ui.closeEditWithMessage("Value object editing is not available.")
		return
	}
	result, err := ui.updateValueObjects(ui.request, tviewValueObjectSettingsFromState(state))
	ui.applyValueObjectsSaveResult(result, err)
}

func tviewValueObjectSettingsFromState(state *tviewValueObjectsEditState) application.ValueObjectSettings {
	settings := application.ValueObjectSettings{ServiceName: state.serviceName, ValueObjects: make([]application.ValueObjectNameSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		settings.ValueObjects = append(settings.ValueObjects, application.ValueObjectNameSetting{OriginalName: row.original, Name: strings.TrimSpace(row.name), Type: trimmedDefault(row.typeName, "string"), Validations: row.validations})
	}
	return settings
}

func (ui *tviewUI) applyValueObjectsSaveResult(result application.UpdateValueObjectSettingsResult, err error) {
	if err != nil {
		ui.closeEditWithError("Value objects save failed.", err)
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.markPlanStale("Value objects saved, but plan refresh failed.", result.PlanError)
		ui.closeEdit()
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.closeEdit()
	ui.refreshPlanAfterSave("Value objects saved.")
}

func (ui *tviewUI) openProjectEdit() {
	form := tview.NewForm()
	addEditInputField(form, "Solution name", ui.plan.Config.SolutionName)
	addEditInputField(form, "Description", ui.plan.Config.SolutionDescription)
	addEditInputField(form, "Target framework", ui.plan.Config.TargetFramework)
	addEditInputField(form, "Solution format", ui.plan.Config.SolutionFormat)
	form.AddCheckbox("Gateway enabled", ui.plan.Config.GatewayEnabled, nil)
	form.AddButton("Save", func() { ui.saveProjectEdit(form) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, " Edit Project ")
}

func (ui *tviewUI) saveProjectEdit(form *tview.Form) {
	if ui.updateSettings == nil {
		ui.closeEditWithMessage("Project editing is not available.")
		return
	}
	gateway := form.GetFormItem(4).(*tview.Checkbox).IsChecked()
	result, err := ui.updateSettings(ui.request, application.SolutionSettings{
		SolutionName:        formInputText(form, 0),
		SolutionDescription: formInputText(form, 1),
		TargetFramework:     formInputText(form, 2),
		SolutionFormat:      formInputText(form, 3),
		GatewayEnabled:      &gateway,
	})
	ui.applyProjectSaveResult(result, err)
}

func (ui *tviewUI) applyProjectSaveResult(result application.UpdateSolutionSettingsResult, err error) {
	if err != nil {
		ui.closeEditWithError("Project save failed.", err)
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.markPlanStale("Project saved, but plan refresh failed.", result.PlanError)
		ui.closeEdit()
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.closeEdit()
	ui.refreshPlanAfterSave("Project saved.")
}

func (ui *tviewUI) openServicesEdit() {
	ui.showServicesManager(tviewServicesStateFromPlan(ui.plan))
}

func (ui *tviewUI) saveServicesEdit(state *tviewServicesEditState) {
	if ui.updateServices == nil {
		ui.closeEditWithMessage("Service editing is not available.")
		return
	}
	settings := application.ServiceSettings{Services: make([]application.ServiceNameSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		settings.Services = append(settings.Services, application.ServiceNameSetting{OriginalName: row.original, Name: strings.TrimSpace(row.name)})
	}
	result, err := ui.updateServices(ui.request, settings)
	ui.applyServicesSaveResult(result, err)
}

func (ui *tviewUI) applyServicesSaveResult(result application.UpdateServiceSettingsResult, err error) {
	if err != nil {
		ui.closeEditWithError("Services save failed.", err)
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.markPlanStale("Services saved, but plan refresh failed.", result.PlanError)
		ui.closeEdit()
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.closeEdit()
	ui.refreshPlanAfterSave("Services saved.")
}

func (ui *tviewUI) showServicesManager(state *tviewServicesEditState) {
	manager := ui.newManagerTable([]string{"Name", "Actions"})
	render := func() { renderManagerRows(manager, state.rows, false, "") }
	render()
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 1 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		if column == 0 {
			ui.openNameEdit(" Edit Service ", state.rows[row-1].name, func(name string) {
				state.rows[row-1].name = name
				ui.showServicesManager(state)
			})
		}
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveServicesEdit(state)
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewService", len(state.rows)+1)})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "d":
			toggleSelectedManagerRow(manager, state.rows)
			render()
			return nil
		}
		return event
	})
	ui.showManagerModal(" Services ", nil, manager, managerFooter("Tab focus  Up/Down move  a add  d delete  enter edit  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewService", len(state.rows)+1)})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Save", func() { ui.saveServicesEdit(state) }}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) showEntitiesManager(state *tviewEntitiesEditState) {
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	serviceSelector := tview.NewDropDown().SetLabel("Service ").SetOptions(serviceNames, func(_ string, index int) {
		if index >= 0 && index < len(serviceNames) && serviceNames[index] != state.serviceName {
			ui.openEntitiesEditForService(serviceNames[index])
		}
	})
	serviceSelector.SetCurrentOption(selectedIndex(serviceNames, state.serviceName))
	manager := ui.newManagerTable([]string{"Name", "Actions"})
	render := func() { renderManagerRows(manager, state.rows, false, "") }
	render()
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 1 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		if column == 0 {
			ui.openNameEdit(" Edit Entity ", state.rows[row-1].name, func(name string) {
				state.rows[row-1].name = name
				ui.showEntitiesManager(state)
			})
		}
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveEntitiesEdit(state)
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewEntity", len(state.rows)+1)})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "d":
			toggleSelectedManagerRow(manager, state.rows)
			render()
			return nil
		case "f":
			row, _ := manager.GetSelection()
			ui.openFieldsFromEntityState(state, row-1)
			return nil
		}
		return event
	})
	ui.showManagerModal(" Entities ", serviceSelector, manager, managerFooter("Tab focus  Up/Down move  a add  d delete  enter edit/select  f fields  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewEntity", len(state.rows)+1)})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Fields", func() {
		row, _ := manager.GetSelection()
		ui.openFieldsFromEntityState(state, row-1)
	}}, {"Save", func() { ui.saveEntitiesEdit(state) }}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) openFieldsFromEntityState(state *tviewEntitiesEditState, index int) {
	if index < 0 || index >= len(state.rows) || state.rows[index].original == "" || state.rows[index].deleted {
		ui.message = "Save the entity before editing fields."
		ui.refresh()
		return
	}
	service, ok := serviceSummaryByName(ui.plan.Config.Services, state.serviceName)
	if !ok {
		return
	}
	for _, entity := range service.Entities {
		if entity.Name == state.rows[index].original {
			ui.showFieldsManager(tviewFieldsStateFromEntity(state.serviceName, entity))
			return
		}
	}
}

func (ui *tviewUI) showFieldsManager(state *tviewFieldsEditState) {
	context := tview.NewTextView().SetDynamicColors(true).SetText(fmt.Sprintf("[aqua]%s[white] / [yellow]%s[white]", state.serviceName, state.entityName))
	manager := ui.newManagerTable([]string{"Name", "Type", "Actions"})
	render := func() { renderManagerRows(manager, state.rows, true, "") }
	render()
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 2 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		ui.openFieldEdit(" Edit Field ", state.rows[row-1], func(updated tviewManagerRow) {
			state.rows[row-1].name = updated.name
			state.rows[row-1].typeName = updated.typeName
			ui.showFieldsManager(state)
		})
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveFieldsEdit(state)
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewField", len(state.rows)+1), typeName: "string"})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "d":
			toggleSelectedManagerRow(manager, state.rows)
			render()
			return nil
		}
		return event
	})
	ui.showManagerModal(" Fields ", context, manager, managerFooter("Tab focus  Up/Down move  a add  d delete  enter edit  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewField", len(state.rows)+1), typeName: "string"})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Save", func() { ui.saveFieldsEdit(state) }}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) showValueObjectsManager(state *tviewValueObjectsEditState) {
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	serviceSelector := tview.NewDropDown().SetLabel("Service ").SetOptions(serviceNames, func(_ string, index int) {
		if index >= 0 && index < len(serviceNames) && serviceNames[index] != state.serviceName {
			ui.openValueObjectsEditForService(serviceNames[index])
		}
	})
	serviceSelector.SetCurrentOption(selectedIndex(serviceNames, state.serviceName))
	manager := ui.newManagerTable([]string{"Name", "Type", "Actions"})
	render := func() { renderManagerRows(manager, state.rows, true, "") }
	render()
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 2 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		ui.openFieldEdit(" Edit Value Object ", state.rows[row-1], func(updated tviewManagerRow) {
			state.rows[row-1].name = updated.name
			state.rows[row-1].typeName = updated.typeName
			ui.showValueObjectsManager(state)
		})
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveValueObjectsEdit(state)
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewValueObject", len(state.rows)+1), typeName: "string"})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "d":
			toggleSelectedManagerRow(manager, state.rows)
			render()
			return nil
		case "r":
			row, _ := manager.GetSelection()
			ui.openRulesFromValueObjectState(state, row-1)
			return nil
		}
		return event
	})
	ui.showManagerModal(" Value Objects ", serviceSelector, manager, managerFooter("Tab focus  Up/Down move  a add  d delete  enter edit/select  r rules  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewValueObject", len(state.rows)+1), typeName: "string"})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Rules", func() {
		row, _ := manager.GetSelection()
		ui.openRulesFromValueObjectState(state, row-1)
	}}, {"Save", func() { ui.saveValueObjectsEdit(state) }}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) openRulesFromValueObjectState(state *tviewValueObjectsEditState, index int) {
	if index < 0 || index >= len(state.rows) || state.rows[index].deleted {
		return
	}
	row := state.rows[index]
	ui.showValueObjectRulesManager(&tviewValueObjectRulesEditState{
		serviceName: state.serviceName,
		valueObject: application.ValueObjectSummary{Name: strings.TrimSpace(row.name), Type: trimmedDefault(row.typeName, "string")},
		rows:        append([]tviewManagerRow(nil), state.rows...),
		validations: row.validations,
	})
}

func (ui *tviewUI) showValueObjectRulesManager(state *tviewValueObjectRulesEditState) {
	form := tview.NewForm()
	form.SetBorder(false)
	form.SetFieldTextColor(tcell.ColorWhite).SetLabelColor(tcell.ColorLightCyan).SetButtonTextColor(tcell.ColorBlack).SetButtonBackgroundColor(tcell.ColorTeal)
	required := boolValue(state.validations.Required)
	notEmpty := boolValue(state.validations.NotEmpty)
	notDefault := boolValue(state.validations.NotDefault)
	form.AddCheckbox("Required", required, func(checked bool) { state.validations.Required = &checked })
	form.AddInputField("Min length", intString(state.validations.MinLength), tviewEditModalInputWidth, nil, func(value string) { state.validations.MinLength = intPointerFromText(value) })
	form.AddInputField("Max length", intString(state.validations.MaxLength), tviewEditModalInputWidth, nil, func(value string) { state.validations.MaxLength = intPointerFromText(value) })
	form.AddInputField("Pattern", stringValue(state.validations.Pattern), tviewEditModalInputWidth, nil, func(value string) { state.validations.Pattern = stringPointerFromText(value) })
	form.AddInputField("Valid example", stringValue(state.validations.ValidExample), tviewEditModalInputWidth, nil, func(value string) { state.validations.ValidExample = stringPointerFromText(value) })
	form.AddInputField("Invalid example", stringValue(state.validations.InvalidExample), tviewEditModalInputWidth, nil, func(value string) { state.validations.InvalidExample = stringPointerFromText(value) })
	form.AddInputField("Minimum", stringValue(state.validations.Minimum), tviewEditModalInputWidth, nil, func(value string) { state.validations.Minimum = stringPointerFromText(value) })
	form.AddInputField("Maximum", stringValue(state.validations.Maximum), tviewEditModalInputWidth, nil, func(value string) { state.validations.Maximum = stringPointerFromText(value) })
	form.AddCheckbox("Not empty", notEmpty, func(checked bool) { state.validations.NotEmpty = &checked })
	form.AddCheckbox("Not default", notDefault, func(checked bool) { state.validations.NotDefault = &checked })
	form.AddButton("Save", func() { ui.saveValueObjectRulesEdit(state, form) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, fmt.Sprintf(" Rules: %s/%s ", state.serviceName, state.valueObject.Name))
}

func (ui *tviewUI) saveValueObjectRulesEdit(state *tviewValueObjectRulesEditState, form *tview.Form) {
	if ui.updateValueObjects == nil {
		ui.closeEditWithMessage("Value object editing is not available.")
		return
	}
	validations := application.ValidationRuleSettings{
		Required:       boolPointerFromCheckbox(form, 0),
		MinLength:      intPointerFromForm(form, 1),
		MaxLength:      intPointerFromForm(form, 2),
		Pattern:        stringPointerFromForm(form, 3),
		ValidExample:   stringPointerFromForm(form, 4),
		InvalidExample: stringPointerFromForm(form, 5),
		Minimum:        stringPointerFromForm(form, 6),
		Maximum:        stringPointerFromForm(form, 7),
		NotEmpty:       boolPointerFromCheckbox(form, 8),
		NotDefault:     boolPointerFromCheckbox(form, 9),
	}
	valueObjectName := strings.TrimSpace(state.valueObject.Name)
	for index := range state.rows {
		if strings.TrimSpace(state.rows[index].name) == valueObjectName {
			state.rows[index].validations = validations
			break
		}
	}
	result, err := ui.updateValueObjects(ui.request, tviewValueObjectSettingsFromState(&tviewValueObjectsEditState{serviceName: state.serviceName, rows: state.rows}))
	ui.applyValueObjectsSaveResult(result, err)
}

func (ui *tviewUI) showEditForm(form *tview.Form, title string) {
	form.SetBorder(false)
	form.SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorLightCyan).
		SetButtonTextColor(tcell.ColorBlack).
		SetButtonBackgroundColor(tcell.ColorTeal)
	form.SetCancelFunc(ui.closeEdit)
	panel := tview.NewFrame(form).
		SetBorders(1, 1, 0, 1, 2, 2).
		AddText("Tab next  Shift+Tab previous  Enter activate  Esc cancel", false, tview.AlignCenter, tcell.ColorGray)
	panel.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	panel.SetBorderColor(tcell.ColorTeal).SetTitleColor(tcell.ColorLightCyan)
	modal := centerPrimitive(panel, 82, 18)
	ui.editOpen = true
	ui.root.RemovePage(tviewEditModalPage)
	ui.root.AddPage(tviewEditModalPage, modal, true, true)
	ui.app.SetFocus(panel)
}

type managerButton struct {
	label string
	run   func()
}

func (ui *tviewUI) newManagerTable(headers []string) *tview.Table {
	table := tview.NewTable().SetSelectable(true, true).SetFixed(1, 0)
	table.SetBorder(false)
	table.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal).Bold(true))
	for column, header := range headers {
		table.SetCell(0, column, tview.NewTableCell(header).SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(1))
	}
	table.Select(1, 0)
	return table
}

func renderManagerRows(table *tview.Table, rows []tviewManagerRow, withType bool, actionLabel string) {
	for table.GetRowCount() > 1 {
		table.RemoveRow(1)
	}
	if len(rows) == 0 {
		table.SetCell(1, 0, tview.NewTableCell("No rows yet. Press a to add one.").SetTextColor(tcell.ColorGray).SetExpansion(2))
		return
	}
	for index, row := range rows {
		textColor := tcell.ColorWhite
		action := "Delete"
		if row.deleted {
			textColor = tcell.ColorGray
			action = "Restore"
		}
		table.SetCell(index+1, 0, tview.NewTableCell(emptyDash(strings.TrimSpace(row.name))).SetTextColor(textColor).SetExpansion(2))
		if withType {
			table.SetCell(index+1, 1, tview.NewTableCell(trimmedDefault(row.typeName, "string")).SetTextColor(textColor).SetExpansion(1))
			if actionLabel != "" && !row.deleted {
				action += " | " + actionLabel
			}
			table.SetCell(index+1, 2, tview.NewTableCell(action).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignRight).SetExpansion(1))
		} else {
			table.SetCell(index+1, 1, tview.NewTableCell(action).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignRight).SetExpansion(1))
		}
	}
}

func managerFooter(shortcuts string, buttons []managerButton) *tview.Flex {
	form := tview.NewForm()
	form.SetBorder(false)
	form.SetButtonTextColor(tcell.ColorBlack).SetButtonBackgroundColor(tcell.ColorTeal)
	for _, button := range buttons {
		form.AddButton(button.label, button.run)
	}
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetDynamicColors(true).SetText("[gray]"+shortcuts), 1, 0, false).
		AddItem(form, 1, 0, false)
}

func (ui *tviewUI) showManagerModal(title string, header tview.Primitive, manager *tview.Table, footer *tview.Flex) {
	body := tview.NewFlex().SetDirection(tview.FlexRow)
	if header != nil {
		body.AddItem(header, 2, 0, false)
	}
	body.AddItem(manager, 0, 1, true).AddItem(footer, 3, 0, false)
	panel := tview.NewFrame(body).SetBorders(1, 1, 0, 1, 1, 1)
	panel.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	panel.SetBorderColor(tcell.ColorTeal).SetTitleColor(tcell.ColorLightCyan)
	modal := centerPrimitive(panel, 86, 22)
	ui.editOpen = true
	ui.root.RemovePage(tviewEditModalPage)
	ui.root.AddPage(tviewEditModalPage, modal, true, true)
	if header != nil {
		ui.app.SetFocus(header)
		return
	}
	ui.app.SetFocus(manager)
}

func (ui *tviewUI) openNameEdit(title, value string, save func(string)) {
	form := tview.NewForm()
	addEditInputField(form, "Name", value)
	form.AddButton("Apply", func() { save(formInputText(form, 0)) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, title)
}

func (ui *tviewUI) openFieldEdit(title string, row tviewManagerRow, save func(tviewManagerRow)) {
	form := tview.NewForm()
	addEditInputField(form, "Name", row.name)
	addEditInputField(form, "Type", trimmedDefault(row.typeName, "string"))
	form.AddButton("Apply", func() {
		row.name = formInputText(form, 0)
		row.typeName = formInputText(form, 1)
		save(row)
	})
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, title)
}

func tviewServicesStateFromPlan(plan application.GenerationPlan) *tviewServicesEditState {
	services := plan.Config.ServiceNames
	if len(services) == 0 && len(plan.Config.Services) > 0 {
		services = serviceSummaryNames(plan.Config.Services)
	}
	state := &tviewServicesEditState{rows: make([]tviewManagerRow, 0, len(services))}
	for _, name := range services {
		state.rows = append(state.rows, tviewManagerRow{original: name, name: name})
	}
	return state
}

func tviewEntitiesStateFromService(service application.ServiceSummary) *tviewEntitiesEditState {
	state := &tviewEntitiesEditState{serviceName: service.Name, rows: make([]tviewManagerRow, 0, len(service.Entities))}
	for _, entity := range service.Entities {
		state.rows = append(state.rows, tviewManagerRow{original: entity.Name, name: entity.Name})
	}
	return state
}

func tviewFieldsStateFromEntity(serviceName string, entity application.EntitySummary) *tviewFieldsEditState {
	state := &tviewFieldsEditState{serviceName: serviceName, entityName: entity.Name, rows: make([]tviewManagerRow, 0, len(entity.Fields))}
	for _, field := range entity.Fields {
		state.rows = append(state.rows, tviewManagerRow{original: field.Name, name: field.Name, typeName: field.Type})
	}
	return state
}

func tviewValueObjectsStateFromService(service application.ServiceSummary) *tviewValueObjectsEditState {
	state := &tviewValueObjectsEditState{serviceName: service.Name, rows: make([]tviewManagerRow, 0, len(service.ValueObjects))}
	for _, valueObject := range service.ValueObjects {
		state.rows = append(state.rows, tviewManagerRow{original: valueObject.Name, name: valueObject.Name, typeName: valueObject.Type, validations: validationRuleSettingsFromSummary(valueObject.Validations)})
	}
	return state
}

func toggleSelectedManagerRow(table *tview.Table, rows []tviewManagerRow) {
	row, _ := table.GetSelection()
	if row <= 0 || row > len(rows) {
		return
	}
	rows[row-1].deleted = !rows[row-1].deleted
}

func nextNamedRow(prefix string, count int) string {
	return fmt.Sprintf("%s%d", prefix, count)
}

func trimmedDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intString(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func stringPointerFromText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func intPointerFromText(value string) *int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil
	}
	return &parsed
}

func stringPointerFromForm(form *tview.Form, index int) *string {
	return stringPointerFromText(formInputText(form, index))
}

func intPointerFromForm(form *tview.Form, index int) *int {
	return intPointerFromText(formInputText(form, index))
}

func boolPointerFromCheckbox(form *tview.Form, index int) *bool {
	checked := formCheckboxChecked(form, index)
	return &checked
}

func (ui *tviewUI) closeEdit() {
	ui.editOpen = false
	ui.root.RemovePage(tviewEditModalPage)
	ui.setFocusedPanel(ui.focus)
	ui.refresh()
}

func centerPrimitive(primitive tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(primitive, width, 0, true).
			AddItem(nil, 0, 1, false), height, 0, true).
		AddItem(nil, 0, 1, false)
}

func (ui *tviewUI) closeEditWithMessage(message string) {
	ui.message = message
	ui.err = nil
	ui.closeEdit()
}

func (ui *tviewUI) closeEditWithError(message string, err error) {
	ui.message = message
	ui.err = err
	ui.closeEdit()
}

func (ui *tviewUI) refreshPlanAfterSave(message string) {
	if ui.planFunc == nil {
		ui.message = message
		ui.refresh()
		return
	}
	plan, err := ui.planFunc(ui.request)
	if err != nil {
		ui.message = message + " Plan refresh failed."
		ui.err = err
		ui.planStale = true
		ui.refresh()
		return
	}
	ui.plan = plan
	ui.err = nil
	ui.planStale = false
	ui.message = message
	ui.refresh()
}

func formInputText(form *tview.Form, index int) string {
	return strings.TrimSpace(form.GetFormItem(index).(*tview.InputField).GetText())
}

func formCheckboxChecked(form *tview.Form, index int) bool {
	return form.GetFormItem(index).(*tview.Checkbox).IsChecked()
}

func addEditInputField(form *tview.Form, label, value string) {
	form.AddInputField(label, value, tviewEditModalInputWidth, nil, nil)
}

func (ui *tviewUI) markPlanStale(message string, err error) {
	ui.planStale = true
	ui.message = message
	ui.err = err
}

func firstServiceWithEntities(services []application.ServiceSummary) (application.ServiceSummary, bool) {
	for _, service := range services {
		if len(service.Entities) > 0 {
			return service, true
		}
	}
	return application.ServiceSummary{}, false
}

func firstService(services []application.ServiceSummary) (application.ServiceSummary, bool) {
	if len(services) == 0 {
		return application.ServiceSummary{}, false
	}
	return services[0], true
}

func serviceSummaryByName(services []application.ServiceSummary, name string) (application.ServiceSummary, bool) {
	for _, service := range services {
		if service.Name == name {
			return service, true
		}
	}
	return application.ServiceSummary{}, false
}

func serviceSummaryNames(services []application.ServiceSummary) []string {
	names := make([]string, len(services))
	for index, service := range services {
		names[index] = service.Name
	}
	return names
}

func entitySummaryNames(entities []application.EntitySummary) []string {
	if len(entities) == 0 {
		return []string{"No entity selected"}
	}
	names := make([]string, len(entities))
	for index, entity := range entities {
		names[index] = entity.Name
	}
	return names
}

func selectedIndex(options []string, selected string) int {
	for index, option := range options {
		if option == selected {
			return index
		}
	}
	return 0
}

func firstServiceWithValueObjects(services []application.ServiceSummary) (application.ServiceSummary, bool) {
	for _, service := range services {
		if len(service.ValueObjects) > 0 {
			return service, true
		}
	}
	return application.ServiceSummary{}, false
}

func validationRuleSettingsFromSummary(summary application.ValidationRuleSummary) application.ValidationRuleSettings {
	return application.ValidationRuleSettings{
		Required:       cloneBool(summary.Required),
		MinLength:      cloneInt(summary.MinLength),
		MaxLength:      cloneInt(summary.MaxLength),
		Pattern:        cloneString(summary.Pattern),
		ValidExample:   cloneString(summary.ValidExample),
		InvalidExample: cloneString(summary.InvalidExample),
		Minimum:        cloneString(summary.Minimum),
		Maximum:        cloneString(summary.Maximum),
		NotEmpty:       cloneBool(summary.NotEmpty),
		NotDefault:     cloneBool(summary.NotDefault),
	}
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (ui *tviewUI) refreshPlan() {
	if ui.planFunc == nil {
		ui.message = "Plan refresh is not available."
		ui.refresh()
		return
	}
	plan, err := ui.planFunc(ui.request)
	if err != nil {
		ui.err = err
		ui.message = "Plan refresh failed."
		ui.refresh()
		return
	}
	ui.plan = plan
	ui.err = nil
	ui.planStale = false
	ui.message = "Plan refreshed."
	ui.refresh()
}

func (ui *tviewUI) generateFiles() {
	if ui.generate == nil {
		ui.message = "Generate action is not available."
		ui.refresh()
		return
	}
	if (ui.plan.ForceRequired || ui.plan.Readiness.OutputForceRequired) && !ui.request.Force {
		ui.message = "Generation is locked until --force is confirmed for this existing output."
		ui.open(tviewScreenGenerate)
		return
	}
	if ui.planStale {
		ui.message = "Generation is blocked until the plan refreshes successfully."
		ui.open(tviewScreenGenerate)
		return
	}
	result, err := ui.generate(ui.request)
	if err != nil {
		ui.err = err
		ui.message = "Generation failed."
		ui.open(tviewScreenResult)
		return
	}
	ui.result = result
	ui.plan = result.Plan
	ui.err = nil
	ui.planStale = false
	ui.message = fmt.Sprintf("Generated %d files in %s.", result.Plan.FileCount, result.OutputDir)
	if result.Warning != "" {
		ui.message += " Warning: " + result.Warning
	}
	ui.open(tviewScreenResult)
}

func (ui *tviewUI) refresh() {
	ui.renderSidebar()
	ui.renderDetail()
	ui.renderFiles()
	ui.renderFooter()
}

func (ui *tviewUI) renderSidebar() {
	ui.sidebar.Clear()
	for row, route := range tviewRoutes {
		prefix := "  "
		color := tcell.ColorGray
		if route.screen == ui.screen {
			prefix = "> "
			color = tcell.ColorLightCyan
		}
		ui.sidebar.SetCell(row, 0, tview.NewTableCell(prefix+route.label).SetTextColor(color).SetExpansion(1))
	}
	ui.sidebar.Select(int(ui.screen), 0)
}

func (ui *tviewUI) renderDetail() {
	ui.detail.SetTitle(" " + ui.screenTitle() + " ")
	ui.detail.SetText(strings.Join(ui.detailLines(), "\n"))
}

func (ui *tviewUI) renderFiles() {
	ui.files.Clear()
	ui.files.SetCell(0, 0, tview.NewTableCell("Action").SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(1))
	ui.files.SetCell(0, 1, tview.NewTableCell("Path").SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(4))
	for row, file := range ui.visibleFiles() {
		color := tcell.ColorWhite
		switch strings.ToLower(file.Action) {
		case "create", "write":
			color = tcell.ColorGreen
		case "replace", "delete":
			color = tcell.ColorYellow
		}
		ui.files.SetCell(row+1, 0, tview.NewTableCell(file.Action).SetTextColor(color).SetExpansion(1))
		ui.files.SetCell(row+1, 1, tview.NewTableCell(file.Path).SetTextColor(tcell.ColorWhite).SetExpansion(4))
	}
	if len(ui.visibleFiles()) == 0 {
		ui.files.SetCell(1, 0, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
		ui.files.SetCell(1, 1, tview.NewTableCell("No planned files yet.").SetTextColor(tcell.ColorGray))
	}
}

func (ui *tviewUI) renderFooter() {
	shortcuts := "[aqua]tab/shift+tab[white] focus  [aqua]j/k[white] move  [aqua]enter[white] open  [aqua]e[white] edit  [aqua]r[white] refresh  [aqua]g[white] generate  [aqua]esc[white] overview  [aqua]q[white] quit"
	if (ui.plan.ForceRequired || ui.plan.Readiness.OutputForceRequired) && !ui.request.Force {
		shortcuts += "  [yellow]--force required"
	}
	ui.footer.SetText(shortcuts)
}

func (ui *tviewUI) screenTitle() string {
	for _, route := range tviewRoutes {
		if route.screen == ui.screen {
			return route.label
		}
	}
	return "Overview"
}

func (ui *tviewUI) detailLines() []string {
	lines := []string{
		"[aqua]" + ui.plan.Config.SolutionName + "[white]",
		fmt.Sprintf("Output: [gray]%s[white]", emptyDash(ui.plan.OutputDir)),
		fmt.Sprintf("Action: [yellow]%s[white]  Files: [green]%d[white]", emptyDash(ui.plan.OutputAction), ui.plan.FileCount),
		fmt.Sprintf("Services: %d  Entities: %d  Value Objects: %d", ui.plan.Config.ServiceCount, ui.plan.Config.EntityCount, ui.plan.Config.ValueObjectCount),
	}
	if ui.plan.ForceRequired || ui.plan.Readiness.OutputForceRequired {
		lines = append(lines, "[yellow]This output needs --force before generation.[white]")
	}
	if ui.message != "" {
		lines = append(lines, "", "[green]"+ui.message+"[white]")
	}
	if ui.planStale {
		lines = append(lines, "[yellow]The plan must refresh successfully before generation.[white]")
	}
	if ui.err != nil {
		lines = append(lines, "[red]"+ui.err.Error()+"[white]")
	}

	switch ui.screen {
	case tviewScreenProject:
		lines = append(lines, "", fmt.Sprintf("Description: %s", emptyDash(ui.plan.Config.SolutionDescription)), fmt.Sprintf("Target Framework: %s", emptyDash(ui.plan.Config.TargetFramework)), fmt.Sprintf("Solution Format: %s", emptyDash(ui.plan.Config.SolutionFormat)), fmt.Sprintf("Gateway: %t", ui.plan.Config.GatewayEnabled), "Press e to edit project settings in this dashboard.")
	case tviewScreenServices:
		lines = append(lines, "", "[aqua]Services[white]", "Press e to rename services in this dashboard.")
		lines = append(lines, prefixedList(ui.plan.Config.ServiceNames)...)
	case tviewScreenEntities:
		lines = append(lines, "", "[aqua]Entities[white]", "Press e to edit entities and fields in this dashboard.")
		lines = append(lines, entityLines(ui.plan.Config.Services)...)
	case tviewScreenValueObjects:
		lines = append(lines, "", "[aqua]Value Objects[white]", "Press e to edit value objects in this dashboard.")
		lines = append(lines, valueObjectLines(ui.plan.Config.Services)...)
	case tviewScreenPreview:
		lines = append(lines, "", "Review the planned file table before generating.")
	case tviewScreenGenerate:
		lines = append(lines, "", ui.generateSummary())
	case tviewScreenResult:
		lines = append(lines, "", ui.resultSummary())
	default:
		lines = append(lines, "", readinessLines(ui.plan.Readiness))
	}
	return lines
}

func (ui *tviewUI) visibleFiles() []application.PlannedFile {
	if ui.screen != tviewScreenPreview && ui.screen != tviewScreenGenerate && ui.screen != tviewScreenResult {
		limit := len(ui.plan.Files)
		if limit > 12 {
			limit = 12
		}
		return ui.plan.Files[:limit]
	}
	return ui.plan.Files
}

func (ui *tviewUI) generateSummary() string {
	if (ui.plan.ForceRequired || ui.plan.Readiness.OutputForceRequired) && !ui.request.Force {
		return "[yellow]Generation is locked. Re-run with --force to replace the verified generated directory.[white]"
	}
	if ui.planStale {
		return "[yellow]Generation is blocked until the plan refreshes successfully.[white]"
	}
	return "Press [aqua]g[white] to generate the planned files."
}

func (ui *tviewUI) resultSummary() string {
	if ui.err != nil {
		return "[red]Generation did not complete. Fix the error and retry with g.[white]"
	}
	if ui.result.OutputDir == "" {
		return "No generation has run in this session yet."
	}
	return fmt.Sprintf("[green]Generated %d files in %s.[white]", ui.result.Plan.FileCount, ui.result.OutputDir)
}

func prefixedList(values []string) []string {
	if len(values) == 0 {
		return []string{"  -"}
	}
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, "  "+value)
	}
	return lines
}

func entityLines(services []application.ServiceSummary) []string {
	var lines []string
	for _, service := range services {
		lines = append(lines, "  [yellow]"+service.Name+"[white]")
		if len(service.Entities) == 0 {
			lines = append(lines, "    -")
			continue
		}
		for _, entity := range service.Entities {
			lines = append(lines, "    "+entity.Name)
		}
	}
	if len(lines) == 0 {
		return []string{"  -"}
	}
	return lines
}

func valueObjectLines(services []application.ServiceSummary) []string {
	var lines []string
	for _, service := range services {
		lines = append(lines, "  [yellow]"+service.Name+"[white]")
		valueObjects := service.ValueObjectNames
		if len(service.ValueObjects) > 0 {
			valueObjects = make([]string, 0, len(service.ValueObjects))
			for _, valueObject := range service.ValueObjects {
				valueObjects = append(valueObjects, valueObject.Name)
			}
		}
		lines = append(lines, prefixedList(valueObjects)...)
	}
	if len(lines) == 0 {
		return []string{"  -"}
	}
	return lines
}

func readinessLines(readiness application.ReadinessSummary) string {
	lines := []string{fmt.Sprintf("Readiness: services=%d entities=%d fields=%d valueObjects=%d", readiness.ServiceCount, readiness.EntityCount, readiness.FieldCount, readiness.ValueObjectCount)}
	if len(readiness.Hints) > 0 {
		lines = append(lines, "", "[yellow]Hints[white]")
		for _, hint := range readiness.Hints {
			lines = append(lines, "  "+hint)
		}
	}
	return strings.Join(lines, "\n")
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
