# Changelog

Release notes for `microgen`, the C#/.NET microservices generator.

## [0.8.0] - 2026-11-26

Simplification and flexibility release. Adds generation modes, unifies error handling, simplifies logging, and consolidates test projects.

### Added

- **Generation modes**: Support for `simple` and `enterprise` modes via `generation.mode` config field
  - Simple mode: Direct handlers without CQRS, no MediatR, no EntitySnapshot (~60 files per service)
  - Enterprise mode: Full Clean Architecture with CQRS, MediatR, EntitySnapshot (~70 files per service)
- **Incremental generation foundation**: Manifest system for tracking generated files (full implementation in v0.8.1)

### Changed

- **Error handling**: Unified on ErrorOr library throughout. Removed custom `DomainResult<T>` and `DomainReconstitutionException`
- **Repository logging**: Removed verbose SQL diagnostics for Value Objects, simplified to standard structured logging
- **Test project consolidation**: Reduced from 5 test projects to 3:
  - `UnitTests` (Domain + Application tests)
  - `IntegrationTests` (Infrastructure + WebApi tests)
  - `ArchitectureTests` (Boundary verification tests)
- Test project naming: Simplified to `UnitTests`, `IntegrationTests`, `ArchitectureTests` (without service name prefix)

### Technical Details

- Added `mode` field to `GenerationOptions` in spec/config.go with validation
- Created separate template sets for simple/enterprise modes
- Updated all command/query handlers to use ErrorOr consistently
- Simplified repository-implementation.tmpl by removing FindReconstitutionDiagnosticsAsync
- Consolidated test project templates and updated namespaces
- Updated generator tests and golden files for new structure

### Known Limitations

- Incremental generation is partially implemented (manifest tracking only). Full selective regeneration will be available in v0.8.1

## [0.7.0] - 2026-11-26

Query shaping support for generated services. Adds explicit control over EF Core Include patterns.

### Added

- **Query shapes**: Named read profiles with bounded depth for controlling EF Core Include behavior
  - Config-driven via `queryShapes` array in service definition
  - Each shape specifies name and defaultDepth (1-3 levels)
  - Validation ensures no circular references or unbounded graphs
- **Shape-specific DTOs**: Generator creates dedicated DTOs per shape (e.g., `ProductSummaryDto`, `ProductDetailDto`)
- **Shaped repository methods**: Generated repositories include `ListShaped_{name}Async` and `GetShaped_{name}Async` methods per configured shape
- **WebAPI shape parameter**: Controllers accept `?shape=name` query parameter with validation against configured shapes
- **Infrastructure-owned shaping**: All Include/ThenInclude logic stays in Infrastructure layer, maintaining Clean Architecture boundaries

### Technical Details

- Added `QueryShape` type with name and defaultDepth fields to spec/config.go
- Added `QueryShapeView` to view_builder.go for template rendering
- Updated dto.tmpl with conditional logic for shape-specific DTO generation
- Updated repository-implementation.tmpl to generate shaped query methods
- Updated controller.tmpl to accept and validate shape parameter
- Added comprehensive golden tests in testdata/golden/query-shapes/
- All tests passing (except pre-existing NuGet vulnerability warnings in runtime tests)

## [0.4.1] - 2026-09-02

Hardening-only release for release hygiene, generated-output compile confidence, and TUI async generation clarity.

### Changed

- Documented the v0.4.1 release as defensive hardening rather than new relationship functionality.
- Added validation coverage for an existing single-service one-to-many relationship fixture.
- Clarified TUI generation progress, success, and failure state so stale output does not remain visible after async completion.

### Scope

- It does not add one-to-one relationship support.
- It does not add many-to-many relationship support.
- It does not add cross-service relationship support.
- It does not change the JSON schema or generated relationship semantics.

## [0.1.0] - 2026-07-26

First public release.

### Added

- Initial public `microgen` generator for deterministic C#/.NET CRUD microservice scaffolds from strict JSON configuration.
- Clean Architecture output split into Domain, Application, Infrastructure, WebApi, and architecture-test projects, with CQRS commands, queries, handlers, validators, and repository ports.
- CLI and Bubble Tea TUI workflows over the same planning and generation core, including guided project setup, service/entity configuration, preview, validation, and safe generation.
- Centralized dependency and target-framework metadata for policy-backed `net8.0`, `net9.0`, and `net10.0` output, including verified package versions and central NuGet package management.
- Cross-platform release workflow for Linux, macOS, and Windows archives on amd64 and arm64, with SHA-256 checksums, SPDX JSON archive SBOMs, and GitHub artifact attestations.
- Deterministic package-manifest handoff for Homebrew, winget, and Chocolatey based on an explicit GitHub Release tag and its `checksums.txt` file.

### Distribution Status

- GitHub Releases are the intended public binary source.
- Homebrew, winget, and Chocolatey publication is not yet configured or published. Generated package metadata remains a reviewable handoff artifact; publisher and package ownership, credentials, repository submissions, and clean-machine verification are still required.
