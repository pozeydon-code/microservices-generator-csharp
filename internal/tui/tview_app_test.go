package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/application"
	"github.com/rivo/tview"
)

// Dispatch through Application.Run, including tview's mouse action translation
// and focused-primitive routing. The sentinel drains earlier queued events.
func runTViewTest(t *testing.T, ui *tviewUI, width, height int, screens ...tcell.SimulationScreen) func(...tcell.Event) {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if len(screens) > 0 {
		screen = screens[0]
	}
	ui.app.SetScreen(screen)
	screen.SetSize(width, height)
	ready, drained := make(chan struct{}, 1), make(chan struct{}, 1)
	drainAfterDraw := false
	ui.app.SetAfterDrawFunc(func(tcell.Screen) {
		if drainAfterDraw {
			drainAfterDraw = false
			drained <- struct{}{}
		}
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	capture := ui.app.GetInputCapture()
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF12 {
			drainAfterDraw = true
			return nil
		}
		return capture(event)
	})
	done := make(chan error, 1)
	go func() { done <- ui.app.SetRoot(ui.root, true).Run() }()
	t.Cleanup(func() {
		ui.app.Stop()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("TView run: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("TView did not stop")
		}
	})
	wait := func(ch <-chan struct{}, phase string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatalf("TView did not dispatch %s", phase)
		}
	}
	wait(ready, "initial draw")
	return func(events ...tcell.Event) {
		t.Helper()
		for _, event := range events {
			ui.app.QueueEvent(event)
		}
		ui.app.QueueEvent(tviewKey(tcell.KeyF12))
		wait(drained, "queued events")
	}
}

func tviewKey(key tcell.Key) *tcell.EventKey { return tcell.NewEventKey(key, 0, tcell.ModNone) }
func tviewRune(r rune) *tcell.EventKey       { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

func tviewClick(x, y int) []tcell.Event {
	return []tcell.Event{tcell.NewEventMouse(x, y, tcell.Button1, 0), tcell.NewEventMouse(x, y, tcell.ButtonNone, 0)}
}

func TestTViewDispatchChildApplyAndEscape(t *testing.T) {
	for _, key := range []tcell.Key{tcell.KeyCtrlS, tcell.KeyEnter} {
		t.Run(fmt.Sprint(key), func(t *testing.T) {
			var calls int
			var saved application.ServiceSettings
			ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil,
				func(_ application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
					calls++
					saved = settings
					return application.UpdateServiceSettingsResult{}, nil
				})
			state := &tviewServicesEditState{}
			for i := 1; i <= 20; i++ {
				state.rows = append(state.rows, tviewManagerRow{name: fmt.Sprintf("Service%d", i)})
			}
			ui.showServicesManager(state)
			send := runTViewTest(t, ui, 90, 24)
			manager := ui.app.GetFocus().(*tview.Table)
			send(tviewRune('a'), tviewKey(tcell.KeyEnter), tviewRune('X'), tviewKey(tcell.KeyEscape))
			if ui.app.GetFocus() != manager || calls != 0 {
				t.Fatalf("Escape did not return to the parent without saving (calls=%d)", calls)
			}
			row, col := manager.GetSelection()
			offset, _ := manager.GetOffset()
			if row != 21 || col != 0 || offset == 0 || manager.GetCell(21, 0).Text != "NewService21" {
				t.Fatalf("Escape lost draft, selection or scroll (row=%d col=%d offset=%d)", row, col, offset)
			}
			send(tviewKey(tcell.KeyEnter), tviewRune('Y'))
			if key == tcell.KeyEnter {
				send(tviewKey(tcell.KeyTAB))
			}
			send(tviewKey(key))
			if ui.app.GetFocus() != manager || calls != 0 || manager.GetCell(21, 0).Text != "NewService21Y" {
				t.Fatalf("child Apply did not update only the draft (calls=%d)", calls)
			}
			send(tviewKey(tcell.KeyCtrlS))
			if calls != 1 || len(saved.Services) != 21 || saved.Services[20].Name != "NewService21Y" {
				t.Fatalf("parent did not save applied draft exactly once (calls=%d)", calls)
			}
		})
	}
}

func TestTViewDispatchRulesApplyDoesNotSaveParent(t *testing.T) {
	var calls int
	var saved application.ValueObjectSettings
	ui := newTViewUI(application.GenerationPlan{}, application.GenerateRequest{}, nil, nil, nil, nil, nil, nil,
		func(_ application.GenerateRequest, settings application.ValueObjectSettings) (application.UpdateValueObjectSettingsResult, error) {
			calls++
			saved = settings
			return application.UpdateValueObjectSettingsResult{}, nil
		})
	ui.showValueObjectsManager(&tviewValueObjectsEditState{serviceName: "Example", rows: []tviewManagerRow{{original: "ExistingDraft", name: "ExistingDraft", typeName: "string"}}})
	send := runTViewTest(t, ui, 40, 12)
	manager := ui.app.GetFocus()
	send(tviewRune('a'), tviewRune('r'), tviewRune(' '), tviewKey(tcell.KeyEscape))
	if ui.app.GetFocus() != manager || calls != 0 {
		t.Fatal("Rules Escape did not return to parent without saving")
	}
	send(tviewRune('r'), tviewRune(' '))
	x, y, _, _ := ui.app.GetFocus().GetRect()
	send(tcell.NewEventMouse(x+1, y, tcell.WheelDown, 0))
	if input, ok := ui.app.GetFocus().(*tview.InputField); !ok || input.GetLabel() != "Min length" {
		t.Fatal("wheel did not advance the hovered Rules form to the next editable control")
	}
	send(tviewRune('3'), tviewKey(tcell.KeyCtrlS))
	if ui.app.GetFocus() != manager || calls != 0 {
		t.Fatalf("Rules Ctrl+S did not apply only to the parent draft (calls=%d)", calls)
	}
	send(tviewKey(tcell.KeyCtrlS))
	if calls != 1 || len(saved.ValueObjects) != 2 || saved.ValueObjects[0].Name != "ExistingDraft" || saved.ValueObjects[0].Validations.Required != nil || saved.ValueObjects[1].Validations.Required == nil || !*saved.ValueObjects[1].Validations.Required || saved.ValueObjects[1].Validations.MinLength == nil || *saved.ValueObjects[1].Validations.MinLength != 3 {
		t.Fatalf("parent save did not preserve the other row and applied Rules (calls=%d)", calls)
	}
}

func TestTViewDispatchConnectionMask(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{
		ServiceNames: []string{"FirstService", "SecondService"},
		Services:     []application.ServiceSummary{{Name: "SecondService", HasConnectionString: true}, {Name: "FirstService"}},
	}}
	var saves []application.ServiceSettings
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil, nil,
		func(_ application.GenerateRequest, settings application.ServiceSettings) (application.UpdateServiceSettingsResult, error) {
			saves = append(saves, settings)
			return application.UpdateServiceSettingsResult{}, nil
		})
	ui.open(tviewScreenServices)
	screen := tcell.NewSimulationScreen("")
	send := runTViewTest(t, ui, 60, 16, screen)
	send(tviewRune('e'))
	manager := ui.app.GetFocus().(*tview.Table)
	if manager.GetCell(1, 1).Text != "-" || manager.GetCell(2, 1).Text != "Set" {
		t.Fatal("connection status must be masked and mapped by service name")
	}
	open := tviewKey(tcell.KeyEnter)
	send(tviewRune('j'), tviewKey(tcell.KeyRight), open, tviewRune('Z'))
	cells, _, _ := screen.GetContents()
	masked := false
	for _, cell := range cells {
		for _, r := range cell.Runes {
			if r == 'Z' {
				t.Fatal("connection input disclosed synthetic text")
			}
			masked = masked || r == '*'
		}
	}
	if !masked {
		t.Fatal("connection input did not draw a password mask")
	}
	send(tviewKey(tcell.KeyEscape), tviewKey(tcell.KeyCtrlS))
	if len(saves) != 1 || saves[0].Services[1].ConnectionString != nil {
		t.Fatal("cancel must omit the connection update")
	}
	send(open, tviewKey(tcell.KeyCtrlS))
	if len(saves) != 1 || manager.GetCell(2, 1).Text != "-" {
		t.Fatal("child Apply must stage, not persist, the clear")
	}
	send(tviewKey(tcell.KeyCtrlS))
	if len(saves) != 2 || saves[1].Services[1].OriginalName != "SecondService" || saves[1].Services[1].ConnectionString == nil || *saves[1].Services[1].ConnectionString != "" || saves[1].Services[0].ConnectionString != nil {
		t.Fatal("parent Save did not explicitly clear only the selected connection")
	}
}

func TestTViewDispatchGenerationModeChoices(t *testing.T) {
	var saved []application.SolutionSettings
	plan := application.GenerationPlan{Config: application.ConfigSummary{GenerationMode: "simple"}}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil,
		func(_ application.GenerateRequest, settings application.SolutionSettings) (application.UpdateSolutionSettingsResult, error) {
			saved = append(saved, settings)
			return application.UpdateSolutionSettingsResult{}, nil
		})
	ui.open(tviewScreenProject)
	send := runTViewTest(t, ui, 60, 16)
	send(tviewRune('e'), tviewRune('k'), tviewKey(tcell.KeyEnter), tviewKey(tcell.KeyCtrlS))
	if len(saved) != 1 || saved[0].GenerationMode != "" || saved[0].GatewayEnabled == nil || !*saved[0].GatewayEnabled {
		t.Fatal("unrelated gateway change did not preserve mode omission")
	}
	saved = nil
	send(tviewRune('e'), tviewRune('j'), tviewRune('j'), tviewRune('j'), tviewRune('j'), tviewKey(tcell.KeyEnter))
	choice, ok := ui.app.GetFocus().(*tview.DropDown)
	if !ok {
		t.Fatalf("generation mode must be a choice, got %T", ui.app.GetFocus())
	}
	if _, value := choice.GetCurrentOption(); value != "simple" {
		t.Fatalf("mode chooser did not start on simple: %q", value)
	}
	send(tviewKey(tcell.KeyEnter), tviewKey(tcell.KeyDown), tviewKey(tcell.KeyEnter))
	if _, value := choice.GetCurrentOption(); value != "enterprise" {
		t.Fatalf("second mode choice is not enterprise: %q", value)
	}
	send(tviewKey(tcell.KeyCtrlS))
	if len(saved) != 0 {
		t.Fatal("mode Apply wrote the config before parent Save")
	}
	send(tviewKey(tcell.KeyCtrlS))
	if len(saved) != 1 || saved[0].GenerationMode != "enterprise" {
		t.Fatal("parent Save did not submit enterprise exactly once")
	}
}

func TestTViewDispatchEntityFieldsAndDelete(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{{
		Name: "Catalog", Entities: []application.EntitySummary{{Name: "Product"}},
	}}}}
	var saves int
	var saved application.FieldSettings
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil, nil, nil, nil,
		func(_ application.GenerateRequest, settings application.FieldSettings) (application.UpdateFieldSettingsResult, error) {
			saves++
			saved = settings
			return application.UpdateFieldSettingsResult{}, nil
		})
	ui.open(tviewScreenEntities)
	send := runTViewTest(t, ui, 60, 16)
	send(tviewRune('e'))
	manager := ui.app.GetFocus().(*tview.Table)
	clickCell := func(row, col int, want string) {
		t.Helper()
		cell := manager.GetCell(row, col)
		if cell.Text != want {
			t.Fatalf("entity action cell %d,%d = %q, want %q", row, col, cell.Text, want)
		}
		x, y, w := cell.GetLastPosition()
		if w == 0 {
			t.Fatalf("entity action %q is not visible", want)
		}
		send(tviewClick(x, y)...)
		if ui.app.GetFocus() != manager || saves != 0 {
			t.Fatalf("single click on %s activated it", want)
		}
		send(tviewClick(x, y)...)
	}
	clickCell(1, 2, "Delete")
	if ui.app.GetFocus() != manager || manager.GetCell(1, 2).Text != "Restore" {
		t.Fatal("Delete did not mark the row without opening Fields")
	}
	send(tviewRune('f'))
	if ui.app.GetFocus() != manager || ui.message != "Save the entity before editing fields." || saves != 0 {
		t.Fatal("Fields accepted a deleted entity")
	}
	// Select Restore via keyboard so the following click pair starts a new target.
	send(tviewKey(tcell.KeyEnter), tviewRune('a'), tviewRune('f'))
	if ui.app.GetFocus() != manager || ui.message != "Save the entity before editing fields." || saves != 0 {
		t.Fatal("Fields accepted a new unsaved entity")
	}
	clickCell(1, 1, "Fields")
	if ui.app.GetFocus() == manager {
		t.Fatal("Fields did not open the persisted entity's field editor")
	}
	send(tviewRune('a'), tviewKey(tcell.KeyCtrlS), tviewKey(tcell.KeyEscape))
	if saves != 1 || saved.EntityName != "Product" || len(saved.Fields) != 1 || saved.Fields[0].Type != "string" || ui.app.GetFocus() != manager {
		t.Fatal("primitive field was not saved once or Escape did not restore Entities")
	}
}

func TestTViewDispatchDoubleClickAddAfterName(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{ServiceNames: []string{"One", "Two"}}}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)
	ui.open(tviewScreenServices)
	send := runTViewTest(t, ui, 60, 16)
	send(tviewRune('e'))
	manager := ui.app.GetFocus().(*tview.Table)
	send(tviewKey(tcell.KeyTAB))
	button := ui.app.GetFocus().(*tview.Button)
	bx, by, _, _ := button.GetRect()
	x, y, _ := manager.GetCell(2, 0).GetLastPosition()
	send(append(tviewClick(x, y), tviewClick(bx+1, by)...)...)
	if got := manager.GetRowCount(); got != 3 || ui.app.GetFocus() != button {
		t.Fatalf("first Add click should only focus it; rows=%d", got)
	}
	send(tviewClick(bx+1, by)...)
	if got := manager.GetRowCount(); got != 4 {
		t.Fatalf("click Name then double-click Add: rows=%d, want 4", got)
	}
}

func TestTViewDispatchNarrowEntityFooter(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{Services: []application.ServiceSummary{{Name: "Catalog", Entities: []application.EntitySummary{{Name: "Product"}}}}}}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)
	ui.open(tviewScreenEntities)
	send := runTViewTest(t, ui, 40, 12)
	send(tviewRune('e'), tviewKey(tcell.KeyTAB))
	for _, label := range []string{"Add", "Service", "Fields", "Save", "Cancel"} {
		button, ok := ui.app.GetFocus().(*tview.Button)
		if !ok || button.GetLabel() != label {
			t.Fatalf("Tab did not focus %s", label)
		}
		x, y, w, h := button.GetRect()
		if x < 0 || y < 0 || w < 1 || h < 1 || x+w > 40 || y+h > 12 {
			t.Errorf("%s clipped at %d,%d %dx%d", label, x, y, w, h)
		}
		send(tviewKey(tcell.KeyTAB))
	}
}

func TestTViewDispatchWheelOverOpenModeList(t *testing.T) {
	plan := application.GenerationPlan{Config: application.ConfigSummary{GenerationMode: "simple"}}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil, noopUpdateSettings)
	ui.open(tviewScreenProject)
	send := runTViewTest(t, ui, 40, 12)
	send(tviewRune('e'), tviewRune('j'), tviewRune('j'), tviewRune('j'), tviewRune('j'), tviewKey(tcell.KeyEnter))
	choice, ok := ui.app.GetFocus().(*tview.DropDown)
	if !ok {
		t.Fatal("mode is not a dropdown")
	}
	send(tviewKey(tcell.KeyEnter))
	list, ok := ui.app.GetFocus().(*tview.List)
	if !ok || !choice.IsOpen() {
		t.Fatal("choice list did not open")
	}
	x, y, _, _ := list.GetRect()
	for _, wheel := range []tcell.ButtonMask{tcell.WheelDown, tcell.WheelUp} {
		send(tcell.NewEventMouse(x+1, y, wheel, 0))
		if got := ui.app.GetFocus(); got != list || !choice.IsOpen() {
			t.Fatalf("wheel over open mode list changed focus from List to %T", got)
		}
	}
	send(tviewKey(tcell.KeyDown), tviewKey(tcell.KeyEnter))
	if _, value := choice.GetCurrentOption(); value != "enterprise" || choice.IsOpen() {
		t.Fatal("keyboard could not select enterprise after using the wheel")
	}
}

func TestTViewDispatchMouse(t *testing.T) {
	plan := application.GenerationPlan{}
	for i := 0; i < 20; i++ {
		plan.Files = append(plan.Files, application.PlannedFile{Path: fmt.Sprintf("file%d", i), Action: "create"})
	}
	ui := newTViewUI(plan, application.GenerateRequest{}, nil, nil)
	send := runTViewTest(t, ui, 60, 16)
	x, y, _ := ui.sidebar.GetCell(int(tviewScreenServices), 0).GetLastPosition()
	send(tviewClick(x, y)...)
	if ui.screen != tviewScreenServices || ui.app.GetFocus() != ui.sidebar {
		t.Fatal("sidebar click did not open the selected tab and focus the sidebar")
	}
	x, y, _ = ui.files.GetCell(1, 0).GetLastPosition()
	send(tviewClick(x, y)...)
	if ui.app.GetFocus() != ui.files || ui.focus != tviewFocusFiles {
		t.Fatal("file click did not synchronize native and keyboard focus")
	}
	send(tviewKey(tcell.KeyTAB))
	if ui.app.GetFocus() != ui.footer {
		t.Fatal("Tab did not continue from the mouse-focused panel")
	}
	send(tcell.NewEventMouse(x, y, tcell.WheelDown, 0))
	if offset, _ := ui.files.GetOffset(); offset == 0 || ui.app.GetFocus() != ui.footer {
		t.Fatal("wheel did not scroll the hovered file tab without stealing focus")
	}
	send(tviewRune('e'))
	manager := ui.app.GetFocus().(*tview.Table)
	for i := 0; i < 20; i++ {
		send(tviewRune('a'))
	}
	x, y, _, _ = manager.GetRect()
	before, _ := manager.GetOffset()
	send(tcell.NewEventMouse(x+1, y+1, tcell.WheelUp, 0))
	if after, _ := manager.GetOffset(); after >= before || ui.app.GetFocus() != manager {
		t.Fatalf("wheel did not scroll the hovered manager (before=%d after=%d)", before, after)
	}
	footer := ui.modalFocus[len(ui.modalFocus)-1].(*tview.Form)
	add := footer.GetButton(0)
	x, y, w, h := add.GetRect()
	if w == 0 || h == 0 || x >= 60 || y >= 16 {
		t.Fatalf("Add button is clipped at narrow size: %d,%d %dx%d", x, y, w, h)
	}
	count := manager.GetRowCount()
	send(tviewClick(x+1, y)...)
	if manager.GetRowCount() != count || ui.app.GetFocus() != add {
		t.Fatal("single button click must focus without activation")
	}
	send(tviewClick(x+1, y)...)
	if manager.GetRowCount() != count+1 {
		t.Fatal("double click did not activate Add exactly once")
	}
	send(tviewKey(tcell.KeyEnter))
	if manager.GetRowCount() != count+2 {
		t.Fatal("Enter on the selected button no longer activates it")
	}
	row, _ := manager.GetSelection()
	x, y, _ = manager.GetCell(row, 0).GetLastPosition()
	send(tviewClick(x, y)...)
	if ui.app.GetFocus() != manager {
		t.Fatal("single cell click should select without opening its editor")
	}
	send(tviewClick(x, y)...)
	if _, ok := ui.app.GetFocus().(*tview.InputField); !ok {
		t.Fatal("double cell click should open the selected editor")
	}
}
