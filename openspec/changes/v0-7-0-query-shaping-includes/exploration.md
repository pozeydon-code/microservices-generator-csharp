## Exploration: v0.7.0 query shaping/includes

### Current State
The generator already models relationships in `spec.Config`, but only as structural links: multiplicity, principal/dependent entities, FK metadata, requiredness, and navigation names. The view builder turns those into entity relationship scalars, reference/collection navigations, and EF mapping data.

Read-side generation is still flat. `List*QueryHandler` and `Get*ByIdQueryHandler` call repository methods, then map snapshots to DTOs that contain only entity fields, relationship scalar FKs, and the concurrency token. The infrastructure repository uses `OrderBy/Skip/Take` and `FindAsync`; it does not emit `Include`/`ThenInclude`, query-specific projections, or shape options.

The TUI can edit relationships, but only endpoint/FK/required/navigation data. There is no query-shaping or include-specific user input yet.

### Affected Areas
- `internal/spec/config.go` — relationship/config model would need new query-shape/include metadata.
- `internal/spec/config_test.go` — validation/canonicalization coverage would need new cases.
- `internal/application/service.go` — plan/summarization and config update flow would need to carry the new settings.
- `internal/application/service_test.go` — relationship summary/readiness tests would need new expectations.
- `internal/generator/view_builder.go` — service/entity view data would need query-shape/include fields.
- `internal/generator/templates/application/*.tmpl` — read handlers, DTOs, and query validators may need new shape-aware code.
- `internal/generator/templates/infrastructure/repository-implementation.tmpl` — repository would need include/projection logic.
- `internal/generator/templates/webapi/controller.tmpl` — query parameters may need to expose include/shape controls.
- `internal/generator/generator_test.go` and `internal/generator/testdata/golden/**` — deterministic output and golden coverage would need updates.
- `internal/tui/model.go` / `internal/tui/tview_app.go` — relationship editor screens are the likely UI entry point if shaping is configurable in-app.

### Approaches
1. **Explicit query-shape profiles** — add config-driven include profiles for read queries/endpoints, then generate repository/query-handler code that applies those includes deterministically.
   - Pros: explicit, reviewable, keeps behavior predictable, fits Clean Architecture boundaries.
   - Cons: adds config surface and generator complexity.
   - Effort: Medium/High

2. **Implicit relationship-based shaping** — infer includes from relationship type and emit a fixed eager-loading strategy (for example, always include configured references on `GetById`).
   - Pros: smaller config change, faster to author.
   - Cons: opaque over-fetching risk, harder to tune, less deterministic for nested graphs.
   - Effort: Medium

### Recommendation
Use explicit query-shape profiles. For this generator, query shaping should mean read-side only, config-driven include/projection decisions that stay deterministic and keep EF details inside the infrastructure boundary. Default list/get endpoints should remain shallow unless the author opts into deeper includes.

### Risks
- Include graphs can explode file size and test golden churn if depth is unbounded.
- Nested DTOs can create circular or over-coupled shapes if the boundaries are not kept explicit.
- TUI and validation changes can cascade across plan summaries, readiness hints, and generated tests.

### Ready for Proposal
Yes — the next step is to write the proposal/spec for explicit read-side include profiles and define the exact config shape, default depth, and generated API surface.
