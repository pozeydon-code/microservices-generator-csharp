# Query Shaping Includes Specification

## Purpose

Define bounded read-side query-shape profiles so generated C# services can expose useful related data while keeping default reads shallow, DTO graphs acyclic, EF Core mechanics internal, and output deterministic.

## Requirements

### Requirement: Query Shape Profile Validation

The system MUST validate every configured read profile before generation. A profile MUST be rejected when it references unknown relationships, creates circular traversal, exceeds bounded depth, requests unbounded graph expansion, or contains multiple sibling collection includes by default.

#### Scenario: Valid profile is accepted

- GIVEN a configured profile with known relationships and bounded depth
- WHEN configuration validation runs
- THEN the profile is accepted for generation
- AND its stable name is available to read endpoints

#### Scenario: Unsafe profile is rejected

- GIVEN a profile with a circular path or multiple sibling collection includes
- WHEN configuration validation runs
- THEN validation fails with a deterministic diagnostic
- AND generation MUST NOT continue for that profile

### Requirement: Shallow Read Defaults

The system MUST keep generated list and get reads shallow unless a configured default profile applies or a valid profile is selected explicitly.

#### Scenario: Default list read is shallow

- GIVEN an entity with configured relationships
- WHEN generating list read behavior without a selected profile
- THEN the generated response uses only the shallow read shape
- AND related entity graphs are not expanded

#### Scenario: Default profile applies only when configured

- GIVEN an endpoint with an explicit default read profile
- WHEN generating that endpoint
- THEN the generated read behavior uses that profile
- AND endpoints without defaults remain shallow

### Requirement: Constrained WebAPI Shape Selection

The system SHALL expose WebAPI shape selection only through stable configured profile names such as `shape=summary` or `shape=detail`. The system MUST NOT expose raw include paths or EF Core navigation syntax to clients.

#### Scenario: Named shape is selected

- GIVEN a request with `shape=detail` and a configured `detail` profile
- WHEN the generated controller handles the request
- THEN the detail read shape is used
- AND no raw include path is accepted

#### Scenario: Unknown shape is rejected

- GIVEN a request with an unconfigured shape name
- WHEN the generated controller validates query parameters
- THEN the request is rejected with a deterministic client error

### Requirement: Shape-Specific Acyclic DTOs

The system MUST generate read DTOs that match the selected shape and remain acyclic. Nested DTOs SHALL include only the fields and relationships required by the profile.

#### Scenario: Shaped DTO is generated

- GIVEN a profile that includes a bounded reference relationship
- WHEN generated read code emits DTOs
- THEN a shape-specific DTO includes the configured nested data
- AND the DTO does not mirror the full domain graph

#### Scenario: Cyclic DTO graph is prevented

- GIVEN relationships that could form a cycle
- WHEN a profile attempts to traverse back to an earlier entity
- THEN validation rejects the profile
- AND no cyclic DTO type is generated

### Requirement: Infrastructure-Owned EF Core Shaping

The system MUST keep EF Core shaping mechanics inside generated infrastructure. Application and WebAPI layers SHALL select profiles by name and MUST NOT assemble `Include`, `ThenInclude`, split-query, tracking, or projection mechanics directly.

#### Scenario: Infrastructure applies shape

- GIVEN a valid selected profile
- WHEN generated read behavior executes
- THEN infrastructure-owned query shaping produces the requested DTO
- AND upper layers remain independent of EF Core include syntax

### Requirement: Deterministic Golden Output

The system MUST generate deterministic source output for shallow reads and shaped reads so golden tests can verify validation, DTOs, WebAPI selection, and infrastructure shaping.

#### Scenario: Golden output is stable

- GIVEN identical input configuration with query profiles
- WHEN generation runs repeatedly
- THEN generated files are byte-stable
- AND golden tests can assert both shallow and shaped read outputs
