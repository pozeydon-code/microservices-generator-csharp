# Feature: v0.8.0 Simplification & Flexibility

## Summary
Implement 5 major improvements to reduce over-engineering and add flexibility to the generated code.

## Spec
- **S1**: Add generation mode (simple/enterprise) to control code generation complexity
- **S2**: Unify error handling system using only ErrorOr (remove custom DomainResult<T>)
- **S3**: Simplify logging by removing SQL diagnostic queries for Value Objects
- **S4**: Consolidate test projects from 5 to 2-3 projects
- **S5**: Implement incremental generation (only regenerate what changed)

## Config Structure (S1)
```json
{
  "generation": {
    "mode": "simple" | "enterprise"
  }
}
```

**Simple mode**:
- Direct handlers (no CQRS)
- No EntitySnapshot pattern
- Direct repository access
- Simpler error handling

**Enterprise mode** (current behavior):
- CQRS with MediatR
- EntitySnapshot pattern
- Value Objects with validation
- Full audit trail

## Tasks

### T1: Add generation mode to spec [S1] ✅
- Add `Mode` field to GenerationOptions in spec/config.go
- Validate mode is "simple" or "enterprise" (default: "enterprise")
- Add tests for validation
- Route: inline (spec model)
- Commit: `0068c18` feat(spec): add generation mode (simple/enterprise) to config

### T2: Create simple mode templates [S1] ✅
- Create alternative templates for simple mode:
  - Direct handlers (no MediatR)
  - Simple DTOs (no EntitySnapshot)
  - Direct repository methods
- Add template selection logic in generator
- Add tests for both modes
- Route: inline (templates)
- Commit: `c65d1d2` feat(t2): create simple mode templates (no CQRS, no EntitySnapshot)

### T3: Unify error handling with ErrorOr [S2] ✅
- Remove DomainResult<T> from Domain/Primitives
- Update Value Object templates to use ErrorOr
- Update Command/Query handlers to use ErrorOr consistently
- Remove DomainReconstitutionException (use ErrorOr)
- Update tests
- Route: inline (templates + domain)
- Commit: `99e55a7` feat(t3): unify error handling with ErrorOr only

### T4: Simplify repository logging [S3] ✅
- Remove SQL diagnostic queries from repository templates
- Simplify LoggerMessage to standard structured logging
- Remove FindReconstitutionDiagnosticsAsync method
- Keep basic error logging
- Update tests
- Route: inline (templates)
- Commit: `99e55a7` (completed as part of T3 - simplified repository)

### T5: Consolidate test projects [S4] ✅
- Consolidated 5 test projects into 3:
  * UnitTests (Domain + Application tests)
  * IntegrationTests (Infrastructure + WebApi tests)
  * ArchitectureTests (Architecture tests)
- src/ keeps full service prefixes (ProductService.Domain, etc.) - .NET convention
- tests/ uses simple names (UnitTests, IntegrationTests, ArchitectureTests)
- Updated view_builder.go, generator.go, scaffold_plan.go
- Updated all test templates with new namespaces
- Updated generator_test.go and golden files
- Route: inline (templates + solution structure)
- Commit: `4f5e8a3` feat(t5): consolidate test projects to simpler structure (5 → 3 projects)

### T6: Implement incremental generation [S5]
- Add manifest tracking to detect changes
- Compare config hash with previous generation
- Only regenerate files affected by changes
- Add `--force` flag to force full regeneration
- Add tests for incremental behavior
- Route: inline (generator logic)
- Commit: `feat/incremental-generation`

### T7: Update documentation and examples [S1-S5]
- Update README with new generation modes
- Add examples for simple mode
- Document incremental generation
- Update CHANGELOG
- Route: inline (docs)
- Commit: `docs/update-documentation`

## Log
L1: User requested implementation of 5 simplification recommendations
L2: Created feature document with 7 tasks
L3: Starting T1 - add generation mode to spec
