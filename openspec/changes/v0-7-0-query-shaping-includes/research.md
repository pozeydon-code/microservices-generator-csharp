---
schema: gentle-ai.sdd-research/v1
change: v0-7-0-query-shaping-includes
revision: 1
outcome: done
accessed_at: 2026-09-07
---

# Research: v0.7.0 Query Shaping Includes

## Admission

| Field | Value |
|---|---|
| Requested classes | documentation |
| Observed exact grants | documentation via Context7 Microsoft EF Core documentation lookup and local repository docs/readback |
| Not requested | open-web |
| Outcome | done |

## Questions

1. What does current EF Core documentation recommend or warn about for eager loading with `Include` / `ThenInclude`, especially multiple collection includes and performance?
2. What does EF Core documentation say about projections (`Select`) versus eager loading full entity graphs for read endpoints?
3. What safe constraints should a code generator impose for deterministic query shaping/includes to avoid circular graphs, over-fetching, or cartesian explosion?
4. What implications should this have for generated Clean Architecture boundaries: repository/infrastructure, application query handlers/DTOs, and WebAPI query params?

## Sources

| ID | Class | Title | Publisher | URL | Accessed | Excerpt |
|---|---|---|---|---|---|---|
| S1 | documentation | Eager Loading of Related Data | Microsoft EF Core docs | https://github.com/dotnet/entityframework.docs/blob/main/entity-framework/core/querying/related-data/eager.md | 2026-09-07 | EF Core uses `Include` to populate navigation properties; multiple relationships can be included, but collection navigations in a single query can cause performance issues and split queries may be considered. |
| S2 | documentation | Efficient Querying: Project Only Properties You Need | Microsoft EF Core docs | https://github.com/dotnet/entityframework.docs/blob/main/entity-framework/core/performance/efficient-querying.md | 2026-09-07 | EF Core fetches whole entity instances by default; `Select` should be used to project only required properties, reducing data transfer and memory, especially for read-only scenarios. |
| S3 | documentation | Tracking vs. No-Tracking Queries | Microsoft EF Core docs | https://github.com/dotnet/entityframework.docs/blob/main/entity-framework/core/querying/tracking.md | 2026-09-07 | Projections that do not include entity types do not track entities, which supports efficient read-only query shapes. |
| S4 | documentation | Single vs. Split Queries | Microsoft EF Core docs | https://github.com/dotnet/entityframework.docs/blob/main/entity-framework/core/querying/single-split-queries.md | 2026-09-07 | Multiple sibling collection includes can create cartesian explosion, multiplying transferred rows as sibling joins are added. |
| S5 | documentation | EF Core 5.0 Split Queries | Microsoft EF Core docs | https://github.com/dotnet/entityframework.docs/blob/main/entity-framework/core/what-is-new/ef-core-5.0/whatsnew.md | 2026-09-07 | Splitting a query with related collections into multiple SQL statements can improve performance, but can return inconsistent results if data changes between queries unless stronger transaction modes are used. |
| S6 | documentation | Exploration: v0.7.0 Query Shaping Includes | Local OpenSpec artifact | openspec/changes/v0-7-0-query-shaping-includes/exploration.md | 2026-09-07 | Current exploration recommends explicit query-shape profiles, shallow defaults, read-side-only behavior, and config-driven include/projection decisions. |

## Validated Claims

| ID | Claim | Sources |
|---|---|---|
| C1 | EF Core eager loading uses `Include`/`ThenInclude` to populate related navigation properties, and multiple paths may be included when the endpoint truly requires related data. | S1 |
| C2 | EF Core documentation warns that eager loading collection navigations in a single query can cause performance issues; multiple sibling collection includes are specifically dangerous because they can produce cartesian explosion. | S1, S4 |
| C3 | Split queries are an EF Core mitigation for collection include performance problems, but they trade one large query for multiple SQL statements and can observe inconsistent data if concurrent changes occur without appropriate transaction isolation. | S5 |
| C4 | For read-only endpoints, EF Core documentation recommends projecting only needed properties with `Select` instead of materializing full entity instances when only a subset of data is required. | S2 |
| C5 | Projections that do not include entity instances avoid entity tracking, which aligns with read-side DTO generation. | S3 |
| C6 | The current generator exploration already favors explicit query-shape profiles, shallow default list/get endpoints, read-side-only behavior, and config-driven shaping. | S6 |

## Implications for v0.7.0

### Generator constraints

- Query shapes should be explicit named profiles, not inferred from every configured relationship. This preserves deterministic generated output and avoids accidental over-fetching. (C1, C2, C6)
- Default list and get endpoints should remain shallow unless a profile is selected or configured as the endpoint default. (C4, C6)
- Profiles should impose bounded depth and reject circular navigation paths during config validation. A generator should treat include paths as a finite directed traversal, not as an open entity graph expansion. (C2, C6)
- Profiles should limit collection includes, especially sibling collections at the same level. If more than one sibling collection include is configured, generation should either reject it, require an explicit split-query mode, or emit a clear validation warning depending on the product policy selected by the orchestrator. (C2, C3)
- Read profiles should prefer projection-to-DTO shapes over eager-loading full entities when endpoint output is known. Includes should remain an infrastructure implementation detail for cases where entity materialization is intentionally required. (C4, C5)

### Clean Architecture boundaries

- Repository/infrastructure should own EF Core-specific mechanics: `Include`, `ThenInclude`, `AsSplitQuery`, projection expressions, and tracking behavior. Application code should not assemble EF include chains directly. (C1, C3, C5)
- Application query handlers should select an allowed profile by stable name and map to DTOs designed for that read model. They should not expose arbitrary navigation paths or EF Core syntax. (C4, C6)
- DTOs should be shape-specific and acyclic. Nested DTOs should be generated only for configured read profiles and should not mirror the full domain graph by default. (C2, C4, C6)
- WebAPI query parameters, if exposed, should be constrained to allowed profile names such as `shape=summary|detail`, not raw `include=Orders.Customer...` paths. This keeps the public API deterministic and prevents client-driven graph expansion. (C2, C6)

## Contradictions

None found in admitted sources. EF Core supports eager loading and split queries, but the documentation frames them as tools with performance and consistency tradeoffs rather than default behavior for every read endpoint.

## Uncertainty and Freshness

- Context7 returned Microsoft EF Core documentation from the `dotnet/entityframework.docs` repository on 2026-09-07.
- Product policy remains non-authoritative here: whether multiple sibling collection includes should be rejected outright or allowed only with explicit split-query mode must be confirmed by the orchestrator before proposal admission.

## Non-Authoritative Product Choices Pending Confirmation

- Recommended default: explicit named profiles, shallow endpoint defaults, bounded include depth, projection-first read DTOs, and profile-name-only WebAPI shape selection.
- Recommended safety policy: reject circular include paths and unbounded depth; require explicit opt-in for split-query behavior when a profile includes multiple collection navigations.
