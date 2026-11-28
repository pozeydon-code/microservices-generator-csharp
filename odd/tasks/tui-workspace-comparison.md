# TUI workspace design comparison
Compare the user-validated TView interface with proposed terminal-native improvements before implementation.
Branch: design/tui-workspace-comparison. Baseline: eac1aab. Status: implementation in progress.

## Specs
S1. User: "listo lo probe y quedo de maravilla... ya lo uni a main... ahora si queda empezar con el rediseño y validar si se queda com oesta o se mejora con tu propuesta"
S2. User: "vale ahora me gustaria que revisaras todo el tui... revisalo de tal forma que la ui se vea actual bonita intuitiva (quizas a lo lazygit o a lo lazydocker)"
S3. User: "a ver primero corrige los fallos funcionales y luego en otra rama hacemos el rediseño"
S4. User: "si", accepting "¿Querés que lo implemente ahora en la rama de rediseño, conservando los comportamientos que ya validaste?" after selecting "Árbol + propiedades — recomendada". Chosen visual specification: design/tui-workspace/01-workspace.html.

## Tasks
T1 [S1-S3] Route: bounded design writer. Create three terminal-expressible alternatives and a faithful labeled current-layout reference for Services → Entities → Fields; preserve actual runtime unchanged. Commit: pending.
T2 [S1-S3] Route: parent. Complete: structural and JS checks passed; user chose tree + properties and authorized implementation. No browser rendering proof claimed for prototypes.
T3 [S1-S4] Route: bounded writer. Implement real TView tree + properties workspace with focused behavioral and layout checks. Commit: pending.
T4 [S1-S4] Route: independent verifier and parent. Verify preserved functionality, narrow layout and visual captures, run full suite and document limitations. Commit: pending.

## Constraints
Comparison phase preserved runtime unchanged. Implementation of the selected tree + properties layout is now authorized. No new dependencies, publishing or releases. Artboards are previews, not evidence of runtime implementation. Inline self-contained HTML with monospace cell grid, fixed character sizes, foreground/background colors, weight, terminal glyphs and thin dividers only; no web-style cards, shadows, rounded controls or external assets. Proposed palette is a starter, not an extracted project design system. Baseline reference must match current panel structure using code and existing captures as evidence, with explicit limitations. Preserve simple/enterprise, primitive fields and value objects, safe connection status (never raw credentials), draft/save semantics, keyboard/mouse access and narrow adaptation. Compare structures, not merely colors. The selected implementation is 01-workspace.html; other artboards remain alternatives. Preserve all existing routes and real generation controls; do not implement prototype-only sample actions or unsupported Required field schema. Connection status remains presence-only. At 80x24 keep controls reachable with focused-pane adaptation rather than clipping.

## Log
L1. User: "listo lo probe y quedo de maravilla... ya lo uni a main... ahora si queda empezar con el rediseño y validar si se queda com oesta o se mejora con tu propuesta"
L2. Main and origin/main point to eac1aab; clean at start. User verified functional fixes manually. gentle_design list found no design sources or system; skip sync, propose starter tokens in preview only.
L3. Added `design/tui-workspace/`: `00-current-reference.html` (static current-layout schematic), `01-workspace.html` (recommended tree + properties), `02-master-detail.html` (focused editor), and `03-guided-workspace.html` (guided steps). These are comparison artifacts only; runtime unchanged and no implementation choice made.

L4. User selected "Árbol + propiedades — recomendada", then answered "si" to implementing it while preserving validated behavior. Native design handoff selected tui-workspace/01-workspace.html; alternatives remain comparison-only. RDD mode status globally off at implementation start.
