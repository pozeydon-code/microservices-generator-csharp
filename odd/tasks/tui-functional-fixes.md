# TUI functional fixes
Fix the shipped TView behavior before a separate visual redesign.
Baseline: 39516b4. Branch: fix/tui-functional. Status: verified; physical-terminal acceptance remains manual.

## Specs
S1. User: "a ver primero corrige los fallos funcionales y luego en otra rama hacemos el rediseño"
S2. User: "tambien me gustaria que aparte de que los modales tengan su Apply puedan guardarse con ctrl + S como lo demas"
S3. User: "tambien veo que no puedo usar la rueda para bajar o subir dentro de modales y tabs..."
S4. User: "los clics tampoco funciona... todo es por tabs, deberia dejar dar clic para seleccionar algo (en el caso de los botones doble clic para que no haya missclic)"
S5. User: "No tengo ningun lado para modificar si es la version simple o la enterprise"
S6. User: "tampoco hay un campo para agregar el connection string en el tui"
S7. User: "tambien no encontre ningun lugar donde pueda configurar los items dentro de las entidades o sea las propiedades (NO TODO SERA UN VALUE OBJECT)"

## Tasks
T1 [S1-S7] Route: bounded writer. Complete. Commits: 853758a (JSON encoding), 60590c5 (input, modal lifecycle and safe service editing).
- T1a [S1-S4]: TView input dispatch, modal lifecycle, and narrow control reachability.
- T1b [S5, S7]: Project mode choices and separate entity Fields/Delete controls. Implemented; focused checks green (see L3–L4).
- T1c [S6]: Connection editing, persistence, summaries, and generated JSON. Implemented; focused checks green (see L4).
T2 [S1-S7] Route: independent verifier. Complete against simulated TCell/TView event dispatch; S1-S7 passed. Evidence recorded in this document.

## Implementation boundaries
No visual redesign, new dependencies, publishing, releases, history rewrites or external database connections. Fix actual TView runtime, not just legacy Bubble Tea. Preserve keyboard behavior and cancellation. Child Ctrl+S must run that child's Apply exactly once, not silently persist unrelated parent drafts. Escape must return to the previous editor with its draft, selection and scroll. Native mouse selection must synchronize keyboard focus; wheel follows the hovered panel. Buttons require double click while a single click selects/focuses; Enter still activates. Remove disconnected dummy mouse code.
Use real simple/enterprise choices; preserve omitted mode on unrelated settings updates. Separate Fields and Delete hit targets with existing guards. Ensure narrow terminal editor controls remain reachable without undertaking a style redesign.
Connection strings must be JSON escaped, clearable explicitly while omitted updates preserve them, mapped by service name rather than array position, and masked in summaries. Do not expose credentials in error messages/tests/screenshots. Use only synthetic values in tests. Preserve existing plaintext config compatibility; document that config/generated settings are not a secret store and environment variables are recommended for real credentials.

## Log
L1. User: "a ver primero corrige los fallos funcionales y luego en otra rama hacemos el rediseño"
L2. Confirmed baseline audit: custom double-click handler disconnected; child forms lack Ctrl+S; Escape closes edit flow; mode is free text; Fields | Delete invokes only Fields; narrow fixed-size modals clip; connection strings unmasked, raw JSON interpolation, blank does not clear. Prior v0.9.1 completion claims are not proof. Existing TUI suite passes despite these gaps. RDD global off at start.
L3. T1b: Real TView dispatch reproduced the free-text mode and missing Delete hit target; both regressions passed after correction. Public application settings tests passed disk round-trips for simple and enterprise, mode omission, and invalid-mode rejection without changing disk. `go test ./internal/tui ./internal/application -short -count=1` passed TUI and failed `TestServicePlanGenerationReturnsUIReadyFileListWithoutWriting`: expected blank summary mode, got enterprise. That test and summary implementation were unchanged by T1b.
L4. T1c: Focused regressions reproduced raw summary exposure, incorrect TView status mapping, and invalid generated JSON. All three passed after the fix; disk checks distinguish omission, replacement, and explicit clear. Corrected only the missing default enterprise field in the L3 test's expected summary. TUI, application, configloader, and generator short suites passed; full verification remains pending.
L5. Follow-up to independent verification: at 40×12, wheel events over an open mode list moved focus to Apply (S3); a Name click immediately followed by two Add clicks failed to activate Add (S4). Both failed in durable dispatch regressions before the fix. The wheel now routes to the active list and click pairs are resolved by focus target and position. The narrow Entities footer and connection rejection/generation probes also passed. TUI and application short suites passed; the full suite was not rerun for this follow-up.

L6. Final independent verification passed S1-S7, including the two corrected mouse edge cases (7 focused tests). Final `go test -p 1 ./... -count=1`, `go vet ./...`, and `git diff --check` exited 0. Differential against 39516b4 for version and rejected-config generate/tui returned identical exit status, stdout and stderr. No physical terminal or production database validation was performed; TView simulations cover interaction, not terminal emulator fidelity.
L7. RDD remained globally off per `gentle-ai review mode status`. The facade ASSESS reported an inconsistent on projection and could not assess the untracked test/spec; no review lineage was started and no native approval is claimed.
L8. Review slices: 853758a has 39 changed lines; 60590c5 has 1,244 changed lines including 430 lines of interaction tests. The second slice exceeds the approximate 400-line review budget: modal lifecycle, input dispatch and callback payload changes were kept together for a coherent working TView boundary. No publishing or delivery shape selected; split review consideration remains for delivery.
L9. Risk: item 2 (credential masking/handling) and item 3 (shared application settings and input contracts). Independent verification completed; plaintext config/generated-settings compatibility is intentional and documented. Visual redesign remains excluded and must use a separate branch.
