# Release Contract

`microgen` releases are published from version tags such as `v1.2.3`. GitHub Releases are the public binary source for this slice. Homebrew, winget, and Chocolatey publication are intentionally not configured yet.

## Local Validation

Install a pinned GoReleaser v2 release and Syft before validating release configuration locally. These commands build only a local snapshot and do not publish a release:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

The snapshot must contain archives for Linux, macOS, and Windows, a `checksums.txt` file, and SPDX JSON SBOMs. Archive names use this stable form:

```text
microgen_<version>_<os>_<arch>.tar.gz
microgen_<version>_windows_<arch>.zip
```

Release builds disable CGO, use `-trimpath`, set archive file timestamps from the commit timestamp, and inject version, commit, and commit date metadata. The generated project output remains unchanged and never includes release or installer scripts.

## GitHub Actions Publication Order

The tag workflow builds the release into `dist/` with GoReleaser v2 using `release --skip=publish --clean`. This keeps the tag-derived version and reproducible archives, checksums, and SBOMs local; GoReleaser does not create or modify a public release at this stage.

The workflow then attests the generated `dist/*` files. Only after that step succeeds does it verify the pushed tag and publish the archives, `checksums.txt`, and SPDX JSON SBOMs. If the GitHub Release does not exist, GitHub CLI creates it with verified-tag and generated-notes checks. If it already exists, GitHub CLI uploads the same asset set with `--clobber`.

If a release run fails before publication, fix the underlying build or attestation problem and rerun the tag workflow. The rerun rebuilds and re-attests the exact local `dist/*` set before changing the public release. If an earlier run already created the release, the existing-release path replaces matching assets safely; do not manually publish un-attested files.

Verify downloaded artifacts independently rather than trusting an archive filename:

```bash
sha256sum -c checksums.txt
```

On Windows, compare `Get-FileHash` SHA256 values with the corresponding `checksums.txt` entries. Inspect each `.spdx.json` document as SPDX JSON and verify the GitHub artifact attestation for the downloaded artifact through GitHub's attestation verification tooling or UI.

## Repository Configuration

The release workflow requires:

- A `v*` tag pushed from the intended commit, with complete history available to GoReleaser.
- GitHub Actions permissions for `contents: write`, `id-token: write`, and `attestations: write`; the workflow scopes these permissions to the release job.
- The repository's normal `GITHUB_TOKEN`; no long-lived signing secret is required.
- GitHub Actions access to the pinned GoReleaser and Syft actions.

The workflow is a release foundation, not a claim of production distribution readiness. Repository administrators still need to review branch/tag governance, action trust and pinning policy, GitHub artifact-attestation availability, and an independent clean-machine verification before calling a release production-ready.
