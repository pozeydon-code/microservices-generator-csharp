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
	app      *tview.Application
	root     tview.Primitive
	sidebar  *tview.Table
	detail   *tview.TextView
	files    *tview.Table
	footer   *tview.TextView
	plan     application.GenerationPlan
	request  application.GenerateRequest
	planFunc PlanFunc
	generate GenerateFunc
	result   application.GenerateResult
	err      error
	message  string
	screen   tviewScreen

	editRequested bool
}

var runTViewApplication = func(app *tview.Application, root tview.Primitive) error {
	return app.SetRoot(root, true).EnableMouse(true).Run()
}

func newTViewUI(plan application.GenerationPlan, request application.GenerateRequest, planFunc PlanFunc, generate GenerateFunc) *tviewUI {
	ui := &tviewUI{
		app:      tview.NewApplication(),
		plan:     plan,
		request:  request,
		planFunc: planFunc,
		generate: generate,
		screen:   tviewScreenOverview,
	}
	ui.root = ui.build()
	ui.refresh()
	return ui
}

func (ui *tviewUI) build() tview.Primitive {
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

	root := tview.NewFlex().
		AddItem(ui.sidebar, 24, 0, true).
		AddItem(main, 0, 1, false)

	ui.app.SetInputCapture(ui.handleKey)
	return root
}

func (ui *tviewUI) handleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		ui.app.Stop()
		return nil
	case tcell.KeyEscape:
		ui.open(tviewScreenOverview)
		return nil
	}

	switch strings.ToLower(string(event.Rune())) {
	case "q":
		ui.app.Stop()
		return nil
	case "j":
		ui.move(1)
		return nil
	case "k":
		ui.move(-1)
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
		ui.editRequested = true
		ui.app.Stop()
		return nil
	}
	return event
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
	shortcuts := "[aqua]j/k[white] route  [aqua]enter[white] open  [aqua]e[white] edit  [aqua]r[white] refresh  [aqua]g[white] generate  [aqua]esc[white] overview  [aqua]q[white] quit"
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
	if ui.err != nil {
		lines = append(lines, "[red]"+ui.err.Error()+"[white]")
	}

	switch ui.screen {
	case tviewScreenProject:
		lines = append(lines, "", fmt.Sprintf("Description: %s", emptyDash(ui.plan.Config.SolutionDescription)), fmt.Sprintf("Target Framework: %s", emptyDash(ui.plan.Config.TargetFramework)), fmt.Sprintf("Solution Format: %s", emptyDash(ui.plan.Config.SolutionFormat)), fmt.Sprintf("Gateway: %t", ui.plan.Config.GatewayEnabled))
	case tviewScreenServices:
		lines = append(lines, "", "[aqua]Services[white]")
		lines = append(lines, prefixedList(ui.plan.Config.ServiceNames)...)
	case tviewScreenEntities:
		lines = append(lines, "", "[aqua]Entities[white]")
		lines = append(lines, entityLines(ui.plan.Config.Services)...)
	case tviewScreenValueObjects:
		lines = append(lines, "", "[aqua]Value Objects[white]")
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
