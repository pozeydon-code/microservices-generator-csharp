# Microgen terminal design

Microgen helps developers model a solution, services, entities, and fields before reviewing planned files. The design question is whether moving model edits into a persistent workspace improves on the validated dashboard and its managers. Retaining today's interface is a valid outcome.

## Proposed constraints (not extracted tokens)

- Use system monospace, one character grid, modest contrast between panes, a single cyan accent, and a labeled warning role. Selection, keyboard focus, and unsaved changes must remain distinguishable without color.
- Treat 80×24 as a complete working view, not a compressed desktop. At 140×40, use room for more context, not more decoration. Put the active target and its next action first; keep shortcuts near that target.
- Keep Simple and Enterprise explicit. Primitive fields belong to an entity; reusable value-object definitions belong to the service and are shown separately. Show connection presence only, never connection strings.
- Preserve mouse and keyboard access, predictable focus, and explicit save/cancel outcomes. A model draft, a saved configuration, and a generation plan are different states. A save failure should retain the draft; a stale plan should not look ready.

## Comparison

See `design/tui-workspace/design.json`. The static baseline preserves the route rail, details, planned files, shortcuts, and the separate manager workflow. The selected direction is tree + properties (`design/tui-workspace/01-workspace.html`), implemented in TView. The view navigation and solution tree share a scrollable rail beside properties at 80×24; wider terminals give the rail more columns. Files occupy the main pane only on Preview. Editing reuses the real settings managers instead of the prototype's synthetic actions. The other artboards remain alternatives and make no filesystem or network changes.
