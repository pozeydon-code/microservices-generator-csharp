# microgen roadmap

This is the versionable roadmap for `microgen`, the CLI-first .NET CRUD microservice generator. The generated Clean Architecture template is **core complete, production distribution not complete**: the architecture and generated CRUD path are implemented and tested, while public binary releases, package-manager distribution, and several quality hardening items remain.

## Current status

### Core complete

- The Go generator validates strict JSON configuration, plans deterministic files, and publishes through a staged, ownership-checked output path. `--force` is limited to verified generated output.
- Generated services include Domain, Application, Infrastructure, WebApi, and architecture/integration test projects.
- Create, List, GetById, Update, and Delete are generated CQRS vertical slices using MediatR, FluentValidation validators/pipeline behavior, repository ports, and ErrorOr result mapping.
- Generated coverage includes Domain/Application/WebApi/Architecture/Infrastructure tests, JWT behavior, SQL Server integration paths, concurrency handling, readiness checks, and value-object fixtures.
- The TUI exists as an adapter over the application/generator/output boundaries. It must not own generation rules.
- Current policy-backed target frameworks are `net8.0`, `net9.0`, and `net10.0`; `net8.0` is the minimum supported floor.

### Not complete

- `microgen version` and an explicit tag-triggered release workflow now exist as a release foundation.
- Reproducible cross-platform archives, checksums, GitHub artifact attestations, and SBOM publication are configured but still require repository-level review and verification before production distribution is claimed.
- Slice C now provides deterministic Homebrew, winget, and Chocolatey metadata rendering and validation from an explicit GitHub Release `checksums.txt`; publication, ecosystem ownership, and clean-machine verification are not established.
- The dependency policy is stored in the versioned manifest at `internal/generator/policy/dependency-policy.json` and verified in Go CI.
- Slice D quality hardening is complete: the documented Value Object witnesses validate, delete ordering is covered at Application and HTTP boundaries, EF ambiguous-commit semantics are explicit, and focused TUI/output safety coverage is present.

## Architecture map

Each configured service generates these projects:

| Project | Owns | May depend on |
|---|---|---|
| `{Service}.Domain` | Entities, value objects, domain factories, and business invariants | No framework, EF, ASP.NET, HTTP, or Infrastructure dependency |
| `{Service}.Application` | CQRS requests/handlers/validators, DTOs, ErrorOr outcomes, pagination, and repository ports | Domain only; no EF, ASP.NET, or Infrastructure |
| `{Service}.Infrastructure` | EF Core SQL Server, persistence mappings, SQL readiness, retries, and repository implementations | Application and Domain |
| `{Service}.WebApi` | ASP.NET controllers, composition root, authentication, middleware, health, and ErrorOr-to-HTTP mapping | Application and Infrastructure |
| `*.Architecture.Tests` | Generated dependency-boundary checks | Test-only references needed to inspect the generated projects |

Dependency direction is inward:

```text
Domain <- Application <- Infrastructure
Application + Infrastructure <- WebApi
```

Domain is framework-free. Application contains CQRS ports, handlers, validators, and ErrorOr but no EF, ASP.NET, or Infrastructure. Infrastructure owns EF and SQL. WebApi owns controllers, composition, and error mapping. Generated architecture tests inspect project references and source/runtime boundaries to enforce this direction.

## Completed work units

These commits are the implementation baseline for the roadmap:

| Work unit | Commit | Result |
|---|---|---|
| WebApi migration and Create CQRS slice | `77f1c78` | Replaced the generated API/host shape with WebApi composition, controllers, error mapping, and the first MediatR Create command/validator/handler path. |
| List/GetById CQRS migration | `0a083e1` | Moved generated read operations to MediatR queries, handlers, and pagination validation. |
| Update/Delete CQRS migration | `4fa54d6` | Moved generated mutations to commands, handlers, validators, ErrorOr mapping, and removed the legacy application-service path. |
| Target framework floor | `350b9f6` | Rejected unsupported legacy targets instead of allowing them through the `net8.0` minimum. |
| Central dependency policy | `733d9ff` | Centralized generated NuGet versions in `Directory.Packages.props`. |
| Verified target/dependency policy | `e76eed6` | Made generation fail when a target has no explicit verified policy entry. |
| Generated shell installer removal | `13f80e1` | Removed the generated shell installer; generated user output contains no installer script. |

## Package policy

- `Directory.Packages.props` is the generated workspace source of truth for NuGet versions. Generated project files must not independently drift package versions.
- EF Core and ASP.NET Core packages align to the generated target major. The EF SQL Server package follows the EF Core version.
- `Microsoft.Data.SqlClient` is independently pinned and audited; it is not inferred from the ASP.NET or EF train.
- MediatR, FluentValidation, and ErrorOr are independently versioned and must be explicitly verified rather than forced into framework-major alignment.
- Generation never resolves dynamic `latest` package versions and never installs packages. A target is selectable only when an explicit verified policy exists.
- Policy data is loaded from `internal/generator/policy/dependency-policy.json`; `internal/generator/target_framework.go` owns typed loading and validation only. Updates must change the manifest deliberately, keep `verified: true` only for independently verified pins, and pass `go test ./internal/generator -run '^Test(DependencyPolicy|LoadDependencyPolicies|ValidateTargetFrameworkPolicy)' -count=1` before the full Go and generated .NET checks.
- Adding a target requires a policy entry, generated `Directory.Packages.props` evidence, restore/build/test evidence, and a reviewable policy change. No target should be added by fallback version synthesis.

## Public distribution roadmap

The distribution path must preserve deterministic generation and must **never generate shell installers in user output**.

1. Add `microgen version` with version data injected by the release build and a deterministic development fallback.
2. Add reproducible cross-platform builds for Linux, macOS, and Windows, with stable archive names and a release verification job.
3. Use GitHub Releases as the public source of truth for binaries and release metadata.
4. Publish checksums, signing/provenance metadata, and an SBOM with every release. Document independent verification.
5. GoReleaser v2 is configured as the release automation candidate after stabilizing the version command and archive contract.
6. Render package-manager metadata from the same tagged GitHub Release artifacts: a future owned Homebrew tap, winget manifests, and Chocolatey packages. The checked-in renderer and manual handoff workflow do not publish; an optional PowerShell release installer may be provided as a distribution asset, never as generated project output.

## Open quality follow-ups

The Slice D quality follow-ups are closed. Future work must preserve these decisions:

- README and generated configuration examples use string witnesses that satisfy or violate the declared rules, and the repository test validates the documented values.
- Delete performs entity lookup before non-empty token decoding. A missing entity returns `NotFound` even for a malformed non-empty token; empty or whitespace tokens fail request validation first. Existing entities return validation for malformed tokens and conflict for stale valid tokens.
- EF keeps bounded transient retries and timeouts. A lost response during commit remains ambiguous and must not be interpreted as a definite rollback. No idempotency key or operation identity is generated; mutation retry safety belongs at the API/application boundary.

## Prioritized next slices

### A. Dependency policy manifest and update verification

| Field | Plan |
|---|---|
| Objective | Make target/package compatibility reviewable data instead of a Go-source-only policy. |
| Scope | Introduce a versioned manifest for target majors and package pins; render `Directory.Packages.props`; validate EF/ASP.NET alignment, independent SqlClient pinning, and independent MediatR/FluentValidation/ErrorOr verification; add update checks and dependency-update PR automation. |
| Non-goals | No new target major without evidence; no dynamic latest resolution; no package installation during generation; no release packaging. |
| Dependencies | Existing policy behavior in `internal/generator/target_framework.go`; generated golden workspaces; CI restore/build/test path. |
| Acceptance evidence | Manifest and rendered props agree; unsupported targets fail; supported targets restore/build/test; policy tests reject alignment drift and unverified versions; CI records the verification command and update PR path. |
| Review/commit boundary | One commit for manifest/rendering and one only if automation is independently reviewable; each must include policy tests and rollback to the current explicit policy. |

### B. Version command and release foundation

| Field | Plan |
|---|---|
| Objective | Establish a trustworthy, reproducible binary contract before package-manager publication. |
| Scope | Add `microgen version`; inject release version/build metadata; add cross-platform archive builds; publish GitHub Releases with checksums, signatures/provenance, and SBOM; evaluate and document GoReleaser. |
| Non-goals | No generated installer scripts; no package-manager publication in this slice; no calendar-based release promises. |
| Dependencies | Stable module/product identity, Slice A policy verification, current Go CI, and an agreed archive/version schema. |
| Acceptance evidence | Local development and tagged builds report expected version values; release CI produces reproducible Linux/macOS/Windows artifacts; a clean verifier checks checksums, signatures/provenance, and SBOM from GitHub Releases. |
| Review/commit boundary | Commit the version command separately from release automation. Keep archive/signing/SBOM changes in reviewable release work units with exact build and verification commands recorded. |

### C. Package-manager publication

| Field | Plan |
|---|---|
| Objective | Make stable GitHub Release artifacts discoverable through common package managers without creating a second binary source. |
| Scope | Render and validate Homebrew formula, winget version/locale/installer manifests, and Chocolatey nuspec/scripts from an explicit release tag plus `checksums.txt`; document upgrade, uninstall, independent verification, and ownership-gated publication. |
| Non-goals | No installer emitted into generated user projects; no package manager may rebuild or resolve a different binary; no publication before Slice B artifacts are verified. |
| Dependencies | Slice B GitHub Release contract, stable archive names, and package-manager account/repository ownership for any publication. |
| Acceptance evidence | The renderer requires all six current archive names and matching release checksums; each output resolves to an explicit GitHub Release asset; failed verification stops rendering. Clean-machine install/upgrade/uninstall scenarios and repository ownership remain required before publication. |
| Review/commit boundary | One publication unit per ecosystem when definitions differ materially; keep manifests and their verification tests together. Rollback means withdrawing the package metadata, not rewriting released binaries. |

### D. Config/docs quality and TUI/output safety

| Field | Plan |
|---|---|
| Objective | Close documented correctness gaps while preserving the CLI-first architecture and output trust boundary. **Complete.** |
| Scope | Fix the README value-object examples; specify delete malformed-token versus missing-entity ordering; resolve EF ambiguous-commit/idempotency guidance; add focused TUI tests for stale plans, force gating, narrow terminals, and output safety where coverage is missing. |
| Non-goals | No generator rule duplication in TUI; no unsafe overwrite mode; no broad feature expansion such as nested value objects, providers, or Docker. |
| Dependencies | Existing generated contracts, output writer ownership checks, current architecture tests, and the policy/release foundations where docs reference them. |
| Acceptance evidence | README examples validate; generated tests assert the chosen delete ordering and EF retry contract; TUI/output tests prove stale or unsafe plans cannot generate; `go test -race -timeout 2m ./...`, `go vet ./...`, and the generated .NET CI harness pass. |
| Review/commit boundary | Keep the README/config-contract correction separate from runtime behavior fixes. Keep TUI/output tests with the behavior they verify; do not mix this slice with release packaging. |

## Invariants for future work

- Keep generation CLI-first and UI-independent; TUI code adapts application ports rather than owning rules.
- Preserve deterministic files, strict config validation, staged publication, ownership checks, rollback, and no shell/database/package execution during generation.
- Keep Domain and Application free of infrastructure/framework concerns and retain generated architecture tests as the boundary gate.
- Do not claim production distribution readiness until release artifacts can be independently verified from GitHub Releases.
