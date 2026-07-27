# Package Metadata Handoff

This directory documents the package-manager metadata foundation for `microgen`.
It is **not** a publication repository and does not claim ownership of any
Homebrew tap, winget publisher identity, or Chocolatey account.

## Render From A GitHub Release

The only binary source is the exact GitHub Release identified by an explicit
tag. Download only that release's checksum file, then render and validate the
metadata:

```bash
release_tag=v1.2.3
mkdir -p /tmp/microgen-release
gh release download "$release_tag" \
  --repo pozeydon-code/microservices-generator-csharp \
  --pattern checksums.txt \
  --dir /tmp/microgen-release
go run ./cmd/package-manifests \
  --version "$release_tag" \
  --checksums /tmp/microgen-release/checksums.txt \
  --output ./package-manifests
```

The command fails closed unless the tag is an explicit `vMAJOR.MINOR.PATCH`
semver tag and `checksums.txt` contains valid SHA-256 entries for all six
archives produced by `.goreleaser.yaml`. It never downloads or rebuilds a
binary. It writes deterministic, reviewable output for Homebrew, winget, and
Chocolatey, and rejects dynamic `latest` references, unrendered templates, and
checksum associations that do not match the release checksum file.

The manual handoff workflow, `.github/workflows/package-manifests.yml`, runs
this same renderer for a `workflow_dispatch` release tag and uploads the
validated metadata as a workflow artifact. It does not publish to a package
repository.

## Metadata And Ownership

- Homebrew output is a formula for a future owned tap. It selects the exact
  macOS/Linux amd64 or arm64 archive and its release SHA-256.
- winget output uses the provisional `PozeydonCode.Microgen` identity. It has
  standard version, `en-US` locale, and installer manifests, with explicit
  GitHub Release URLs/checksums, x64/arm64 ZIP installers, the `microgen`
  portable command alias, and `UpgradeBehavior: install`.
- Chocolatey output uses the provisional `microgen` package identity. Its
  install hook selects the exact Windows x64/arm64 archive, verifies the
  checksum before extraction, and registers the normal `microgen` shim. The
  uninstall hook removes that shim; Chocolatey owns normal package-directory
  cleanup during uninstall and upgrade.

Publisher/package identity must be confirmed against repository and account
ownership before submitting any metadata. No credentials, generated-project
installer behavior, or package publication command belongs in this repository.

## Independent Verification And Operations

Before publication, a release operator must independently verify the downloaded
binary archives, not just their filenames:

```bash
sha256sum -c checksums.txt
```

On Windows, compare each archive with `Get-FileHash -Algorithm SHA256` and the
matching `checksums.txt` entry. Then perform clean-machine install, upgrade,
command, and uninstall checks for each supported package-manager path:

- Homebrew: `brew install microgen`, `brew upgrade microgen`,
  `brew uninstall microgen`.
- winget: `winget install --id PozeydonCode.Microgen`,
  `winget upgrade --id PozeydonCode.Microgen`,
  `winget uninstall --id PozeydonCode.Microgen`.
- Chocolatey: `choco install microgen`, `choco upgrade microgen`,
  `choco uninstall microgen`.

Those commands are verification handoff examples only. They are deliberately
not run by the renderer or CI workflow. Production distribution remains gated
on clean-machine evidence, confirmed ecosystem ownership, and explicit
publication configuration/secrets.
