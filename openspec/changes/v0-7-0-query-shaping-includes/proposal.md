# Proposal: v0.7.0 Query Shaping Includes

## Intent

Add explicit read-side query-shape profiles so generated C# services can return useful related data without exposing EF Core mechanics or unbounded graph expansion.

## Scope

### In Scope
- Add named query-shape profiles with bounded include depth and validation.
- Keep default list/get endpoints shallow unless a profile is configured or selected.
- Expose constrained WebAPI `shape=<profile>` selection using stable names such as `summary` and `detail`.
- Generate acyclic read DTOs and infrastructure-owned EF Core shaping/projection.

### Out of Scope
- Raw client include paths such as `include=Orders.Customer`.
- Circular, unbounded, or full-domain DTO graph generation.
- Allowing multiple sibling collection includes by default.
- Write-side shape behavior or mutation endpoint changes.

## Capabilities

### New Capabilities
- `query-shaping-includes`: Read profiles, include validation, shape-specific DTOs, infrastructure EF Core shaping, and WebAPI profile selection.

### Modified Capabilities
- None. No existing `openspec/specs/` source specs are present to modify.

## Approach

Model query shapes as explicit config profiles selected by stable names. Validate profiles before generation: reject circular paths, unbounded depth, unknown relationships, and multiple sibling collection includes. Prefer projection-to-DTO for known read outputs; keep `Include`, `ThenInclude`, `AsSplitQuery`, tracking, and projection inside infrastructure templates.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/spec/config.go` | Modified | Query-shape profile model and validation. |
| `internal/application/service.go` | Modified | Plan summaries carry shapes. |
| `internal/generator/view_builder.go` | Modified | Shape metadata for templates. |
| `internal/generator/templates/application/*.tmpl` | Modified | Shape-aware DTOs, queries, and handlers. |
| `internal/generator/templates/infrastructure/repository-implementation.tmpl` | Modified | EF Core projection/include implementation. |
| `internal/generator/templates/webapi/controller.tmpl` | Modified | Constrained `shape` query parameter. |
| `internal/tui/*` | Modified | Optional profile editing surface. |
| `internal/generator/testdata/golden/**` | Modified | Deterministic generated output coverage. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn across generated layers | Med | Keep profiles small and deterministic. |
| Unsafe EF query expansion | Med | Reject circular/unbounded paths and sibling collections by default. |
| Public API lock-in | Low | Expose profile names only, not path syntax. |

## Rollback Plan

Revert the query-shape config fields, validation, templates, WebAPI `shape` parameter, and golden outputs. Existing shallow list/get behavior remains the fallback baseline.

## Dependencies

- EF Core generated infrastructure remains the only layer using EF-specific query mechanics.

## Success Criteria

- [ ] Config validation rejects circular, unbounded, unknown, and multiple sibling collection profiles.
- [ ] Generated list/get endpoints remain shallow by default.
- [ ] Generated WebAPI accepts only configured shape names.
- [ ] DTOs are acyclic and shape-specific.
- [ ] Golden tests prove deterministic output for shallow and shaped reads.
