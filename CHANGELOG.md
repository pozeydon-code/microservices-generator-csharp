# Changelog

Release notes for `microgen`, the C#/.NET microservices generator.

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
