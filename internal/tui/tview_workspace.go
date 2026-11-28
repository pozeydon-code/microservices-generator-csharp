package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
	"github.com/rivo/tview"
)

// Keep identity separate from table row positions; plan refresh can reorder rows.
type workspaceSelection struct{ service, entity string }

func savedWorkspaceName(rows []tviewManagerRow, original string) string {
	if original != "" {
		for _, row := range rows {
			if row.original == original && !row.deleted {
				return strings.TrimSpace(row.name)
			}
		}
	}
	return ""
}

type workspaceDashboard struct {
	*tview.Flex
	body, rail *tview.Flex
	routes     *tview.Table
}

func (dashboard *workspaceDashboard) Draw(screen tcell.Screen) {
	_, _, width, height := dashboard.GetRect()
	railWidth := 24
	if width >= 140 {
		railWidth = 34
	}
	dashboard.body.ResizeItem(dashboard.rail, min(railWidth, width/2), 0)
	dashboard.rail.ResizeItem(dashboard.routes, min(len(tviewRoutes)+2, max(1, (height-3)*2/3)), 0)
	dashboard.Flex.Draw(screen)
}

func (ui *tviewUI) buildWorkspace() *workspaceDashboard {
	ui.header = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	ui.header.SetBackgroundColor(tcell.ColorDarkSlateGray)
	ui.tree = tview.NewTable().SetSelectable(true, false)
	ui.tree.SetBorder(true).SetTitle(" Solution tree ").SetTitleAlign(tview.AlignLeft)
	ui.tree.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal).Bold(true))

	rail := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.sidebar, len(tviewRoutes)+2, 0, true).
		AddItem(ui.tree, 0, 1, false)
	preview := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(ui.previewInfo, 3, 0, false).AddItem(ui.files, 0, 1, true)
	ui.workspace = tview.NewPages().AddPage("detail", ui.detail, true, true).AddPage("preview", preview, true, false)
	body := tview.NewFlex().AddItem(rail, 24, 0, true).AddItem(ui.workspace, 0, 1, false)
	dashboard := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ui.header, 1, 0, false).
		AddItem(body, 0, 1, true).
		AddItem(ui.footer, 2, 0, false)
	return &workspaceDashboard{Flex: dashboard, body: body, rail: rail, routes: ui.sidebar}
}

func (ui *tviewUI) renderTree() {
	ui.tree.SetSelectionChangedFunc(nil).Clear()
	selected, fallback := -1, -1
	add := func(label string, selection workspaceSelection) {
		row := ui.tree.GetRowCount()
		ui.tree.SetCell(row, 0, tview.NewTableCell(label).SetExpansion(1).SetReference(selection))
		if selection == ui.selection {
			selected = row
		}
		if selection.service == ui.selection.service && selection.entity == "" {
			fallback = row
		}
	}
	for _, service := range ui.plan.Config.Services {
		add("▾ "+service.Name, workspaceSelection{service: service.Name})
		for i, entity := range service.Entities {
			branch := "  ├ "
			if i == len(service.Entities)-1 {
				branch = "  └ "
			}
			add(branch+entity.Name, workspaceSelection{service.Name, entity.Name})
		}
	}
	if ui.tree.GetRowCount() == 0 {
		ui.tree.SetCell(0, 0, tview.NewTableCell("No services yet").SetSelectable(false))
	} else if selected < 0 {
		selected = max(0, fallback)
		if fallback < 0 && len(ui.plan.Config.Services[0].Entities) > 0 {
			selected = 1
		}
	}
	ui.selection, _ = ui.tree.GetCell(max(0, selected), 0).GetReference().(workspaceSelection)
	if row, _ := ui.tree.GetSelection(); row != selected && selected >= 0 {
		ui.tree.Select(selected, 0)
	}
	ui.tree.SetSelectionChangedFunc(func(row, _ int) {
		selection, ok := ui.tree.GetCell(row, 0).GetReference().(workspaceSelection)
		if !ok || selection == ui.selection {
			return
		}
		ui.selection = selection
		if ui.screen != tviewScreenOverview && ui.screen != tviewScreenEntities {
			ui.open(tviewScreenOverview)
			return
		}
		ui.detail.ScrollToBeginning()
		ui.renderDetail()
		ui.renderFooter()
	})
	ui.tree.SetSelectedFunc(func(_, _ int) { ui.open(tviewScreenOverview) })
}

func (ui *tviewUI) selectedEntityIndex(service application.ServiceSummary) int {
	for i, entity := range service.Entities {
		if entity.Name == ui.selection.entity {
			return i
		}
	}
	return -1
}

func (ui *tviewUI) editSelectedFields() {
	if service, ok := serviceSummaryByName(ui.plan.Config.Services, ui.selection.service); ok {
		if index := ui.selectedEntityIndex(service); index >= 0 {
			ui.showFieldsManager(tviewFieldsStateFromEntity(service.Name, service.Entities[index]))
		}
	}
}

func (ui *tviewUI) propertyLines() []string {
	escape := tview.Escape
	service, ok := serviceSummaryByName(ui.plan.Config.Services, ui.selection.service)
	if !ok {
		return []string{"No services yet. Open Services to add one.", "", readinessLines(ui.plan.Readiness)}
	}
	target := escape(service.Name)
	index := ui.selectedEntityIndex(service)
	if index >= 0 {
		target += " / " + escape(service.Entities[index].Name)
	}
	connection := "not configured"
	if service.HasConnectionString {
		connection = "configured"
	}
	lines := []string{"[aqua::b]" + target + "[-::-]", "Connection: " + connection}

	objects := make(map[string]bool, len(service.ValueObjects))
	for _, name := range service.ValueObjectNames {
		objects[name] = true
	}
	for _, object := range service.ValueObjects {
		objects[object.Name] = true
	}
	if index >= 0 {
		fields := service.Entities[index].Fields
		lines = append(lines, "", "[::b]Entity fields[-::-]")
		if len(fields) == 0 {
			lines = append(lines, "  No fields yet")
		}
		for _, field := range fields {
			kind := "scalar"
			if objects[field.Type] {
				kind = "value object"
			}
			lines = append(lines, fmt.Sprintf("  %s: %s (%s)", escape(field.Name), escape(field.Type), kind))
		}
	} else {
		lines = append(lines, fmt.Sprintf("Entities: %d", len(service.Entities)))
	}
	lines = append(lines, "", "[::b]Service value objects[-::-]")
	switch {
	case len(service.ValueObjects) > 0:
		for _, object := range service.ValueObjects {
			lines = append(lines, fmt.Sprintf("  %s: %s (reusable)", escape(object.Name), escape(object.Type)))
		}
	case len(service.ValueObjectNames) > 0:
		for _, name := range service.ValueObjectNames {
			lines = append(lines, "  "+escape(name)+" (reusable)")
		}
	default:
		lines = append(lines, "  None")
	}
	return lines
}

func (ui *tviewUI) renderWorkspaceHeader() {
	mode := "Enterprise"
	if strings.EqualFold(ui.plan.Config.GenerationMode, "simple") {
		mode = "Simple"
	}
	ui.header.SetText(fmt.Sprintf(" [::b]microgen[-::-] / %s  │  Mode: %s", tview.Escape(emptyDash(ui.plan.Config.SolutionName)), mode))
}

func (ui *tviewUI) workspaceStatus() string {
	state := "No draft"
	switch {
	case ui.generating:
		state = "Generating"
	case ui.planStale:
		state = "Plan stale"
	case (ui.plan.ForceRequired || ui.plan.Readiness.OutputForceRequired) && !ui.request.Force:
		state = "--force required"
	case ui.err != nil:
		state = "Action failed"
	}
	if ui.message == "" {
		return " " + state
	}
	return " " + state + "  │  " + tview.Escape(ui.message)
}
