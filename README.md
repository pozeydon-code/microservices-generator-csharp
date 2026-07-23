# microgen: .NET CRUD microservice generator

`microgen` generates deterministic .NET CRUD microservice scaffolds from JSON. The generator stays CLI-first and UI-independent; the TUI is an adapter over the same planning and generation core, not a place for business rules.

## Quick path from a fresh clone

Prerequisites: Go 1.22+ for the generator and a .NET SDK matching the configured generated target framework.

```bash
go test ./...
go vet ./...
go run ./cmd/microgen generate --config examples/product-service.json --output ./out
dotnet build ./out/CommercePlatform.sln -warnaserror
```

Run a generated API with a connection string supplied by environment/configuration:

```bash
ConnectionStrings__DefaultConnection='Server=sql.example.com;Database=Products;Encrypt=True;TrustServerCertificate=False' \
Authentication__Authority='https://issuer.example.com' \
Authentication__Audience='ProductService' \
  dotnet run --project ./out/src/ProductService/ProductService.Host/ProductService.Host.csproj
```

For local development only, `TrustServerCertificate=True` can be used against a trusted local SQL Server instance. Production connection strings should validate certificates.

The generated `/health/live` endpoint does not access SQL Server. `/health/ready` checks database connectivity and verifies generated tables exist. Readiness failure responses are generic; detailed SQL/schema reasons are logged server-side only.

Generated Hosts redirect HTTP to HTTPS and enable HSTS outside Development/Testing. Production traffic must use HTTPS all the way to the Host, including private backend TLS from a trusted reverse proxy, unless operators explicitly implement and audit trusted forwarded-header configuration. Do not treat arbitrary edge TLS termination as sufficient.

Generated Hosts emit vendor-neutral OpenTelemetry traces and metrics with `service.name`, `service.version`, and `deployment.environment` resource attributes from deployment configuration. Configure `OTEL_EXPORTER_OTLP_ENDPOINT` for a collector or explicitly set `OTEL_SDK_DISABLED=true` to opt out; silent telemetry loss is not assumed safe. Production alert provisioning is deployment-specific; reasonable starting alerts are HTTP 5xx error rate above 2% for 5 minutes and p95 server latency above 1 second for 10 minutes.

## Generated structure

For each configured service, `microgen` emits:

| Project | Responsibility |
|---|---|
| `{Service}.Domain` | Enterprise entities, reusable Value Objects, and business validation; no framework, persistence, or rowversion dependency. |
| `{Service}.Application` | Use cases, ports, DTOs, neutral validation outcomes, opaque concurrency tokens, and pagination contracts; references Domain only. |
| `{Service}.Api` | HTTP endpoint adapter and health HTTP mapping; references Application only. |
| `{Service}.Infrastructure` | EF Core SQL Server adapter, shadow rowversion conversion, bounded retries/timeouts, repositories, and SQL/schema readiness adapter registration. |
| `{Service}.Host` | Executable composition root; wires auth, HTTPS/HSTS, request timeout budget, OpenTelemetry, ProblemDetails, middleware, API endpoints, Infrastructure, and startup guards. |
| `{Service}.Architecture.Tests` | Runtime, project-file, and source-text tests proving generated dependency boundaries. |
| root `.sln` or `.slnx` | References every generated project deterministically. |

Dependency direction:

```text
Domain <- Application <- Api
Domain <- Application <- Infrastructure
Application + Api + Infrastructure <- Host
```

## Generated dependency policy

Generated workspaces centralize NuGet versions in `Directory.Packages.props`. The generator chooses ASP.NET Core, EF Core, SqlClient, and `System.Security.Cryptography.Xml` versions from a target-framework dependency policy table, so `net10.0` and future targets are deliberate generator behavior rather than scattered template literals.

NuGet audit remains enabled in generated workspaces. `CentralPackageTransitivePinningEnabled` is also enabled so the central XML package pin can override vulnerable transitives; NuGet still rejects unsafe downgrades with NU1109 instead of hiding audit failures.

Generated workspaces centralize quality defaults in `Directory.Build.props`: nullable reference types, implicit usings, SDK recommended analyzers, code-style enforcement during build, and warnings-as-errors. The runtime harness keeps generated `net10.0` `.slnx` output warning-clean by restoring, building, and testing it when `dotnet` is available.

When `dotnet` is available, the generator test harness validates a generated `net10.0` `.slnx` workspace with restore, build, and test runtime commands.

## Command

```bash
microgen generate --config <path> --output <dir> [--force]
microgen tui --config <path> --output <dir> [--force]
microgen tui --new --config <path> --output <dir> [--force]
```

- `--config`: path to a strict JSON config file.
- `--output`: directory where files are planned or published.
- `--force`: replaces only verified, unchanged directories previously generated by `microgen`.
- `--new`: for `microgen tui`, creates a minimal starter JSON config at `--config` before opening the TUI. It refuses to overwrite an existing file.

`microgen tui` opens a terminal UI over the same planning and generation core. Use existing JSON with `--config <path>`, or start from scratch with `--new --config <path>` and a required `--output <dir>`. The starter config includes schema/generation defaults, solution metadata, and one service/entity with `Guid` identity plus `string` name fields so the generator can plan immediately; the TUI confirms that the starter config was created and can be edited incrementally.

The TUI is organized as a step-based dashboard: `Source`, `Project`, `Services`, `Preview`, and `Generate`. Use `tab` or `]` to move forward and `shift+tab` or `[` to move backward. `Source` confirms the JSON config source and output path, `Project` edits solution metadata and target framework, `Services` shows a read-only service/entity/value-object summary, `Preview` shows impact and planned files, and `Generate` focuses final confirmation/status/result.

When existing generated output is present, the `Preview` impact summary compares planned files with on-disk generated files and counts `create`, `replace`, and `unchanged` file actions. If replacing a verified generated directory would remove previously generated files that are no longer in the new plan, the preview shows a concise danger summary with sample paths; this is reporting only and does not change writer behavior. If all planned files are unchanged, the preview says that no generated file content changes were detected.

Use `up`/`down`, `k`/`j`, `pgup`/`pgdown`, `home`, and `end` to inspect planned files on the `Preview` step, `a` to cycle the planned-file action filter, `r` to refresh the plan from disk, `e` to edit solution name, description, or target framework, `g` to generate, and `q`, `esc`, or `ctrl+c` to quit. In edit mode, `tab` and `shift+tab` move between fields instead of changing dashboard steps. Target framework editing shows installed .NET SDK major-version suggestions first when `dotnet --list-sdks` is available; otherwise it falls back to a known newest-first list. You can also type a major or TFM manually, such as `6`, `7`, `net10.0`, or `net11.0`. Exit is blocked while saving settings or generating files. Service, entity, field, and value-object editing is not supported yet.

## Config convention

Configs may declare the current schema and generated .NET target framework:

```json
{
  "schemaVersion": 1,
  "generation": { "targetFramework": "net8.0" },
  "solution": { "name": "CommercePlatform" },
  "services": [
    {
      "name": "ProductService",
      "entities": [
        { "name": "Product", "fields": [{ "name": "Id", "type": "Guid" }] }
      ]
    }
  ]
}
```

Configs that omit `schemaVersion` are treated as legacy input and migrated to the current schema when loaded. Explicit schema versions must be valid integers for a supported schema; `schemaVersion: 0` and future versions are rejected. `generation.targetFramework` defaults to `net8.0` and accepts canonical `netN.0` target frameworks such as `net6.0`, `net7.0`, `net10.0`, and future major versions. TUI manual entry normalizes shorthand majors before saving, so typing `7` persists `net7.0`.

`generation.solutionFormat` is optional. Explicit values are `sln` and `slnx`; when omitted, target frameworks below `net10.0` generate `{Solution}.sln`, while `net10.0` or newer generates `{Solution}.slnx` so modern .NET consumers start from the newer solution format.

Each entity must contain exactly one identity field:

```json
{ "name": "Id", "type": "Guid" }
```

Create/update request contracts intentionally exclude `Id`; the Application layer assigns identity on create and keeps identity from being replaced on update.
Update/delete operations transport an opaque concurrency token and return conflict when the submitted version is stale. SQL Server rowversion remains an Infrastructure shadow property; Domain and Application do not know EF or byte-array rowversion storage.

Supported scalar field types: `bool`, `DateTime`, `decimal`, `double`, `Guid`, `int`, `long`, `string`.

Services can declare reusable Value Objects and then reference them from entity fields:

```json
"valueObjects": [
  {
    "name": "ProductName",
    "type": "string",
    "validations": {
      "required": true,
      "minLength": 3,
      "maxLength": 100,
      "pattern": "^[A-Za-z0-9 .'-]+$"
    }
  },
  {
    "name": "ProductPrice",
    "type": "decimal",
    "validations": { "minimum": 0, "maximum": 999999.99 }
  }
]
```

Initial rule support is intentionally small: strings support `required`, `minLength`, `maxLength`, and `pattern`; numeric types support `minimum` and `maximum`; `Guid` supports `notEmpty`; `DateTime` supports `notDefault`; `bool` has no rules yet. Unknown or inapplicable rules fail config validation before output is created. Nested/composed Value Objects are not supported yet.

Generated HTTP create/update contracts expose primitive values, not a Domain JSON shape. Application constructs Value Objects through Domain factories, aggregates field-addressable validation issues, and avoids repository mutation on validation failure. API maps those neutral validation outcomes to deterministic 400 validation ProblemDetails. Persisted Value Object data is reconstituted through Domain factories; invalid stored values raise a typed Domain signal that Infrastructure logs with service/entity/operation/value-object context without logging the sensitive value.

## SQL Server and migrations

Generated Infrastructure uses EF Core SQL Server with framework-aligned package versions pinned by the generator. The Host reads `ConnectionStrings:DefaultConnection`, including from `ConnectionStrings__DefaultConnection`, and passes configuration into Infrastructure composition. SQL command timeout/retry settings are bounded so database work fits inside the Host request timeout budget.
JWT Bearer auth reads `Authentication:Authority` and `Authentication:Audience`, including `Authentication__Authority` and `Authentication__Audience`; no signing secrets are generated.

Generated files do not contain credentials and do not run migrations, `EnsureCreated`, package installation, or database commands. Migration commands are manual user actions. A production runbook should include backup, target-environment verification, SQL review, apply, smoke test, readiness verification, repair criteria, and fix-forward planning:

```bash
TARGET_DATABASE='Products'
TARGET_SERVER='<confirmed-server-name-or-host,port>'
test -n "$TARGET_DATABASE"
test -n "$TARGET_SERVER"
dotnet ef migrations add InitialCreate --project ./out/src/ProductService/ProductService.Infrastructure --startup-project ./out/src/ProductService/ProductService.Host
dotnet ef migrations script --idempotent --project ./out/src/ProductService/ProductService.Infrastructure --startup-project ./out/src/ProductService/ProductService.Host --output ./artifacts/ProductService-migration.sql
# Review ./artifacts/ProductService-migration.sql before continuing.
sqlcmd -S "$TARGET_SERVER" -d master -Q "IF DB_ID('$TARGET_DATABASE') IS NULL THROW 50000, 'Target database not found.', 1; SELECT @@SERVERNAME AS server_name, DB_NAME(DB_ID('$TARGET_DATABASE')) AS database_name"
sqlcmd -S "$TARGET_SERVER" -d master -Q "BACKUP DATABASE [$TARGET_DATABASE] TO DISK = N'/var/opt/mssql/backups/$TARGET_DATABASE-predeploy.bak' WITH COPY_ONLY, CHECKSUM"
sqlcmd -S "$TARGET_SERVER" -d master -Q "RESTORE VERIFYONLY FROM DISK = N'/var/opt/mssql/backups/$TARGET_DATABASE-predeploy.bak' WITH CHECKSUM"
sqlcmd -S "$TARGET_SERVER" -d "$TARGET_DATABASE" -b -i ./artifacts/ProductService-migration.sql
sqlcmd -S "$TARGET_SERVER" -d "$TARGET_DATABASE" -Q "SELECT 1 AS smoke_test"
```

Run database updates only after reviewing the idempotent SQL script, taking a backup, and confirming the connection string points at the intended database. Before tightening Value Object rules, run the generated `ValueObjectPreflight.sql` artifact, quarantine/backup identified rows, repair only in an explicit transaction with business-approved replacement values, rerun preflight, then deploy. Never invent replacement business data. After deployment, `/health/ready` verifies SQL connectivity plus generated table, column type, nullability, max-length, and rowversion expectations. SQL Server reports rowversion/timestamp as `IS_NULLABLE = YES` in `INFORMATION_SCHEMA` even though the column remains provider-managed concurrency metadata; generated readiness follows that provider metadata quirk while preserving semantic concurrency handling. If readiness or reconstitution logs identify bad persisted values, repair data with a reviewed SQL script against the confirmed target database, re-run readiness, then deploy a fix-forward migration when schema/code changes are needed. Prefer expand/contract migrations for releases; after a migration reaches production, rollback is usually a new fix-forward migration rather than deleting applied schema changes.

## Safety guarantees

- Does not run `dotnet`, NuGet, shell commands, migrations, package installation, or database commands during generation.
- Keeps the core generation path UI-independent; the TUI adapter uses Bubble Tea.
- Rejects unknown, duplicate, incorrectly cased, trailing, or oversized JSON config input.
- Aggregates validation errors with actionable paths.
- Rejects invalid identifiers, duplicate names, excessive counts, Windows reserved path segment names, and invalid/missing entity `Id` fields.
- Rejects fields that collide with generated C# type names for the enclosing entity.
- Requires JWT Bearer authorization on CRUD routes; liveness remains anonymous.
- Generated API tests exercise the real JwtBearer handler with deterministic test-only signing material; no production signing secret is generated.
- Uses server-side pagination defaults and maximums instead of unbounded list queries.
- Uses Infrastructure-owned SQL Server shadow rowversion for optimistic concurrency while exposing only opaque Application tokens.
- Generated readiness checks use SQL connectivity plus generated schema verification, including string max-length checks, with generic external failure responses.
- Generated Infrastructure.Tests require `MICROGEN_TEST_SQLSERVER` and verify readiness schema drift, EF materialization, scalar mapping, opaque concurrency token refresh, update/delete, malformed-token handling, stale conflicts, corrupt persisted Value Object signals, and two-context races against SQL Server.
- Generates Architecture.Tests that inspect assembly references and enforce Domain/Application/API dependency boundaries.
- Publishes generated output through a private sibling staging directory.
- Before `--force`, verifies manifest ownership, generated file hashes, and absence of unknown files or symlinks.
- Preserves the previous generated directory and attempts rollback if publication fails.
- Uses deterministic ordering and formatting for generated files.

## Local filesystem trust boundary

`microgen` canonicalizes the nearest existing output ancestor and rejects symlinks in existing output trees before publishing. This protects normal local use from misleading paths and accidental symlink escapes. It is not a race-free security sandbox against a malicious same-user process mutating the filesystem concurrently; run it in a trusted parent directory.

## CI supply-chain note

The workflow uses pinned Go and .NET SDK versions. GitHub Actions are referenced by major tags (`actions/checkout@v4`, `actions/setup-go@v5`) because exact trusted commit SHAs were not verified locally; pinning those SHAs is a future hardening step.

## Current limitations

- Nested/composed Value Objects are a later slice; this generator remains CLI-first with a TUI adapter.
- The TUI can edit solution metadata and target framework. The Services step is currently read-only; service, entity, field, and value-object editing remains future work.
- Request idempotency is not generated; add idempotency keys or operation IDs at the API edge when required.
- Production telemetry exporters are deployment-specific; generated code provides logging/problem-details hooks but no vendor exporter.
- JSON is the only input format.
- Docker files and production telemetry exporters are future slices.

## Roadmap

1. Add nested/composed Value Objects when the simple reusable-VO model stabilizes.
2. Add Docker packaging for generated services.
3. Add TUI config editing over the existing spec/generation core.
