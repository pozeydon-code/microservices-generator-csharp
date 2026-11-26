# Feature: v0.7.0 Query Shaping

## Summary
Add explicit read-side query-shape profiles so generated C# services can return useful related data without exposing EF Core mechanics or unbounded graph expansion.

## Spec
- **S1**: Config validation rejects circular, unbounded, unknown, and multiple sibling collection profiles
- **S2**: Generated list/get endpoints remain shallow by default
- **S3**: Generated WebAPI accepts only configured shape names (e.g., `shape=detail`)
- **S4**: DTOs are acyclic and shape-specific
- **S5**: Infrastructure owns EF Core shaping (Include/ThenInclude/projection)
- **S6**: Golden tests prove deterministic output for shallow and shaped reads

## Config Structure
Query shapes are defined per service with a name and bounded depth:
```json
"queryShapes": [
  {
    "name": "summary",
    "defaultDepth": 1
  },
  {
    "name": "detail",
    "defaultDepth": 2
  }
]
```

Generator automatically:
- Builds include graph up to defaultDepth
- Rejects circular paths
- Rejects unknown relationships
- Rejects multiple sibling collection includes (prevents cartesian explosion)
- Default endpoints remain shallow (no includes)

## Tasks
- **T1**: Add QueryShape model and validation to `internal/spec/config.go` ✅
  - Added QueryShape type with name and defaultDepth fields
  - Added queryShapes field to Service struct
  - Validate shape names are unique, depth > 0 and <= 3
  - Route: inline (spec model, no high-risk)
  - Commit: `b973ce5` feat(spec): add query shape profile model and validation
  - Slice boundary: 1799 lines on feat/v0.7.0-query-shaping

- **T2**: Add query shape metadata to `internal/generator/view_builder.go` ✅
  - Added QueryShapeView type
  - Added QueryShapes and HasQueryShapes fields to ServiceView
  - Propagated query shapes from spec to templates
  - Route: inline (view builder, no high-risk)
  - Commit: `012a6a2` feat(generator): add query shape metadata to view builder
  - Slice boundary: 1864 lines on feat/v0.7.0-query-shaping

- **T3**: Update application templates for shape-aware DTOs ✅
  - Added capitalize template function for shape name formatting
  - Modified dto.tmpl to generate shape-specific DTOs ({Entity}{Shape}Dto)
  - Generate additional DTOs alongside base {Entity}Dto when query shapes configured
  - Tests passing: TestGenerateProducesShapeSpecificDtos, TestGenerateWithoutQueryShapesProducesOnlyBaseDto
  - Route: inline (templates, no high-risk)
  - Commit: `1e4f5b5` feat(t3): implement shape-specific DTOs generation
  - Slice boundary: 2066 lines on feat/v0.7.0-query-shaping
  
- **T4**: Update infrastructure templates for EF Core shaping ✅
  - Generate ListShaped_{name}Async and GetShaped_{name}Async methods per shape
  - Apply Include statements based on entity navigation properties
  - Tests verify shaped methods, ApplyShaping, and Include statements
  - Route: inline (templates, no high-risk)
  - Commit: `d9eb59d` feat(t4): add shaped read methods to repository
  - Slice boundary: 2209 lines on feat/v0.7.0-query-shaping
  
- **T5**: Update WebAPI templates for shape parameter
  - Modify `controller.tmpl` to accept `shape` query parameter
  - Validate shape against configured names
  - Route: inline (templates, no high-risk)
  - Commit: feat/webapi-templates-query-shapes
  
- **T6**: Add golden tests for query shaping
  - Create test config with query shapes
  - Generate golden output for shallow and shaped reads
  - Verify deterministic output
  - Route: inline (tests, no high-risk)
  - Commit: test/golden-query-shaping

## Log
L1: User requested to advance v0.7.0 query shaping from OpenSpec specification
L2: Created feature document and task breakdown
L3: Starting T1 - query shape model and validation
