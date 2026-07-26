# CommercePlatform

Minimal generated .NET 8 microservice workspace for product management.

## Services

- `ProductService` Domain, Application, Infrastructure, WebApi, and test projects.

## Architecture

Generated services use a four-project Clean Architecture model:

| Project | Responsibility |
|---|---|
| `{Service}.Domain` | Entities with private setters and explicit create/update behavior. No framework, EF, persistence token, or rowversion dependency. |
| `{Service}.Application` | CQRS commands/handlers/validators for Create/Update/Delete, List/GetById CQRS queries/handlers, repository ports, DTOs, pagination contracts, and opaque concurrency tokens. References Domain only. |
| `{Service}.Infrastructure` | EF Core SQL Server adapter, shadow rowversion conversion, repository implementations, bounded retries/timeouts, and SQL/schema readiness adapter registration. |
| `{Service}.WebApi` | ASP.NET Core controller presentation and executable composition root for auth, HTTPS/HSTS, request timeout budget, OpenTelemetry, ProblemDetails, middleware, health endpoints, Infrastructure composition, and startup guards. References Application and Infrastructure. |

```text
Domain <- Application <- Infrastructure
Application + Infrastructure <- WebApi
```

`{Service}.Architecture.Tests` inspects generated assembly references, project references, and source text to enforce these boundaries. Value Objects live in Domain; Application constructs them and WebApi only maps validation outcomes to HTTP.

## CQRS slices

Create, List, GetById, Update, and Delete are generated vertical slices under `Application/Features/{PluralEntity}`. FluentValidation runs through closed Application MediatR behavior registrations, handlers use the Application repository port and return `ErrorOr`, and WebApi dispatches every operation through `ISender`. List preserves its pagination exception and legacy `{ "error": ... }` 400 contract; concurrency-token failures use the same compact 400 contract, while field validation uses RFC-compatible ProblemDetails.

## Value Objects

Service-level Value Objects wrap supported scalar primitives and keep business validation in Domain. Create/update request contracts use primitive JSON values. Application constructs Value Objects, aggregates field-addressable validation issues, and does not call repositories when validation fails. WebApi maps validation failures to RFC-compatible 400 validation ProblemDetails. Persisted Value Object data is reconstituted through Domain factories; invalid stored values raise a typed Domain signal that Infrastructure logs with service/entity/operation/value-object context without logging the sensitive value.

Supported rules: strings use `required`, `minLength`, `maxLength`, and `pattern`; numeric types use `minimum` and `maximum`; `Guid` uses `notEmpty`; `DateTime` uses `notDefault`; `bool` has no rules yet. Nested/composed Value Objects are intentionally out of scope for this slice.

## Safety

This output is generated as source files only. It does not run dotnet, NuGet, migrations, shell commands, or database commands.

## Dotnet scaffold plan

The generator currently renders source files directly. This deterministic plan documents the base CLI scaffold the generated workspace is moving toward; review before executing it in a separate empty directory.

Package versions are intentionally omitted from project/package commands. Generated `Directory.Packages.props` owns versions through central package management, including target-framework-specific EF Core and ASP.NET Core packages plus the compatible SqlClient policy pin. The Create slice uses verified stable `MediatR 14.2.0`, `FluentValidation 12.1.1`, `FluentValidation.DependencyInjectionExtensions 12.1.1`, and `ErrorOr 2.1.1`; the first three are consumed on `net8.0` and remain compatible with supported `net9.0` and `net10.0` targets through their `net8.0` assets where applicable.

```bash
dotnet new sln --format sln --name CommercePlatform
dotnet new classlib --framework net8.0 --name ProductService.Domain --output ./src/ProductService/ProductService.Domain
dotnet new classlib --framework net8.0 --name ProductService.Application --output ./src/ProductService/ProductService.Application
dotnet new classlib --framework net8.0 --name ProductService.Infrastructure --output ./src/ProductService/ProductService.Infrastructure
dotnet new webapi --use-controllers --framework net8.0 --name ProductService.WebApi --output ./src/ProductService/ProductService.WebApi
dotnet new xunit --framework net8.0 --name ProductService.Domain.Tests --output ./tests/ProductService/ProductService.Domain.Tests
dotnet new xunit --framework net8.0 --name ProductService.Application.Tests --output ./tests/ProductService/ProductService.Application.Tests
dotnet new xunit --framework net8.0 --name ProductService.WebApi.Tests --output ./tests/ProductService/ProductService.WebApi.Tests
dotnet new xunit --framework net8.0 --name ProductService.Architecture.Tests --output ./tests/ProductService/ProductService.Architecture.Tests
dotnet new xunit --framework net8.0 --name ProductService.Infrastructure.Tests --output ./tests/ProductService/ProductService.Infrastructure.Tests
dotnet sln ./CommercePlatform.sln add ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet sln ./CommercePlatform.sln add ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet sln ./CommercePlatform.sln add ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj
dotnet sln ./CommercePlatform.sln add ./src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj
dotnet sln ./CommercePlatform.sln add ./tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj
dotnet sln ./CommercePlatform.sln add ./tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj
dotnet sln ./CommercePlatform.sln add ./tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj
dotnet sln ./CommercePlatform.sln add ./tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj
dotnet sln ./CommercePlatform.sln add ./tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj
dotnet add ./src/ProductService/ProductService.Application/ProductService.Application.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj reference ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj
dotnet add ./tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj reference ./src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj
dotnet add ./tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
dotnet add ./tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj reference ./src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj
dotnet add ./tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj reference ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj
dotnet add ./tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj reference ./src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj
dotnet add ./tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj reference ./src/ProductService/ProductService.Application/ProductService.Application.csproj
dotnet add ./tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj reference ./src/ProductService/ProductService.Domain/ProductService.Domain.csproj
```

Versionless package plan:

| Project | Package |
|---|---|
| `src/ProductService/ProductService.Application/ProductService.Application.csproj` | `ErrorOr` |
| `src/ProductService/ProductService.Application/ProductService.Application.csproj` | `FluentValidation` |
| `src/ProductService/ProductService.Application/ProductService.Application.csproj` | `MediatR` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `Microsoft.AspNetCore.Authentication.JwtBearer` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `ErrorOr` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `FluentValidation.DependencyInjectionExtensions` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `MediatR` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `OpenTelemetry.Exporter.OpenTelemetryProtocol` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `OpenTelemetry.Extensions.Hosting` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `OpenTelemetry.Instrumentation.AspNetCore` |
| `src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj` | `OpenTelemetry.Instrumentation.Http` |
| `src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj` | `Microsoft.EntityFrameworkCore.Design` |
| `src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj` | `Microsoft.EntityFrameworkCore.SqlServer` |
| `src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj` | `Microsoft.Data.SqlClient` |
| `tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj` | `Microsoft.AspNetCore.Mvc.Testing` |
| `tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj` | `System.IdentityModel.Tokens.Jwt` |
| `tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj` | `Microsoft.NET.Test.Sdk` |
| `tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj` | `xunit` |
| `tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj` | `xunit.runner.visualstudio` |
| `tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj` | `Microsoft.NET.Test.Sdk` |
| `tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj` | `xunit` |
| `tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj` | `xunit.runner.visualstudio` |
| `tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj` | `Microsoft.NET.Test.Sdk` |
| `tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj` | `xunit` |
| `tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj` | `xunit.runner.visualstudio` |
| `tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj` | `Microsoft.NET.Test.Sdk` |
| `tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj` | `xunit` |
| `tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj` | `xunit.runner.visualstudio` |
| `tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj` | `Microsoft.NET.Test.Sdk` |
| `tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj` | `xunit` |
| `tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj` | `xunit.runner.visualstudio` |
| `tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj` | `Microsoft.EntityFrameworkCore.SqlServer` |
| `tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj` | `Microsoft.Data.SqlClient` |

## Configuration

- Set `ConnectionStrings__DefaultConnection` to a SQL Server connection string. WebApi configuration passes it to Infrastructure; production strings should validate certificates, and `TrustServerCertificate=True` is a local-development exception only.
- Set `Authentication__Authority` and `Authentication__Audience` for JWT Bearer authentication. No signing secrets are generated.
- CRUD endpoints require authorization. `/health/live` is anonymous and process-only. `/health/ready` checks SQL connectivity and verifies generated table, column type, nullability, max-length, and rowversion expectations. External readiness failures use a generic response; detailed failure reasons are written to logs only.
- WebApi projects redirect HTTP to HTTPS and enable HSTS outside Development/Testing. Production traffic must use HTTPS all the way to the WebApi, including private backend TLS from a trusted reverse proxy, unless operators explicitly implement and audit trusted forwarded-header configuration.
- Configure OpenTelemetry OTLP export with `OTEL_EXPORTER_OTLP_ENDPOINT` and standard `OTEL_EXPORTER_OTLP_*` variables, or explicitly set `OTEL_SDK_DISABLED=true` to opt out. Generated code does not provision vendor alerts; start with deployment-specific HTTP 5xx and p95 latency alerts.

## Build and test

Generated workspaces keep quality defaults in `Directory.Build.props`: nullable reference types, implicit usings, SDK recommended analyzers, code-style enforcement during build, and warnings-as-errors. The generator runtime harness restores, builds, and tests the generated `net10.0` `.slnx` workspace with these settings so clean output is validated continuously.

```bash
dotnet restore ./CommercePlatform.sln
dotnet build ./CommercePlatform.sln --nologo -warnaserror
MICROGEN_TEST_SQLSERVER='Server=localhost,1433;User Id=sa;Password=<password>;Encrypt=True;TrustServerCertificate=True' \
  dotnet test ./CommercePlatform.sln --nologo -warnaserror --blame-hang --blame-hang-timeout 90s --logger "trx;LogFileName=generated-tests.trx"
```

## Manual migrations

Create and apply EF migrations manually after review, backup, staging validation, and deployment approval. The generator never runs migrations or opens a database connection.

Run the same sequence for each generated service:

### ProductService

1. Build and test the generated solution.
2. Resolve and confirm one explicit target server/database from runtime configuration.
3. Create or update the reviewed migration source.
4. Generate an idempotent SQL artifact and review it before deployment.
5. Validate against staging before production.
6. Back up the target database and verify the backup.
7. Apply the reviewed SQL artifact in the approved deployment window.
8. Smoke test the API and readiness endpoint after the application version is deployed.
9. If readiness or reconstitution logs identify bad persisted values, repair data only with a reviewed SQL script against the confirmed target database, re-run readiness, and preserve the repair script with the deployment artifact.

```bash
mkdir -p ./artifacts
dotnet test ./CommercePlatform.sln --nologo -warnaserror --logger "trx;LogFileName=ProductService-tests.trx"
TARGET_DATABASE='<confirmed-database-name>'
TARGET_SERVER='<confirmed-server-name-or-host,port>'
test -n "$TARGET_DATABASE"
test -n "$TARGET_SERVER"
dotnet ef migrations add InitialCreate --project ./src/ProductService/ProductService.Infrastructure --startup-project ./src/ProductService/ProductService.WebApi
dotnet ef migrations script --idempotent --project ./src/ProductService/ProductService.Infrastructure --startup-project ./src/ProductService/ProductService.WebApi --output ./artifacts/ProductService-migration.sql
# Review ./artifacts/ProductService-migration.sql before continuing.
sqlcmd -S "$TARGET_SERVER" -d master -Q "IF DB_ID('$TARGET_DATABASE') IS NULL THROW 50000, 'Target database not found.', 1; SELECT @@SERVERNAME AS server_name, DB_NAME(DB_ID('$TARGET_DATABASE')) AS database_name"
sqlcmd -S "$TARGET_SERVER" -d master -Q "BACKUP DATABASE [$TARGET_DATABASE] TO DISK = N'/var/opt/mssql/backups/$TARGET_DATABASE-predeploy.bak' WITH COPY_ONLY, CHECKSUM"
sqlcmd -S "$TARGET_SERVER" -d master -Q "RESTORE VERIFYONLY FROM DISK = N'/var/opt/mssql/backups/$TARGET_DATABASE-predeploy.bak' WITH CHECKSUM"
sqlcmd -S "$TARGET_SERVER" -d "$TARGET_DATABASE" -b -i ./artifacts/ProductService-migration.sql
sqlcmd -S "$TARGET_SERVER" -d "$TARGET_DATABASE" -Q "SELECT 1 AS smoke_test"
```

Before tightening Value Object rules, run `./src/ProductService/ProductService.Infrastructure/Persistence/ValueObjectPreflight.sql` against the confirmed target database. The script covers SQL-safe rules only: null/required, min/max length, numeric min/max, Guid empty, and DateTime default. Regex rules require application/manual audit because regex semantics are not safely portable to SQL. Quarantine and back up identified rows, repair only in an explicit transaction with business-approved values, rerun preflight, and only then deploy the migration. Never invent replacement business data.

Rollback limitations: database rollback is not automatically reversible after schema changes. If rollback-to-target is required, create and review a target-specific SQL script from a verified backup/restore plan; otherwise prefer fix-forward.

Fix-forward flow:

```bash
dotnet ef migrations add FixForwardDescription --project ./src/ProductService/ProductService.Infrastructure --startup-project ./src/ProductService/ProductService.WebApi
dotnet ef migrations script --idempotent --project ./src/ProductService/ProductService.Infrastructure --startup-project ./src/ProductService/ProductService.WebApi --output ./artifacts/ProductService-fix-forward.sql
# Review and validate ./artifacts/ProductService-fix-forward.sql in staging.
sqlcmd -S "$TARGET_SERVER" -d "$TARGET_DATABASE" -b -i ./artifacts/ProductService-fix-forward.sql
```

Prefer expand/contract changes for releases. Never auto-run generated migration commands from the generator or CI unless your delivery process explicitly owns that database deployment step.
