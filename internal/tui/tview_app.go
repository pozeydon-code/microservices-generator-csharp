package tui

import (
	"fmt"
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
	form := tview.NewForm()
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	form.AddDropDown("Service", serviceNames, selectedIndex(serviceNames, service.Name), func(_ string, index int) {
		if index >= 0 && index < len(serviceNames) && serviceNames[index] != service.Name {
			ui.openEntitiesEditForService(serviceNames[index])
		}
	})
	entityNames := entitySummaryNames(service.Entities)
	selectedEntityIndex := 0
	form.AddDropDown("Fields for", entityNames, selectedEntityIndex, func(_ string, index int) {
		if index >= 0 && index < len(entityNames) && index != selectedEntityIndex {
			ui.openEntitiesEditForServiceAndEntity(service.Name, index)
		}
	})
	ui.addEntityInputs(form, service, selectedEntityIndex)
	form.AddButton("Save", func() { ui.saveEntitiesEdit(form, service, selectedEntityIndex) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, " Edit Entities ")
}

func (ui *tviewUI) openEntitiesEditForServiceAndEntity(serviceName string, selectedEntityIndex int) {
	service, ok := serviceSummaryByName(ui.plan.Config.Services, serviceName)
	if !ok {
		ui.openEntitiesEditForService(serviceName)
		return
	}
	form := tview.NewForm()
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	form.AddDropDown("Service", serviceNames, selectedIndex(serviceNames, service.Name), func(_ string, index int) {
		if index >= 0 && index < len(serviceNames) && serviceNames[index] != service.Name {
			ui.openEntitiesEditForService(serviceNames[index])
		}
	})
	entityNames := entitySummaryNames(service.Entities)
	if selectedEntityIndex < 0 || selectedEntityIndex >= len(entityNames) {
		selectedEntityIndex = 0
	}
	form.AddDropDown("Fields for", entityNames, selectedEntityIndex, func(_ string, index int) {
		if index >= 0 && index < len(entityNames) && index != selectedEntityIndex {
			ui.openEntitiesEditForServiceAndEntity(service.Name, index)
		}
	})
	ui.addEntityInputs(form, service, selectedEntityIndex)
	form.AddButton("Save", func() { ui.saveEntitiesEdit(form, service, selectedEntityIndex) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, " Edit Entities ")
}

func (ui *tviewUI) addEntityInputs(form *tview.Form, service application.ServiceSummary, selectedEntityIndex int) {
	for index, entity := range service.Entities {
		addEditInputField(form, fmt.Sprintf("Entity %d", index+1), entity.Name)
		form.AddCheckbox("Delete", false, nil)
	}
	addEditInputField(form, "New entity", "")
	if len(service.Entities) > 0 && selectedEntityIndex >= 0 && selectedEntityIndex < len(service.Entities) {
		for index, field := range service.Entities[selectedEntityIndex].Fields {
			addEditInputField(form, fmt.Sprintf("Field %d name", index+1), field.Name)
			addEditInputField(form, fmt.Sprintf("Field %d type", index+1), field.Type)
			form.AddCheckbox("Delete field", false, nil)
		}
	}
	addEditInputField(form, "New field name", "")
	addEditInputField(form, "New field type", "string")
}

func (ui *tviewUI) saveEntitiesEdit(form *tview.Form, service application.ServiceSummary, selectedEntityIndex int) {
	if ui.updateEntities == nil {
		ui.closeEditWithMessage("Entity editing is not available.")
		return
	}
	settings := application.EntitySettings{ServiceName: service.Name, Entities: make([]application.EntityNameSetting, 0, len(service.Entities))}
	indexOffset := 2
	for index, entity := range service.Entities {
		name := formInputText(form, indexOffset+(index*2))
		deleted := formCheckboxChecked(form, indexOffset+(index*2)+1)
		if name == "" || deleted {
			continue
		}
		settings.Entities = append(settings.Entities, application.EntityNameSetting{OriginalName: entity.Name, Name: name})
	}
	newEntityName := formInputText(form, indexOffset+(len(service.Entities)*2))
	if newEntityName != "" {
		settings.Entities = append(settings.Entities, application.EntityNameSetting{Name: newEntityName})
	}
	result, err := ui.updateEntities(ui.request, settings)
	if err != nil || result.PlanError != nil {
		ui.applyEntitiesSaveResult(result, err)
		return
	}
	selectedEntityName, selectedEntityKept := entityNameAfterEdit(form, service, selectedEntityIndex, indexOffset)
	if selectedEntityKept && ui.updateFields != nil {
		fieldResult, fieldErr := ui.saveFieldsFromEntitiesEdit(form, service, selectedEntityIndex, selectedEntityName, indexOffset+(len(service.Entities)*2)+1)
		ui.applyFieldsSaveResult(fieldResult, fieldErr)
		return
	}
	ui.applyEntitiesSaveResult(result, nil)
}

func (ui *tviewUI) saveFieldsFromEntitiesEdit(form *tview.Form, service application.ServiceSummary, selectedEntityIndex int, entityName string, offset int) (application.UpdateFieldSettingsResult, error) {
	entity := service.Entities[selectedEntityIndex]
	settings := application.FieldSettings{ServiceName: service.Name, EntityName: entityName, Fields: make([]application.FieldSetting, 0, len(entity.Fields))}
	for index, field := range entity.Fields {
		name := formInputText(form, offset+(index*3))
		typeName := formInputText(form, offset+(index*3)+1)
		deleted := formCheckboxChecked(form, offset+(index*3)+2)
		if name == "" || deleted {
			continue
		}
		settings.Fields = append(settings.Fields, application.FieldSetting{OriginalName: field.Name, Name: name, Type: typeName})
	}
	newFieldOffset := offset + (len(entity.Fields) * 3)
	newFieldName := formInputText(form, newFieldOffset)
	newFieldType := formInputText(form, newFieldOffset+1)
	if newFieldName != "" {
		settings.Fields = append(settings.Fields, application.FieldSetting{Name: newFieldName, Type: newFieldType})
	}
	return ui.updateFields(ui.request, settings)
}

func entityNameAfterEdit(form *tview.Form, service application.ServiceSummary, selectedEntityIndex, offset int) (string, bool) {
	if selectedEntityIndex < 0 || selectedEntityIndex >= len(service.Entities) {
		return "", false
	}
	name := formInputText(form, offset+(selectedEntityIndex*2))
	deleted := formCheckboxChecked(form, offset+(selectedEntityIndex*2)+1)
	return name, name != "" && !deleted
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
	form := tview.NewForm()
	serviceNames := serviceSummaryNames(ui.plan.Config.Services)
	form.AddDropDown("Service", serviceNames, selectedIndex(serviceNames, service.Name), func(_ string, index int) {
		if index >= 0 && index < len(serviceNames) && serviceNames[index] != service.Name {
			ui.openValueObjectsEditForService(serviceNames[index])
		}
	})
	for index, valueObject := range service.ValueObjects {
		addEditInputField(form, fmt.Sprintf("Value object %d name", index+1), valueObject.Name)
		addEditInputField(form, fmt.Sprintf("Value object %d type", index+1), valueObject.Type)
		form.AddCheckbox("Delete", false, nil)
	}
	addEditInputField(form, "New value object name", "")
	addEditInputField(form, "New value object type", "string")
	form.AddButton("Save", func() { ui.saveValueObjectsEdit(form, service) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, " Edit Value Objects ")
}

func (ui *tviewUI) saveValueObjectsEdit(form *tview.Form, service application.ServiceSummary) {
	if ui.updateValueObjects == nil {
		ui.closeEditWithMessage("Value object editing is not available.")
		return
	}
	settings := application.ValueObjectSettings{ServiceName: service.Name, ValueObjects: make([]application.ValueObjectNameSetting, 0, len(service.ValueObjects))}
	for index, valueObject := range service.ValueObjects {
		name := formInputText(form, 1+(index*3))
		typeName := formInputText(form, 1+(index*3)+1)
		deleted := formCheckboxChecked(form, 1+(index*3)+2)
		if name == "" || deleted {
			continue
		}
		settings.ValueObjects = append(settings.ValueObjects, application.ValueObjectNameSetting{OriginalName: valueObject.Name, Name: name, Type: typeName, Validations: validationRuleSettingsFromSummary(valueObject.Validations)})
	}
	newValueObjectOffset := 1 + (len(service.ValueObjects) * 3)
	newValueObjectName := formInputText(form, newValueObjectOffset)
	newValueObjectType := formInputText(form, newValueObjectOffset+1)
	if newValueObjectName != "" {
		settings.ValueObjects = append(settings.ValueObjects, application.ValueObjectNameSetting{Name: newValueObjectName, Type: newValueObjectType})
	}
	result, err := ui.updateValueObjects(ui.request, settings)
	ui.applyValueObjectsSaveResult(result, err)
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
	form := tview.NewForm()
	services := append([]string(nil), ui.plan.Config.ServiceNames...)
	for index, name := range services {
		addEditInputField(form, fmt.Sprintf("Service %d", index+1), name)
		form.AddCheckbox("Delete", false, nil)
	}
	addEditInputField(form, "New service", "")
	form.AddButton("Save", func() { ui.saveServicesEdit(form, services) })
	form.AddButton("Cancel", ui.closeEdit)
	ui.showEditForm(form, " Edit Services ")
}

func (ui *tviewUI) saveServicesEdit(form *tview.Form, original []string) {
	if ui.updateServices == nil {
		ui.closeEditWithMessage("Service editing is not available.")
		return
	}
	settings := application.ServiceSettings{Services: make([]application.ServiceNameSetting, 0, len(original))}
	for index, current := range original {
		name := formInputText(form, index*2)
		deleted := formCheckboxChecked(form, index*2+1)
		if name == "" || deleted {
			continue
		}
		settings.Services = append(settings.Services, application.ServiceNameSetting{OriginalName: current, Name: name})
	}
	newServiceName := formInputText(form, len(original)*2)
	if newServiceName != "" {
		settings.Services = append(settings.Services, application.ServiceNameSetting{Name: newServiceName})
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
