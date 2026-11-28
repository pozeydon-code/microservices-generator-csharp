# TUI workspace design comparison
Compare the user-validated TView interface with proposed terminal-native improvements before implementation.
Branch: design/tui-workspace-comparison. Baseline: eac1aab. Status: implemented and verified; user manual comparison pending.

## Specs
S1. User: "listo lo probe y quedo de maravilla... ya lo uni a main... ahora si queda empezar con el rediseño y validar si se queda com oesta o se mejora con tu propuesta"
S2. User: "vale ahora me gustaria que revisaras todo el tui... revisalo de tal forma que la ui se vea actual bonita intuitiva (quizas a lo lazygit o a lo lazydocker)"
S3. User: "a ver primero corrige los fallos funcionales y luego en otra rama hacemos el rediseño"
S4. User: "si", accepting "¿Querés que lo implemente ahora en la rama de rediseño, conservando los comportamientos que ya validaste?" after selecting "Árbol + propiedades — recomendada". Chosen visual specification: design/tui-workspace/01-workspace.html.

## Tasks
T1 [S1-S3] Route: bounded design writer. Complete: comparison artifacts and baseline schematic. Commit: dff9d66.
T2 [S1-S3] Route: parent. Complete: structural and JS checks passed; user chose tree + properties and authorized implementation. No browser rendering proof claimed for prototypes.
T3 [S1-S4] Route: bounded writer. Complete: native tree/properties workspace fb706fb, diagnostics corrections 702218a and 11f7939.
T4 [S1-S4] Route: independent verifier and parent. Complete: S1-S4 approved independently, final full suite and vet passed; live terminal captures inspected. User physical-terminal acceptance remains pending.

## Constraints
Comparison phase preserved runtime unchanged. Implementation of the selected tree + properties layout is now authorized. No new dependencies, publishing or releases. Artboards are previews, not evidence of runtime implementation. Inline self-contained HTML with monospace cell grid, fixed character sizes, foreground/background colors, weight, terminal glyphs and thin dividers only; no web-style cards, shadows, rounded controls or external assets. Proposed palette is a starter, not an extracted project design system. Baseline reference must match current panel structure using code and existing captures as evidence, with explicit limitations. Preserve simple/enterprise, primitive fields and value objects, safe connection status (never raw credentials), draft/save semantics, keyboard/mouse access and narrow adaptation. Compare structures, not merely colors. The selected implementation is 01-workspace.html; other artboards remain alternatives. Preserve all existing routes and real generation controls; do not implement prototype-only sample actions or unsupported Required field schema. Connection status remains presence-only. At 80x24 keep controls reachable with focused-pane adaptation rather than clipping.

## Log
L1. User: "listo lo probe y quedo de maravilla... ya lo uni a main... ahora si queda empezar con el rediseño y validar si se queda com oesta o se mejora con tu propuesta"
L2. Main and origin/main point to eac1aab; clean at start. User verified functional fixes manually. gentle_design list found no design sources or system; skip sync, propose starter tokens in preview only.
L3. Added `design/tui-workspace/`: `00-current-reference.html` (static current-layout schematic), `01-workspace.html` (recommended tree + properties), `02-master-detail.html` (focused editor), and `03-guided-workspace.html` (guided steps). These are comparison artifacts only; runtime unchanged and no implementation choice made.

L4. User selected "Árbol + propiedades — recomendada", then answered "si" to implementing it while preserving validated behavior. Native design handoff selected tui-workspace/01-workspace.html; alternatives remain comparison-only. RDD mode status globally off at implementation start.

L5. Implemented native tree + properties in fb706fb. RED/GREEN actual TView dispatch at 80x24,140x40,180x50; full serialized Go suite and vet passed. Parent inspected actual ANSI-rasterized live terminal captures /tmp/gpnIvGbP/tmp.uKsHFLyxKs/after-80.png and after-140.png. Worker before captures inaccessible to parent; no desktop-emulator visual fidelity claim.
L6. Independent verification found baseline regressions: Preview refresh error reason hidden; long Result cleanup instruction hidden in footer; populated Overview readiness hints omitted. Durable regressions observed RED, then GREEN with corrections 702218a and 11f7939. Final independent S1-S4 approval on 11f7939; cross-service targeting, rename/delete selection, modal save/cancel, wheel, stale/force gates and CLI rejected starter overwrite preserved. Verifier scratch retained by request at /tmp/gpnIvGbP/gazG5NTk/tmp.XVvIJ2gIp9.
L7. Final `go test -p 1 ./... -count=1 && go vet ./... && git diff --check` exited 0 after 68s. Native review declined for corrected candidate sha256:796ac04470c14efb73b219af6b0b8057ffc5c7e4ae4895c92fb6fd3e171bc6aa; no lineage created or native approval claimed. No push, merge or release. Canvas handoff marked implemented; user physical WSL comparison remains.
L8. Risk: item 3, shared input/focus/context contracts; independent verification completed. Review boundaries: design selection dff9d66 (263 lines), native workspace fb706fb (579 lines, exceeds approximate 400-line slice), Preview regression 702218a (49 lines), Result/readiness regression 11f7939 (86 lines). Workspace rendering and editor integration kept coherent with tests; delivery shape not selected.
