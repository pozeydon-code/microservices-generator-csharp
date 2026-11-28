package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
	"github.com/rivo/tview"
)

type tviewScreen int

const (
	tviewScreenOverview tviewScreen = iota
	tviewScreenProject
	tviewScreenServices
	tviewScreenEntities
	tviewScreenValueObjects
	tviewScreenRelationships
	tviewScreenPreview
	tviewScreenGenerate
	tviewScreenResult
)

type tviewFocusPanel int

const (
	tviewFocusSidebar tviewFocusPanel = iota
	tviewFocusTree
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
	{tviewScreenRelationships, "Relationships"},
	{tviewScreenPreview, "Preview"},
	{tviewScreenGenerate, "Generate"},
	{tviewScreenResult, "Result"},
}

type tviewUI struct {
	app                        *tview.Application
	root                       *tview.Pages
	dashboard                  tview.Primitive
	sidebar                    *tview.Table
	tree                       *tview.Table
	selection                  workspaceSelection
	workspace                  *tview.Pages
	header                     *tview.TextView
	previewInfo                *tview.TextView
	detail                     *tview.TextView
	files                      *tview.Table
	footer                     *tview.TextView
	plan                       application.GenerationPlan
	request                    application.GenerateRequest
	planFunc                   PlanFunc
	generate                   GenerateFunc
	queueUpdateDraw            func(func())
	stopApp                    func()
	updateSettings             UpdateSettingsFunc
	updateServices             UpdateServicesFunc
	updateEntities             UpdateEntitiesFunc
	updateFields               UpdateFieldsFunc
	updateValueObjects         UpdateValueObjectsFunc
	updateRelationships        UpdateRelationshipsFunc
	targetFrameworkSuggestions []string
	result                     application.GenerateResult
	err                        error
	message                    string
	screen                     tviewScreen
	focus                      tviewFocusPanel
	editOpen                   bool
	generating                 bool
	modalFocus                 []tview.Primitive
	modalFocusIndex            int
	planStale                  bool
	servicePickerReturnFocus   tview.Primitive
	editBack                   func()
	childApply                 func()
	lastClick                  tviewClickTarget
}

const (
	tviewEditModalPage       = "edit-modal"
	tviewServicePickerPage   = "service-picker"
	tviewEditModalInputWidth = 32
)

type tviewManagerRow struct {
	original         string
	name             string
	typeName         string
	hasConnection    bool
	connectionString *string
	deleted          bool
	validations      application.ValidationRuleSettings
}

type tviewServicesEditState struct {
	rows []tviewManagerRow
}

type tviewEntitiesEditState struct {
	serviceName string
	selected    int
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

type tviewRelationshipsEditState struct {
	serviceName string
	rows        []tviewRelationshipRow
}

type tviewRelationshipRow struct {
	originalName        string
	name                string
	multiplicity        string
	principalEntity     string
	dependentEntity     string
	foreignKeyName      string
	foreignKeyType      string
	required            bool
	principalNavigation string
	dependentNavigation string
	deleted             bool
}

type tviewValueObjectRulesEditState struct {
	serviceName string
	valueObject application.ValueObjectSummary
	rows        []tviewManagerRow
	index       int
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
	ui.app.EnableMouse(true)
	ui.queueUpdateDraw = func(fn func()) { ui.app.QueueUpdateDraw(fn) }
	ui.stopApp = func() { ui.app.Stop() }
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
		ui.updateRelationships = tviewUpdateRelationshipsCallback(callbacks[5])
	}
	if len(callbacks) > 6 {
		ui.targetFrameworkSuggestions, _ = callbacks[6].([]string)
	}
	ui.root = ui.build()
	ui.setFocusedPanel(tviewFocusSidebar)
	ui.refresh()
	return ui
}

func tviewUpdateRelationshipsCallback(callback any) UpdateRelationshipsFunc {
	switch fn := callback.(type) {
	case UpdateRelationshipsFunc:
		return fn
	case func(application.GenerateRequest, application.RelationshipSettings) (application.UpdateRelationshipSettingsResult, error):
		return fn
	default:
		return nil
	}
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
	ui.sidebar.SetBorder(true).SetTitle(" Views ").SetTitleAlign(tview.AlignLeft)
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

	ui.previewInfo = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	ui.footer = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	dashboard := ui.buildWorkspace()
	dashboard.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if ui.editOpen {
			return action, nil
		}
		return action, event
	})
	ui.dashboard = dashboard

	ui.app.SetInputCapture(ui.handleKey)
	ui.app.SetMouseCapture(ui.captureMouse)
	return tview.NewPages().AddPage("dashboard", ui.dashboard, true, true)
}

type tviewClickTarget struct {
	x, y  int
	focus tview.Primitive
	at    time.Time
}

func (ui *tviewUI) captureMouse(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
	x, y := event.Position()
	switch action {
	case tview.MouseLeftClick, tview.MouseLeftDoubleClick:
		// TView pairs clicks globally. Re-pair both native actions using the
		// position and the target focused by mouse-down, without losing the
		// first click on a new target.
		click := tviewClickTarget{x: x, y: y, focus: ui.app.GetFocus(), at: time.Now()}
		last := ui.lastClick
		if click.x == last.x && click.y == last.y && click.focus == last.focus && click.at.Sub(last.at) <= tview.DoubleClickInterval {
			ui.lastClick = tviewClickTarget{}
			action = tview.MouseLeftDoubleClick
		} else {
			ui.lastClick = click
			action = tview.MouseLeftClick
		}
	case tview.MouseScrollUp, tview.MouseScrollDown:
		// An open DropDown focuses its List, but its own mouse handler does
		// not forward wheel events. Route before the native mouse capture.
		if list, ok := ui.app.GetFocus().(*tview.List); ok && list.InRect(x, y) {
			list.MouseHandler()(action, event, func(p tview.Primitive) { ui.app.SetFocus(p) })
			return nil, action
		}
	case tview.MouseLeftDown:
		if !ui.editOpen {
			for panel, box := range []interface{ InRect(int, int) bool }{ui.sidebar, ui.tree, ui.detail, ui.files, ui.footer} {
				if ui.panelVisible(tviewFocusPanel(panel)) && box.InRect(x, y) {
					ui.setFocusedPanel(tviewFocusPanel(panel))
					break
				}
			}
		}
	}
	return event, action
}

func (ui *tviewUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if ui.editOpen {
		if event.Key() == tcell.KeyCtrlC {
			ui.requestQuit()
			return nil
		}
		if ui.servicePickerReturnFocus != nil {
			if event.Key() == tcell.KeyEscape {
				ui.closeServicePicker()
				return nil
			}
			return event
		}
		switch event.Key() {
		case tcell.KeyCtrlS:
			if ui.childApply != nil {
				ui.childApply()
				return nil
			}
		case tcell.KeyTAB, tcell.KeyBacktab:
			if len(ui.modalFocus) > 0 {
				delta := 1
				if event.Key() == tcell.KeyBacktab {
					delta = -1
				}
				for i, primitive := range ui.modalFocus {
					if primitive.HasFocus() {
						ui.modalFocusIndex = i
						if form, ok := primitive.(*tview.Form); ok {
							_, button := form.GetFocusedItemIndex()
							if next := button + delta; button >= 0 && next >= 0 && next < form.GetButtonCount() {
								return event
							}
						}
						break
					}
				}
				ui.cycleModalFocus(delta)
				return nil
			}
		case tcell.KeyEscape:
			ui.backFromEdit()
			return nil
		}
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		ui.requestQuit()
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
		ui.requestQuit()
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
	case "f":
		if ui.screen == tviewScreenOverview || ui.screen == tviewScreenEntities {
			ui.editSelectedFields()
		}
		return nil
	}
	return event
}

func (ui *tviewUI) requestQuit() {
	if ui.generating {
		ui.message = "Generation is running. Wait for completion before quitting."
		ui.refresh()
		return
	}
	ui.stopApp()
}

func (ui *tviewUI) cycleModalFocus(delta int) {
	if len(ui.modalFocus) == 0 {
		return
	}
	ui.modalFocusIndex += delta
	if ui.modalFocusIndex < 0 {
		ui.modalFocusIndex = len(ui.modalFocus) - 1
	}
	if ui.modalFocusIndex >= len(ui.modalFocus) {
		ui.modalFocusIndex = 0
	}
	ui.app.SetFocus(ui.modalFocus[ui.modalFocusIndex])
}

func (ui *tviewUI) panelVisible(panel tviewFocusPanel) bool {
	switch panel {
	case tviewFocusDetails:
		return ui.screen != tviewScreenPreview
	case tviewFocusFiles:
		return ui.screen == tviewScreenPreview
	default:
		return true
	}
}

func (ui *tviewUI) cycleFocus(delta int) {
	count := int(tviewFocusShortcuts) + 1
	for next := (int(ui.focus) + delta + count) % count; ; next = (next + delta + count) % count {
		if ui.panelVisible(tviewFocusPanel(next)) {
			ui.setFocusedPanel(tviewFocusPanel(next))
			return
		}
	}
}

func (ui *tviewUI) setFocusedPanel(panel tviewFocusPanel) {
	ui.focus = panel
	ui.sidebar.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.detail.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.files.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	ui.tree.SetBorderColor(tcell.ColorDarkCyan).SetTitleColor(tcell.ColorLightCyan)
	switch panel {
	case tviewFocusTree:
		ui.tree.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.tree)
	case tviewFocusDetails:
		ui.detail.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.detail)
	case tviewFocusFiles:
		ui.files.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.files)
	case tviewFocusShortcuts:
		ui.app.SetFocus(ui.footer)
	default:
		ui.sidebar.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
		ui.app.SetFocus(ui.sidebar)
	}
	ui.renderFooter()
}

func (ui *tviewUI) moveFocused(delta int) {
	switch ui.focus {
	case tviewFocusTree, tviewFocusDetails:
		primitive := tview.Primitive(ui.tree)
		if ui.focus == tviewFocusDetails {
			primitive = ui.detail
		}
		key := tcell.KeyDown
		if delta < 0 {
			key = tcell.KeyUp
		}
		primitive.InputHandler()(tcell.NewEventKey(key, 0, tcell.ModNone), func(p tview.Primitive) { ui.app.SetFocus(p) })
	case tviewFocusFiles:
		row, _ := ui.files.GetSelection()
		maxRow := len(ui.plan.Files)
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
	case tviewFocusSidebar:
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
	if ui.screen != screen {
		ui.detail.ScrollToBeginning()
	}
	ui.screen = screen
	ui.refresh()
	if !ui.panelVisible(ui.focus) {
		if screen == tviewScreenPreview {
			ui.setFocusedPanel(tviewFocusFiles)
		} else {
			ui.setFocusedPanel(tviewFocusDetails)
		}
	}
}

func (ui *tviewUI) startEdit() {
	switch ui.screen {
	case tviewScreenOverview:
		if ui.selection.entity != "" {
			ui.editSelectedFields()
		} else if ui.selection.service != "" {
			ui.openEntitiesEdit()
		} else {
			ui.openServicesEdit()
		}
	case tviewScreenProject:
		ui.openProjectEdit()
	case tviewScreenServices:
		ui.openServicesEdit()
	case tviewScreenEntities:
		ui.openEntitiesEdit()
	case tviewScreenValueObjects:
		ui.openValueObjectsEdit()
	case tviewScreenRelationships:
		ui.openRelationshipsEdit()
	default:
		ui.message = "Open Project, Services, Entities, Value Objects, or Relationships before editing."
		ui.refresh()
	}
}

func tviewRelationshipsStateFromService(service application.ServiceSummary) *tviewRelationshipsEditState {
	state := &tviewRelationshipsEditState{serviceName: service.Name, rows: make([]tviewRelationshipRow, 0, len(service.Relationships))}
	for _, relationship := range service.Relationships {
		state.rows = append(state.rows, tviewRelationshipRow{originalName: relationship.Name, name: relationship.Name, multiplicity: relationship.Multiplicity, principalEntity: relationship.PrincipalEntity, dependentEntity: relationship.DependentEntity, foreignKeyName: relationship.ForeignKeyName, foreignKeyType: relationship.ForeignKeyType, required: relationship.Required, principalNavigation: relationship.PrincipalNavigation, dependentNavigation: relationship.DependentNavigation})
	}
	return state
}

func tviewRelationshipSettingsFromState(state *tviewRelationshipsEditState) application.RelationshipSettings {
	settings := application.RelationshipSettings{ServiceName: state.serviceName, Relationships: make([]application.RelationshipSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if row.deleted || strings.TrimSpace(row.name) == "" {
			continue
		}
		required := row.required
		settings.Relationships = append(settings.Relationships, application.RelationshipSetting{OriginalName: row.originalName, Name: strings.TrimSpace(row.name), Multiplicity: strings.TrimSpace(row.multiplicity), PrincipalEntity: strings.TrimSpace(row.principalEntity), DependentEntity: strings.TrimSpace(row.dependentEntity), ForeignKeyName: strings.TrimSpace(row.foreignKeyName), ForeignKeyType: strings.TrimSpace(row.foreignKeyType), Required: &required, PrincipalNavigation: strings.TrimSpace(row.principalNavigation), DependentNavigation: strings.TrimSpace(row.dependentNavigation)})
	}
	return settings
}

func (ui *tviewUI) openRelationshipsEdit() {
	ui.openRelationshipsEditForService(ui.selection.service)
}

func (ui *tviewUI) openRelationshipsEditForService(serviceName string) {
	service, ok := serviceSummaryByName(ui.plan.Config.Services, serviceName)
	if !ok {
		service, ok = firstService(ui.plan.Config.Services)
		if !ok {
			ui.closeEditWithMessage("No services are available to edit relationships.")
			return
		}
	}
	ui.showRelationshipsManager(tviewRelationshipsStateFromService(service))
}

func (ui *tviewUI) saveRelationshipsEdit(state *tviewRelationshipsEditState) {
	if ui.updateRelationships == nil {
		ui.message = "Relationship editing is not available."
		return
	}
	result, err := ui.updateRelationships(ui.request, tviewRelationshipSettingsFromState(state))
	ui.applyRelationshipsSaveResult(result, err)
	if err == nil {
		for index := range state.rows {
			state.rows[index].originalName = strings.TrimSpace(state.rows[index].name)
		}
	}
}

func (ui *tviewUI) applyRelationshipsSaveResult(result application.UpdateRelationshipSettingsResult, err error) {
	if err != nil {
		ui.message = "Relationships save failed: " + err.Error()
		ui.err = err
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.planStale = true
		ui.message = "Relationships saved, but plan refresh failed."
		ui.err = result.PlanError
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.refreshPlanKeepOpen("Relationships saved.")
}

func (ui *tviewUI) openEntitiesEdit() {
	ui.openEntitiesEditForService(ui.selection.service)
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
	state := tviewEntitiesStateFromService(service)
	if service.Name == ui.selection.service {
		state.selected = max(0, ui.selectedEntityIndex(service))
	}
	ui.showEntitiesManager(state)
}

func (ui *tviewUI) saveEntitiesEdit(state *tviewEntitiesEditState) {
	if ui.updateEntities == nil {
		ui.message = "Entity editing is not available."
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
	if err == nil {
		if state.serviceName == ui.selection.service {
			ui.selection.entity = savedWorkspaceName(state.rows, ui.selection.entity)
		}
		state.rows = updateRowsAfterSave(state.rows)
	}
}

func (ui *tviewUI) saveFieldsEdit(state *tviewFieldsEditState) {
	if ui.updateFields == nil {
		ui.message = "Field editing is not available."
		return
	}
	result, err := ui.saveFieldsState(state)
	ui.applyFieldsSaveResult(result, err)
	if err == nil {
		state.rows = updateRowsAfterSave(state.rows)
	}
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
		ui.message = "Entities save failed: " + err.Error()
		ui.err = err
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.planStale = true
		ui.message = "Entities saved, but plan refresh failed."
		ui.err = result.PlanError
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.refreshPlanKeepOpen("Entities saved.")
}

func (ui *tviewUI) applyFieldsSaveResult(result application.UpdateFieldSettingsResult, err error) {
	if err != nil {
		ui.planStale = true
		ui.message = "Fields save failed: " + err.Error()
		ui.err = err
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.planStale = true
		ui.message = "Fields saved, but plan refresh failed."
		ui.err = result.PlanError
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.refreshPlanKeepOpen("Fields saved.")
}

func (ui *tviewUI) openValueObjectsEdit() {
	ui.openValueObjectsEditForService(ui.selection.service)
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
		ui.message = "Value object editing is not available."
		return
	}
	result, err := ui.updateValueObjects(ui.request, tviewValueObjectSettingsFromState(state))
	ui.applyValueObjectsSaveResult(result, err)
	if err == nil {
		state.rows = updateRowsAfterSave(state.rows)
	}
}

func tviewValueObjectSettingsFromState(state *tviewValueObjectsEditState) application.ValueObjectSettings {
	settings := application.ValueObjectSettings{ServiceName: state.serviceName, ValueObjects: make([]application.ValueObjectNameSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		typeName := trimmedDefault(row.typeName, "string")
		validations := row.validations
		if row.original == "" && typeName == "string" && emptyValidationRuleSettings(validations) {
			validations = defaultStringValueObjectValidations()
		}
		settings.ValueObjects = append(settings.ValueObjects, application.ValueObjectNameSetting{OriginalName: row.original, Name: strings.TrimSpace(row.name), Type: typeName, Validations: validations})
	}
	return settings
}

func defaultStringValueObjectValidations() application.ValidationRuleSettings {
	required := true
	minLength := 1
	maxLength := 100
	validExample := "Sample"
	return application.ValidationRuleSettings{Required: &required, MinLength: &minLength, MaxLength: &maxLength, ValidExample: &validExample}
}

func emptyValidationRuleSettings(validations application.ValidationRuleSettings) bool {
	return validations.Required == nil &&
		validations.MinLength == nil &&
		validations.MaxLength == nil &&
		validations.Pattern == nil &&
		validations.ValidExample == nil &&
		validations.InvalidExample == nil &&
		validations.Minimum == nil &&
		validations.Maximum == nil &&
		validations.NotEmpty == nil &&
		validations.NotDefault == nil
}

func (ui *tviewUI) applyValueObjectsSaveResult(result application.UpdateValueObjectSettingsResult, err error) {
	if err != nil {
		ui.message = "Value objects save failed: " + err.Error()
		ui.err = err
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.planStale = true
		ui.message = "Value objects saved, but plan refresh failed."
		ui.err = result.PlanError
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.refreshPlanKeepOpen("Value objects saved.")
}

func (ui *tviewUI) openProjectEdit() {
	if ui.updateSettings == nil {
		ui.closeEditWithMessage("Project editing is not available.")
		return
	}
	name := ui.plan.Config.SolutionName
	description := ui.plan.Config.SolutionDescription
	targetFramework := ui.plan.Config.TargetFramework
	solutionFormat := ui.plan.Config.SolutionFormat
	generationMode := ui.plan.Config.GenerationMode
	if generationMode == "" {
		generationMode = spec.DefaultGenerationMode
	}
	modeUpdate := ""
	gatewayEnabled := fmt.Sprintf("%t", ui.plan.Config.GatewayEnabled)
	type projectField struct {
		label  string
		value  *string
		toggle bool
	}
	fields := []projectField{
		{"Solution name", &name, false},
		{"Description", &description, false},
		{"Target framework", &targetFramework, false},
		{"Solution format", &solutionFormat, false},
		{"Generation mode", &generationMode, false},
		{"Gateway enabled", &gatewayEnabled, true},
	}
	saveSettings := func() {
		gateway := gatewayEnabled == "true"
		result, err := ui.updateSettings(ui.request, application.SolutionSettings{
			SolutionName:        strings.TrimSpace(name),
			SolutionDescription: strings.TrimSpace(description),
			TargetFramework:     strings.TrimSpace(targetFramework),
			SolutionFormat:      strings.TrimSpace(solutionFormat),
			GenerationMode:      modeUpdate,
			GatewayEnabled:      &gateway,
		})
		ui.applyProjectSaveResult(result, err)
	}
	render := func(manager *tview.Table) {
		for manager.GetRowCount() > 1 {
			manager.RemoveRow(1)
		}
		for index, field := range fields {
			manager.SetCell(index+1, 0, managerCell(field.label, tcell.StyleDefault.Foreground(tcell.ColorWhite)).SetExpansion(2))
			manager.SetCell(index+1, 1, managerCell(*field.value, tcell.StyleDefault.Foreground(tcell.ColorWhite)).SetExpansion(3))
		}
	}
	var showManager func()
	showManager = func() {
		manager := ui.newManagerTable([]string{"Property", "Value"})
		render(manager)
		manager.SetSelectedFunc(func(row, column int) {
			if row <= 0 || row > len(fields) {
				return
			}
			field := &fields[row-1]
			if field.value == &generationMode {
				modes := []string{spec.GenerationModeSimple, spec.GenerationModeEnterprise}
				choice := tview.NewDropDown().SetLabel("Generation mode").SetOptions(modes, nil).
					SetCurrentOption(selectedStringIndex(modes, generationMode))
				form := tview.NewForm().AddFormItem(choice)
				help := tview.NewTextView().SetTextColor(tcell.ColorGray).
					SetText("simple: compact CRUD; enterprise: layered scaffold")
				ui.showChildForm(form, help, " Generation mode ", 82, 12, func() {
					_, generationMode = choice.GetCurrentOption()
					modeUpdate = generationMode
					render(manager)
				})
				return
			}
			if field.toggle {
				*field.value = fmt.Sprintf("%t", *field.value != "true")
				render(manager)
				return
			}
			ui.openNameEdit(" Edit "+field.label+" ", *field.value, func(value string) {
				*field.value = value
				render(manager)
			})
		})
		manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCtrlS:
				saveSettings()
				render(manager)
				return nil
			}
			if moveManagerSelection(manager, len(fields), event) {
				return nil
			}
			return event
		})
		ui.showManagerModal(" Project ", nil, nil, manager, managerFooter("Tab focus  Up/Down/j/k move  Enter edit/toggle  ctrl+s save  esc cancel", []managerButton{{"Save", func() {
			saveSettings()
			render(manager)
		}}, {"Cancel", ui.closeEdit}}))
	}
	showManager()
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
		ui.message = "Service editing is not available."
		return
	}
	settings := application.ServiceSettings{Services: make([]application.ServiceNameSetting, 0, len(state.rows))}
	for _, row := range state.rows {
		if strings.TrimSpace(row.name) == "" || row.deleted {
			continue
		}
		settings.Services = append(settings.Services, application.ServiceNameSetting{
			OriginalName:     row.original,
			Name:             strings.TrimSpace(row.name),
			ConnectionString: row.connectionString,
		})
	}
	result, err := ui.updateServices(ui.request, settings)
	ui.applyServicesSaveResult(result, err)
	if err == nil {
		ui.selection.service = savedWorkspaceName(state.rows, ui.selection.service)
		state.rows = updateRowsAfterSave(state.rows)
		for i := range state.rows {
			row := &state.rows[i]
			if row.connectionString != nil {
				row.hasConnection = *row.connectionString != ""
				row.connectionString = nil
			}
		}
	}
}

func (ui *tviewUI) applyServicesSaveResult(result application.UpdateServiceSettingsResult, err error) {
	if err != nil {
		ui.message = "Services save failed: " + err.Error()
		ui.err = err
		return
	}
	if result.PlanError != nil {
		ui.plan.Config = result.Config
		ui.planStale = true
		ui.message = "Services saved, but plan refresh failed."
		ui.err = result.PlanError
		return
	}
	if result.Plan.Config.SolutionName != "" || result.Plan.FileCount > 0 || len(result.Plan.Files) > 0 {
		ui.plan = result.Plan
	}
	ui.refreshPlanKeepOpen("Services saved.")
}

func (ui *tviewUI) showServicesManager(state *tviewServicesEditState) {
	manager := ui.newManagerTable([]string{"Name", "Connection", "Actions"})
	render := func() {
		for manager.GetRowCount() > 1 {
			manager.RemoveRow(1)
		}
		if len(state.rows) == 0 {
			manager.SetCell(1, 0, tview.NewTableCell("No services yet. Press a to add one.").SetTextColor(tcell.ColorGray).SetExpansion(3))
			return
		}
		for index, row := range state.rows {
			textStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			action := "Delete"
			if row.deleted {
				textStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
				action = "Restore"
			}
			manager.SetCell(index+1, 0, managerCell(emptyDash(strings.TrimSpace(row.name)), textStyle).SetExpansion(2))
			configured := row.hasConnection
			if row.connectionString != nil {
				configured = *row.connectionString != ""
			}
			connDisplay := "-"
			if configured {
				connDisplay = "Set"
			}
			manager.SetCell(index+1, 1, managerCell(connDisplay, textStyle).SetExpansion(2))
			manager.SetCell(index+1, 2, managerCell(action, tcell.StyleDefault.Foreground(tcell.ColorYellow)).SetAlign(tview.AlignRight).SetExpansion(1))
		}
	}
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
		if column == 0 {
			ui.openNameEdit(" Edit Service ", state.rows[row-1].name, func(name string) {
				state.rows[row-1].name = name
				render()
			})
			return
		}
		if column == 1 {
			var draft string
			if pending := state.rows[row-1].connectionString; pending != nil {
				draft = *pending
			}
			form := tview.NewForm().AddPasswordField("Connection", draft, tviewEditModalInputWidth, '*', nil)
			help := tview.NewTextView().SetTextColor(tcell.ColorGray).SetText("Apply empty to clear; Esc to keep unchanged.")
			ui.showChildForm(form, help, " Edit Connection String ", 82, 18, func() {
				value := form.GetFormItem(0).(*tview.InputField).GetText()
				state.rows[row-1].connectionString = &value
				render()
			})
		}
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveServicesEdit(state)
			render()
			return nil
		}
		if moveManagerSelection(manager, len(state.rows), event) {
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
	ui.showManagerModal(" Services ", nil, nil, manager, managerFooter("Tab focus  Up/Down/j/k move rows  a add  d delete  enter edit  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewService", len(state.rows)+1)})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Save", func() {
		ui.saveServicesEdit(state)
		render()
	}}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) showEntitiesManager(state *tviewEntitiesEditState) {
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	header := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(serviceManagerHeader(state.serviceName, "s service"), 1, 0, false).
		AddItem(tview.NewTextView().SetTextColor(tcell.ColorGray).
			SetText("Fields: primitives. Value Objects: reusable types."), 1, 0, false)
	manager := ui.newManagerTable([]string{"Name", "Fields", "Delete"})
	render := func() {
		for manager.GetRowCount() > 1 {
			manager.RemoveRow(1)
		}
		if len(state.rows) == 0 {
			manager.SetCell(1, 0, tview.NewTableCell("No entities yet. Press a to add one.").SetTextColor(tcell.ColorGray).SetExpansion(2))
			return
		}
		for index, row := range state.rows {
			textStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			action := "Delete"
			fieldsStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			if row.deleted {
				textStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
				action = "Restore"
			}
			if row.deleted || row.original == "" {
				fieldsStyle = textStyle.Foreground(tcell.ColorGray)
			}
			manager.SetCell(index+1, 0, managerCell(emptyDash(strings.TrimSpace(row.name)), textStyle).SetExpansion(2))
			manager.SetCell(index+1, 1, managerCell("Fields", fieldsStyle))
			manager.SetCell(index+1, 2, managerCell(action, tcell.StyleDefault.Foreground(tcell.ColorYellow)))
		}
	}
	render()
	manager.Select(state.selected+1, 0)
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 1 {
			ui.openFieldsFromEntityState(state, row-1)
			return
		}
		if column == 2 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		if column == 0 {
			ui.openNameEdit(" Edit Entity ", state.rows[row-1].name, func(name string) {
				state.rows[row-1].name = name
				render()
			})
		}
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveEntitiesEdit(state)
			render()
			return nil
		}
		if moveManagerSelection(manager, len(state.rows), event) {
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewEntity", len(state.rows)+1)})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "s":
			ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openEntitiesEditForService)
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
	ui.showManagerModal(" Entities ", header, nil, manager, managerFooter("Tab focus  Up/Down/j/k move rows  s service  a add  d delete  enter edit/select  f fields  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewEntity", len(state.rows)+1)})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Service", func() {
		ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openEntitiesEditForService)
	}}, {"Fields", func() {
		row, _ := manager.GetSelection()
		ui.openFieldsFromEntityState(state, row-1)
	}}, {"Save", func() {
		ui.saveEntitiesEdit(state)
		render()
	}}, {"Cancel", ui.closeEdit}}))
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
			back := ui.parentEditor()
			ui.showFieldsManager(tviewFieldsStateFromEntity(state.serviceName, entity))
			ui.editBack = back
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
			render()
		})
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveFieldsEdit(state)
			render()
			return nil
		}
		if moveManagerSelection(manager, len(state.rows), event) {
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
	ui.showManagerModal(" Fields ", context, nil, manager, managerFooter("Tab focus  Up/Down/j/k move rows  a add  d delete  enter edit  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewField", len(state.rows)+1), typeName: "string"})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Save", func() {
		ui.saveFieldsEdit(state)
		render()
	}}, {"Cancel", ui.backFromEdit}}))
}

func (ui *tviewUI) showValueObjectsManager(state *tviewValueObjectsEditState) {
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	header := serviceManagerHeader(state.serviceName, "s service")
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
			render()
		})
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveValueObjectsEdit(state)
			render()
			return nil
		}
		if moveManagerSelection(manager, len(state.rows), event) {
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewValueObject", len(state.rows)+1), typeName: "string"})
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "s":
			ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openValueObjectsEditForService)
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
	ui.showManagerModal(" Value Objects ", header, nil, manager, managerFooter("Tab focus  Up/Down/j/k move rows  s service  a add  d delete  enter edit/select  r rules  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, tviewManagerRow{name: nextNamedRow("NewValueObject", len(state.rows)+1), typeName: "string"})
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Service", func() {
		ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openValueObjectsEditForService)
	}}, {"Rules", func() {
		row, _ := manager.GetSelection()
		ui.openRulesFromValueObjectState(state, row-1)
	}}, {"Save", func() {
		ui.saveValueObjectsEdit(state)
		render()
	}}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) showRelationshipsManager(state *tviewRelationshipsEditState) {
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	header := serviceManagerHeader(state.serviceName, "s service")
	manager := ui.newManagerTable([]string{"Name", "Principal", "Dependent", "FK", "Required", "Actions"})
	render := func() {
		for manager.GetRowCount() > 1 {
			manager.RemoveRow(1)
		}
		for index, row := range state.rows {
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			if row.deleted {
				style = tcell.StyleDefault.Foreground(tcell.ColorGray).StrikeThrough(true)
			}
			required := "optional"
			if row.required {
				required = "required"
			}
			values := []string{row.name, row.principalEntity, row.dependentEntity, trimmedDefault(row.foreignKeyName, "Guid"), required, "delete"}
			for column, value := range values {
				manager.SetCell(index+1, column, managerCell(value, style).SetExpansion(1))
			}
		}
	}
	render()
	manager.SetSelectedFunc(func(row, column int) {
		if row <= 0 || row > len(state.rows) {
			return
		}
		if column == 5 {
			state.rows[row-1].deleted = !state.rows[row-1].deleted
			render()
			return
		}
		ui.openRelationshipEdit(" Edit Relationship ", state, row-1, render)
	})
	manager.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			ui.saveRelationshipsEdit(state)
			render()
			return nil
		}
		if moveManagerSelection(manager, len(state.rows), event) {
			return nil
		}
		switch strings.ToLower(string(event.Rune())) {
		case "a":
			state.rows = append(state.rows, ui.newTViewRelationshipRow(state.serviceName, len(state.rows)+1))
			render()
			manager.Select(len(state.rows), 0)
			return nil
		case "s":
			ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openRelationshipsEditForService)
			return nil
		case "d":
			tviewToggleSelectedRelationshipRow(manager, state.rows)
			render()
			return nil
		}
		return event
	})
	ui.showManagerModal(" Relationships ", header, nil, manager, managerFooter("Tab focus  Up/Down/j/k move rows  s service  a add  d delete  enter edit/select  ctrl+s save  esc cancel", []managerButton{{"Add", func() {
		state.rows = append(state.rows, ui.newTViewRelationshipRow(state.serviceName, len(state.rows)+1))
		render()
		manager.Select(len(state.rows), 0)
	}}, {"Service", func() {
		ui.openServicePicker(serviceNames, state.serviceName, manager, ui.openRelationshipsEditForService)
	}}, {"Save", func() {
		ui.saveRelationshipsEdit(state)
		render()
	}}, {"Cancel", ui.closeEdit}}))
}

func (ui *tviewUI) openRelationshipEdit(title string, state *tviewRelationshipsEditState, index int, render func()) {
	if index < 0 || index >= len(state.rows) || state.rows[index].deleted {
		return
	}
	row := state.rows[index]
	entityNames := tviewRelationshipEntityNames(ui.plan.Config.Services, state.serviceName)
	form := tview.NewForm()
	addEditInputField(form, "Name", row.name)
	form.AddDropDown("Multiplicity", []string{"one-to-many", "many-to-one"}, selectedStringIndex([]string{"one-to-many", "many-to-one"}, row.multiplicity), func(_ string, optionIndex int) {
		row.multiplicity = []string{"one-to-many", "many-to-one"}[optionIndex]
	})
	form.AddDropDown("Principal entity", entityNames, selectedStringIndex(entityNames, row.principalEntity), func(value string, _ int) { row.principalEntity = value })
	form.AddDropDown("Dependent entity", entityNames, selectedStringIndex(entityNames, row.dependentEntity), func(value string, _ int) { row.dependentEntity = value })
	addEditInputField(form, "Foreign key name", row.foreignKeyName)
	addEditInputField(form, "Foreign key type", trimmedDefault(row.foreignKeyType, "Guid"))
	form.AddCheckbox("Required", row.required, func(checked bool) { row.required = checked })
	addEditInputField(form, "Principal navigation", row.principalNavigation)
	addEditInputField(form, "Dependent navigation", row.dependentNavigation)
	ui.showEditForm(form, title, func() {
		row.name = formInputTextByLabel(form, "Name")
		row.foreignKeyName = formInputTextByLabel(form, "Foreign key name")
		row.foreignKeyType = trimmedDefault(formInputTextByLabel(form, "Foreign key type"), "Guid")
		row.required = formCheckboxCheckedByLabel(form, "Required")
		row.principalNavigation = formInputTextByLabel(form, "Principal navigation")
		row.dependentNavigation = formInputTextByLabel(form, "Dependent navigation")
		state.rows[index] = row
		render()
	})
}

func (ui *tviewUI) newTViewRelationshipRow(serviceName string, count int) tviewRelationshipRow {
	entities := tviewRelationshipEntityNames(ui.plan.Config.Services, serviceName)
	principal, dependent := "Principal", "Dependent"
	if len(entities) > 0 {
		principal = entities[0]
		dependent = entities[0]
	}
	if len(entities) > 1 {
		dependent = entities[1]
	}
	return tviewRelationshipRow{name: nextNamedRow("Relationship", count), multiplicity: "one-to-many", principalEntity: principal, dependentEntity: dependent, foreignKeyName: principal + "Id", foreignKeyType: "Guid", required: true, principalNavigation: dependent + "s", dependentNavigation: principal}
}

func tviewRelationshipEntityNames(services []application.ServiceSummary, serviceName string) []string {
	service, ok := serviceSummaryByName(services, serviceName)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(service.Entities))
	for _, entity := range service.Entities {
		names = append(names, entity.Name)
	}
	return names
}

func selectedStringIndex(values []string, selected string) int {
	for index, value := range values {
		if value == selected {
			return index
		}
	}
	return 0
}

func tviewToggleSelectedRelationshipRow(table *tview.Table, rows []tviewRelationshipRow) {
	row, _ := table.GetSelection()
	if row <= 0 || row > len(rows) {
		return
	}
	rows[row-1].deleted = !rows[row-1].deleted
}

func (ui *tviewUI) openRulesFromValueObjectState(state *tviewValueObjectsEditState, index int) {
	if index < 0 || index >= len(state.rows) || state.rows[index].deleted {
		return
	}
	row := state.rows[index]
	ui.showValueObjectRulesManager(&tviewValueObjectRulesEditState{
		serviceName: state.serviceName,
		valueObject: application.ValueObjectSummary{Name: strings.TrimSpace(row.name), Type: trimmedDefault(row.typeName, "string")},
		rows:        state.rows,
		index:       index,
		validations: row.validations,
	})
}

func (ui *tviewUI) showValueObjectRulesManager(state *tviewValueObjectRulesEditState) {
	context := tview.NewTextView().SetDynamicColors(true)
	context.SetText(fmt.Sprintf("[aqua]%s[white] / [yellow]%s[white] ([gray]%s[white])", state.serviceName, state.valueObject.Name, state.valueObject.Type))
	form := tview.NewForm()
	form.SetBorder(false)
	form.SetFieldTextColor(tcell.ColorWhite).SetLabelColor(tcell.ColorLightCyan).SetButtonTextColor(tcell.ColorBlack).SetButtonBackgroundColor(tcell.ColorTeal)
	required := boolValue(state.validations.Required)
	notEmpty := boolValue(state.validations.NotEmpty)
	notDefault := boolValue(state.validations.NotDefault)
	form.AddCheckbox("Required", required, func(checked bool) { state.validations.Required = &checked })
	form.AddTextView("", "[darkcyan]── Length Constraints ──", 0, 1, true, false)
	form.AddInputField("Min length", intString(state.validations.MinLength), tviewEditModalInputWidth, nil, func(value string) { state.validations.MinLength = intPointerFromText(value) })
	form.AddInputField("Max length", intString(state.validations.MaxLength), tviewEditModalInputWidth, nil, func(value string) { state.validations.MaxLength = intPointerFromText(value) })
	form.AddTextView("", "[darkcyan]── Pattern Matching ──", 0, 1, true, false)
	form.AddInputField("Pattern", stringValue(state.validations.Pattern), tviewEditModalInputWidth, nil, func(value string) { state.validations.Pattern = stringPointerFromText(value) })
	form.AddInputField("Valid example", stringValue(state.validations.ValidExample), tviewEditModalInputWidth, nil, func(value string) { state.validations.ValidExample = stringPointerFromText(value) })
	form.AddInputField("Invalid example", stringValue(state.validations.InvalidExample), tviewEditModalInputWidth, nil, func(value string) { state.validations.InvalidExample = stringPointerFromText(value) })
	form.AddTextView("", "[darkcyan]── Value Range ──", 0, 1, true, false)
	form.AddInputField("Minimum", stringValue(state.validations.Minimum), tviewEditModalInputWidth, nil, func(value string) { state.validations.Minimum = stringPointerFromText(value) })
	form.AddInputField("Maximum", stringValue(state.validations.Maximum), tviewEditModalInputWidth, nil, func(value string) { state.validations.Maximum = stringPointerFromText(value) })
	form.AddTextView("", "[darkcyan]── State Checks ──", 0, 1, true, false)
	form.AddCheckbox("Not empty", notEmpty, func(checked bool) { state.validations.NotEmpty = &checked })
	form.AddCheckbox("Not default", notDefault, func(checked bool) { state.validations.NotDefault = &checked })
	ui.showChildForm(form, context, fmt.Sprintf(" Rules: %s/%s ", state.serviceName, state.valueObject.Name), 76, 28, func() {
		applyValueObjectRulesEdit(state, form)
	})
}

func applyValueObjectRulesEdit(state *tviewValueObjectRulesEditState, form *tview.Form) {
	state.rows[state.index].validations = validationRuleSettingsFromRulesForm(state.valueObject.Type, form)
}

func validationRuleSettingsFromRulesForm(valueObjectType string, form *tview.Form) application.ValidationRuleSettings {
	switch valueObjectType {
	case "string":
		return application.ValidationRuleSettings{
			Required:       boolRulePointerFromFormLabel(form, "Required"),
			MinLength:      intPointerFromFormLabel(form, "Min length"),
			MaxLength:      intPointerFromFormLabel(form, "Max length"),
			Pattern:        stringPointerFromFormLabel(form, "Pattern"),
			ValidExample:   stringPointerFromFormLabel(form, "Valid example"),
			InvalidExample: stringPointerFromFormLabel(form, "Invalid example"),
		}
	case "int", "long", "double", "decimal":
		return application.ValidationRuleSettings{
			Minimum: stringPointerFromFormLabel(form, "Minimum"),
			Maximum: stringPointerFromFormLabel(form, "Maximum"),
		}
	case "Guid":
		return application.ValidationRuleSettings{NotEmpty: boolRulePointerFromFormLabel(form, "Not empty")}
	case "DateTime":
		return application.ValidationRuleSettings{NotDefault: boolRulePointerFromFormLabel(form, "Not default")}
	default:
		return application.ValidationRuleSettings{}
	}
}

func (ui *tviewUI) showEditForm(form *tview.Form, title string, apply func()) {
	ui.showChildForm(form, nil, title, 82, 18, apply)
}

func (ui *tviewUI) showChildForm(form *tview.Form, header tview.Primitive, title string, width, height int, apply func()) {
	ui.editBack = ui.parentEditor()
	ui.childApply = func() {
		apply()
		ui.backFromEdit()
	}
	form.AddButton("Apply", ui.childApply)
	form.AddButton("Cancel", ui.backFromEdit)
	form.SetCancelFunc(ui.backFromEdit)
	form.SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorLightCyan).
		SetButtonTextColor(tcell.ColorBlack).
		SetButtonBackgroundColor(tcell.ColorTeal)
	form.SetBorderPadding(0, 0, 1, 1)
	configureEditButtons(form)
	// Native form drawing scrolls to keep the focused control visible.
	form.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		x, y := event.Position()
		if (action != tview.MouseScrollUp && action != tview.MouseScrollDown) || !form.InRect(x, y) {
			return action, event
		}
		if _, listOpen := ui.app.GetFocus().(*tview.List); listOpen {
			return action, nil // Do not move form focus away from an open dropdown.
		}
		delta := 1
		if action == tview.MouseScrollUp {
			delta = -1
		}
		item, button := form.GetFocusedItemIndex()
		if button >= 0 {
			item = form.GetFormItemCount() + button
		}
		for next := item + delta; next >= 0 && next < form.GetFormItemCount()+form.GetButtonCount(); next += delta {
			if next < form.GetFormItemCount() {
				if _, readOnly := form.GetFormItem(next).(*tview.TextView); readOnly {
					continue
				}
			}
			ui.app.SetFocus(form.SetFocus(next))
			break
		}
		return action, nil
	})
	body := tview.NewFlex().SetDirection(tview.FlexRow)
	if header != nil {
		body.AddItem(header, 1, 0, false)
	}
	body.AddItem(form, 0, 1, true)
	panel := tview.NewFrame(body).
		SetBorders(0, 0, 0, 0, 1, 1).
		AddText("Tab next  Ctrl+S Apply  Esc cancel", false, tview.AlignCenter, tcell.ColorGray)
	panel.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	panel.SetBorderColor(tcell.ColorTeal).SetTitleColor(tcell.ColorLightCyan)
	ui.editOpen = true
	ui.modalFocus = nil
	ui.modalFocusIndex = 0
	ui.root.RemovePage(tviewEditModalPage)
	ui.root.AddPage(tviewEditModalPage, centerPrimitive(panel, width, height), true, true)
	ui.app.SetFocus(form)
}

func configureEditButtons(form *tview.Form) {
	for i := 0; i < form.GetButtonCount(); i++ {
		button := form.GetButton(i)
		button.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			x, y := event.Position()
			if button.InRect(x, y) {
				switch action {
				case tview.MouseLeftClick:
					return action, nil // Native mouse-down already focused the button.
				case tview.MouseLeftDoubleClick:
					return tview.MouseLeftClick, event
				}
			}
			return action, event
		})
	}
}

type managerButton struct {
	label string
	run   func()
}

func (ui *tviewUI) newManagerTable(headers []string) *tview.Table {
	table := tview.NewTable().SetSelectable(true, true).SetFixed(1, 0)
	table.SetBorder(false)
	table.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorTeal).Bold(true))
	for column, header := range headers {
		table.SetCell(0, column, tview.NewTableCell(header).SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(1))
	}
	table.Select(1, 0)
	table.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		x, y := event.Position()
		if action == tview.MouseLeftDoubleClick && table.InRect(x, y) {
			row, column := table.CellAt(x, y)
			if selectedRow, selectedColumn := table.GetSelection(); row > 0 && row == selectedRow && column == selectedColumn {
				table.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) { ui.app.SetFocus(p) })
			}
		}
		return action, event
	})
	return table
}

func serviceManagerHeader(serviceName, shortcut string) *tview.TextView {
	header := tview.NewTextView().SetDynamicColors(true).SetTextColor(tcell.ColorLightCyan)
	header.SetText(fmt.Sprintf("Service: [white]%s[-]  [gray]%s[-]", emptyDash(serviceName), shortcut))
	return header
}

func (ui *tviewUI) openServicePicker(serviceNames []string, selected string, returnFocus tview.Primitive, choose func(string)) {
	if len(serviceNames) == 0 {
		return
	}
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle(" Select Service ").SetTitleAlign(tview.AlignLeft)
	list.SetBorderColor(tcell.ColorTeal).SetTitleColor(tcell.ColorLightCyan)
	list.SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorTeal)
	for _, serviceName := range serviceNames {
		name := serviceName
		list.AddItem(name, "", 0, func() {
			ui.closeServicePicker()
			choose(name)
		})
	}
	index := selectedIndex(serviceNames, selected)
	if index < 0 {
		index = 0
	}
	list.SetCurrentItem(index)
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch strings.ToLower(string(event.Rune())) {
		case "j":
			list.SetCurrentItem((list.GetCurrentItem() + 1) % len(serviceNames))
			return nil
		case "k":
			next := list.GetCurrentItem() - 1
			if next < 0 {
				next = len(serviceNames) - 1
			}
			list.SetCurrentItem(next)
			return nil
		}
		return event
	})
	ui.servicePickerReturnFocus = returnFocus
	ui.root.RemovePage(tviewServicePickerPage)
	ui.root.AddPage(tviewServicePickerPage, centerPrimitive(list, 44, 12), true, true)
	ui.app.SetFocus(list)
}

func (ui *tviewUI) closeServicePicker() {
	returnFocus := ui.servicePickerReturnFocus
	ui.servicePickerReturnFocus = nil
	ui.root.RemovePage(tviewServicePickerPage)
	if returnFocus != nil {
		ui.app.SetFocus(returnFocus)
	}
}

func (ui *tviewUI) newServiceSelector(serviceNames []string, selected string, choose func(string)) *tview.Table {
	selector := tview.NewTable().SetSelectable(true, false)
	selector.SetBorder(false)
	selector.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorTeal).Bold(true))
	render := func() {
		selector.Clear()
		selector.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Service: %s", emptyDash(selected))).SetTextColor(tcell.ColorLightCyan).SetExpansion(1))
		selector.SetCell(0, 1, tview.NewTableCell("n/p or [/] switch").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight).SetExpansion(1))
		selector.Select(0, 0)
	}
	cycle := func(delta int) {
		if len(serviceNames) == 0 {
			return
		}
		index := selectedIndex(serviceNames, selected)
		if index < 0 {
			index = 0
		}
		index += delta
		if index < 0 {
			index = len(serviceNames) - 1
		}
		if index >= len(serviceNames) {
			index = 0
		}
		if serviceNames[index] != selected {
			choose(serviceNames[index])
		}
	}
	render()
	selector.SetSelectedFunc(func(_, _ int) { ui.cycleModalFocus(1) })
	selector.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch strings.ToLower(string(event.Rune())) {
		case "n", "]":
			cycle(1)
			return nil
		case "p", "[":
			cycle(-1)
			return nil
		}
		return event
	})
	return selector
}

func moveManagerSelection(table *tview.Table, rowCount int, event *tcell.EventKey) bool {
	delta := 0
	switch event.Key() {
	case tcell.KeyUp:
		delta = -1
	case tcell.KeyDown:
		delta = 1
	}
	switch strings.ToLower(string(event.Rune())) {
	case "k":
		delta = -1
	case "j":
		delta = 1
	}
	if delta == 0 || rowCount == 0 {
		return false
	}
	row, column := table.GetSelection()
	row += delta
	if row < 1 {
		row = rowCount
	}
	if row > rowCount {
		row = 1
	}
	table.Select(row, column)
	return true
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
		textStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		action := "Delete"
		if row.deleted {
			textStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
			action = "Restore"
		}
		table.SetCell(index+1, 0, managerCell(emptyDash(strings.TrimSpace(row.name)), textStyle).SetExpansion(2))
		if withType {
			table.SetCell(index+1, 1, managerCell(trimmedDefault(row.typeName, "string"), textStyle).SetExpansion(1))
			if actionLabel != "" && !row.deleted {
				action += " | " + actionLabel
			}
			table.SetCell(index+1, 2, managerCell(action, tcell.StyleDefault.Foreground(tcell.ColorYellow)).SetAlign(tview.AlignRight).SetExpansion(1))
		} else {
			table.SetCell(index+1, 1, managerCell(action, tcell.StyleDefault.Foreground(tcell.ColorYellow)).SetAlign(tview.AlignRight).SetExpansion(1))
		}
	}
}

func managerCell(text string, style tcell.Style) *tview.TableCell {
	return tview.NewTableCell(text).
		SetStyle(style).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorTeal).Bold(true))
}

type tviewManagerFooter struct {
	view *tview.Flex
	form *tview.Form
}

func managerFooter(shortcuts string, buttons []managerButton) *tviewManagerFooter {
	form := tview.NewForm().SetHorizontal(true)
	form.SetBorderPadding(0, 0, 0, 0)
	form.SetButtonTextColor(tcell.ColorBlack).SetButtonBackgroundColor(tcell.ColorTeal)
	for _, button := range buttons {
		form.AddButton(button.label, button.run)
	}
	configureEditButtons(form)
	view := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetDynamicColors(true).SetText("[gray]"+shortcuts), 1, 0, false).
		AddItem(form, 0, 1, false)
	return &tviewManagerFooter{view: view, form: form}
}

func (ui *tviewUI) showManagerModal(title string, header tview.Primitive, headerFocus tview.Primitive, manager *tview.Table, footer *tviewManagerFooter) {
	body := tview.NewFlex().SetDirection(tview.FlexRow)
	if header != nil {
		body.AddItem(header, 2, 0, false)
	}
	body.AddItem(manager, 0, 1, true).AddItem(footer.view, 3, 0, false)
	panel := tview.NewFrame(body).SetBorders(1, 1, 0, 1, 1, 1)
	panel.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if ui.servicePickerReturnFocus != nil {
			return action, nil
		}
		return action, event
	})
	panel.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	panel.SetBorderColor(tcell.ColorTeal).SetTitleColor(tcell.ColorLightCyan)
	modal := centerPrimitive(panel, 86, 22)
	ui.editOpen = true
	ui.modalFocus = nil
	if headerFocus != nil {
		ui.modalFocus = append(ui.modalFocus, headerFocus)
	}
	ui.modalFocus = append(ui.modalFocus, manager, footer.form)
	ui.modalFocusIndex = 0
	ui.root.RemovePage(tviewEditModalPage)
	ui.root.AddPage(tviewEditModalPage, modal, true, true)
	ui.app.SetFocus(ui.modalFocus[ui.modalFocusIndex])
}

func (ui *tviewUI) openNameEdit(title, value string, save func(string)) {
	form := tview.NewForm()
	addEditInputField(form, "Name", value)
	ui.showEditForm(form, title, func() { save(formInputText(form, 0)) })
}

func (ui *tviewUI) openFieldEdit(title string, row tviewManagerRow, save func(tviewManagerRow)) {
	form := tview.NewForm()
	addEditInputField(form, "Name", row.name)
	addEditInputField(form, "Type", trimmedDefault(row.typeName, "string"))
	ui.showEditForm(form, title, func() {
		row.name = formInputText(form, 0)
		row.typeName = formInputText(form, 1)
		save(row)
	})
}

func tviewServicesStateFromPlan(plan application.GenerationPlan) *tviewServicesEditState {
	services := plan.Config.ServiceNames
	if len(services) == 0 && len(plan.Config.Services) > 0 {
		services = serviceSummaryNames(plan.Config.Services)
	}
	state := &tviewServicesEditState{rows: make([]tviewManagerRow, 0, len(services))}
	for _, name := range services {
		summary, _ := serviceSummaryByName(plan.Config.Services, name)
		state.rows = append(state.rows, tviewManagerRow{original: name, name: name, hasConnection: summary.HasConnectionString})
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

func formInputTextByLabel(form *tview.Form, label string) string {
	item := form.GetFormItemByLabel(label)
	if item == nil {
		return ""
	}
	if input, ok := item.(*tview.InputField); ok {
		return strings.TrimSpace(input.GetText())
	}
	return ""
}

func formCheckboxCheckedByLabel(form *tview.Form, label string) bool {
	item := form.GetFormItemByLabel(label)
	if item == nil {
		return false
	}
	if checkbox, ok := item.(*tview.Checkbox); ok {
		return checkbox.IsChecked()
	}
	return false
}

func stringPointerFromFormLabel(form *tview.Form, label string) *string {
	return stringPointerFromText(formInputTextByLabel(form, label))
}

func intPointerFromFormLabel(form *tview.Form, label string) *int {
	return intPointerFromText(formInputTextByLabel(form, label))
}

func boolPointerFromFormLabel(form *tview.Form, label string) *bool {
	checked := formCheckboxCheckedByLabel(form, label)
	return &checked
}

func boolRulePointerFromFormLabel(form *tview.Form, label string) *bool {
	checked := formCheckboxCheckedByLabel(form, label)
	if !checked {
		return nil
	}
	return &checked
}

// Retain the actual parent primitive so its viewport and selection survive.
func (ui *tviewUI) parentEditor() func() {
	if !ui.editOpen {
		return nil
	}
	_, page := ui.root.GetFrontPage()
	focus, modalFocus, index := ui.app.GetFocus(), ui.modalFocus, ui.modalFocusIndex
	back, apply := ui.editBack, ui.childApply
	return func() {
		ui.editBack, ui.childApply = back, apply
		ui.modalFocus, ui.modalFocusIndex = modalFocus, index
		ui.root.RemovePage(tviewEditModalPage)
		ui.root.AddPage(tviewEditModalPage, page, true, true)
		ui.app.SetFocus(focus)
	}
}

func (ui *tviewUI) backFromEdit() {
	if ui.editBack != nil {
		ui.editBack()
		return
	}
	ui.closeEdit()
}

func (ui *tviewUI) closeEdit() {
	ui.editBack, ui.childApply = nil, nil
	ui.editOpen = false
	ui.modalFocus = nil
	ui.modalFocusIndex = 0
	ui.servicePickerReturnFocus = nil
	ui.root.RemovePage(tviewServicePickerPage)
	ui.root.RemovePage(tviewEditModalPage)
	ui.setFocusedPanel(ui.focus)
	ui.refresh()
}

type centeredModal struct {
	*tview.Flex
	row           *tview.Flex
	content       tview.Primitive
	width, height int
}

func (modal *centeredModal) Draw(screen tcell.Screen) {
	_, _, width, height := modal.GetRect()
	modal.ResizeItem(modal.row, min(modal.height, height), 0)
	modal.row.ResizeItem(modal.content, min(modal.width, width), 0)
	modal.Flex.Draw(screen)
}

func centerPrimitive(primitive tview.Primitive, width, height int) tview.Primitive {
	row := tview.NewFlex().AddItem(nil, 0, 1, false).AddItem(primitive, width, 0, true).AddItem(nil, 0, 1, false)
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).AddItem(row, height, 0, true).AddItem(nil, 0, 1, false)
	return &centeredModal{Flex: column, row: row, content: primitive, width: width, height: height}
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

func (ui *tviewUI) refreshPlanKeepOpen(message string) {
	if ui.planFunc == nil {
		ui.message = message
		ui.err = nil
		ui.planStale = false
		return
	}
	plan, err := ui.planFunc(ui.request)
	if err != nil {
		ui.message = message + " Plan refresh failed."
		ui.err = err
		ui.planStale = true
		return
	}
	ui.plan = plan
	ui.err = nil
	ui.planStale = false
	ui.message = message
}

func updateRowsAfterSave(rows []tviewManagerRow) []tviewManagerRow {
	remaining := make([]tviewManagerRow, 0, len(rows))
	for _, row := range rows {
		if row.deleted {
			continue
		}
		row.original = strings.TrimSpace(row.name)
		remaining = append(remaining, row)
	}
	return remaining
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
	if ui.generating {
		ui.message = "Generation is already running."
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
	ui.generating = true
	ui.renderFooter()
	request := ui.request
	go func() {
		result, err := ui.generate(request)
		ui.queueUpdateDraw(func() {
			ui.finishGeneration(result, err)
		})
	}()
}

func (ui *tviewUI) finishGeneration(result application.GenerateResult, err error) {
	ui.generating = false
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
	ui.renderWorkspaceHeader()
	ui.renderSidebar()
	ui.renderTree()
	if ui.screen == tviewScreenPreview {
		ui.workspace.SwitchToPage("preview")
		info := fmt.Sprintf(" Output: %s\n Action: %s  │  Files: %d", tview.Escape(emptyDash(ui.plan.OutputDir)), tview.Escape(emptyDash(ui.plan.OutputAction)), ui.plan.FileCount)
		if ui.err != nil {
			info += "\n Error: " + tview.Escape(ui.err.Error())
		}
		ui.previewInfo.SetText(info)
		ui.renderFiles()
	} else {
		ui.workspace.SwitchToPage("detail")
		ui.renderDetail()
	}
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
		ui.sidebar.SetCell(row, 0, tview.NewTableCell(prefix+route.label).SetTextColor(color).SetExpansion(1).SetClickedFunc(func() bool {
			ui.open(route.screen)
			return true
		}))
	}
	ui.sidebar.Select(int(ui.screen), 0)
}

func (ui *tviewUI) renderDetail() {
	title := ui.screenTitle()
	if ui.screen == tviewScreenOverview || ui.screen == tviewScreenEntities {
		title = "Properties"
	}
	ui.detail.SetTitle(" " + title + " ")
	ui.detail.SetText(strings.Join(ui.detailLines(), "\n"))
}

func (ui *tviewUI) renderFiles() {
	ui.files.Clear()
	ui.files.SetCell(0, 0, tview.NewTableCell("Action").SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(1))
	ui.files.SetCell(0, 1, tview.NewTableCell("Path").SetTextColor(tcell.ColorLightCyan).SetSelectable(false).SetExpansion(4))
	for row, file := range ui.plan.Files {
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
	if len(ui.plan.Files) == 0 {
		ui.files.SetCell(1, 0, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
		ui.files.SetCell(1, 1, tview.NewTableCell("No planned files yet.").SetTextColor(tcell.ColorGray))
	}
}

func (ui *tviewUI) renderFooter() {
	action := "Esc workspace"
	switch ui.screen {
	case tviewScreenOverview:
		switch {
		case ui.selection.entity != "":
			action = "e/f fields"
		case ui.selection.service != "":
			action = "e entities"
		default:
			action = "e services"
		}
	case tviewScreenEntities:
		action = "e entities"
		if ui.selection.entity != "" {
			action += "  f fields"
		}
	case tviewScreenProject, tviewScreenServices, tviewScreenValueObjects, tviewScreenRelationships:
		action = "e edit " + strings.ToLower(ui.screenTitle())
	case tviewScreenGenerate:
		action = "g generate"
	}
	move := "↑/↓ route"
	switch ui.focus {
	case tviewFocusTree:
		move = "↑/↓ select"
	case tviewFocusDetails, tviewFocusFiles:
		move = "↑/↓ scroll"
	case tviewFocusShortcuts:
		move = "Esc workspace"
	}
	ui.footer.SetText(fmt.Sprintf(" [aqua]Tab[-] focus  %s  │  %s  │  r refresh  q quit\n%s", move, action, ui.workspaceStatus()))
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
	var lines []string
	switch ui.screen {
	case tviewScreenOverview, tviewScreenEntities:
		lines = ui.propertyLines()
	case tviewScreenProject:
		lines = append(lines, "", fmt.Sprintf("Description: %s", emptyDash(ui.plan.Config.SolutionDescription)), fmt.Sprintf("Target Framework: %s", emptyDash(ui.plan.Config.TargetFramework)), fmt.Sprintf("Solution Format: %s", emptyDash(ui.plan.Config.SolutionFormat)), fmt.Sprintf("Gateway: %t", ui.plan.Config.GatewayEnabled), "Press e to edit project settings in this dashboard.")
	case tviewScreenServices:
		lines = append(lines, "", "[aqua]Services[white]", "Press e to rename services in this dashboard.")
		lines = append(lines, prefixedList(ui.plan.Config.ServiceNames)...)
	case tviewScreenValueObjects:
		lines = append(lines, "", "[aqua]Value Objects[white]", "Press e to edit value objects in this dashboard.")
		lines = append(lines, valueObjectLines(ui.plan.Config.Services)...)
	case tviewScreenRelationships:
		lines = append(lines, "", "[aqua]Relationships[white]", "Press e to edit relationship endpoints, FK, requiredness, and navigation names in this dashboard.")
		lines = append(lines, relationshipLines(ui.plan.Config.Services)...)
	case tviewScreenGenerate:
		lines = append(lines, ui.generateSummary(), "", fmt.Sprintf("Output: %s", tview.Escape(emptyDash(ui.plan.OutputDir))),
			fmt.Sprintf("Action: %s  │  Files: %d", tview.Escape(emptyDash(ui.plan.OutputAction)), ui.plan.FileCount))
	case tviewScreenResult:
		lines = append(lines, "", ui.resultSummary())
	}
	if ui.err != nil {
		lines = append(lines, "", "[red]"+tview.Escape(ui.err.Error())+"[-]")
	}
	return lines
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

func relationshipLines(services []application.ServiceSummary) []string {
	var lines []string
	for _, service := range services {
		lines = append(lines, "  [yellow]"+service.Name+"[white]")
		if len(service.Relationships) == 0 {
			lines = append(lines, "    -")
			continue
		}
		for _, relationship := range service.Relationships {
			lines = append(lines, "    "+relationship.Summary)
		}
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
