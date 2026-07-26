package packaging

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUsesExactReleaseAssetsAcrossManagers(t *testing.T) {
	checksums := fixtureChecksums("v1.2.3")
	release, err := ParseRelease("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(bundle, release); err != nil {
		t.Fatal(err)
	}

	for _, artifact := range release.Artifacts {
		if artifact.URL == "" || !strings.Contains(bundle.HomebrewFormula+bundle.WingetInstaller+bundle.ChocolateyInstall, artifact.URL) {
			t.Errorf("rendered metadata does not reference %s", artifact.Name)
		}
	}
	for _, expected := range []string{
		"Hardware::CPU.arm?",
		"Architecture: x64",
		"Architecture: arm64",
		"PortableCommandAlias: microgen",
		"UpgradeBehavior: install",
		"Get-ChocolateyUnzip",
		"Install-BinFile -Name 'microgen'",
	} {
		if !strings.Contains(bundle.HomebrewFormula+bundle.WingetInstaller+bundle.ChocolateyInstall, expected) {
			t.Errorf("rendered metadata is missing %q", expected)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	checksums := fixtureChecksums("v1.2.3")
	first, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("rendering the same release twice produced different metadata")
	}
}

func TestParseReleaseAcceptsGoReleaserArchiveNames(t *testing.T) {
	checksums := strings.Join([]string{
		strings.Repeat("a", 64) + "  microgen_1.2.3_linux_amd64.tar.gz",
		strings.Repeat("b", 64) + "  microgen_1.2.3_linux_arm64.tar.gz",
		strings.Repeat("c", 64) + "  microgen_1.2.3_darwin_amd64.tar.gz",
		strings.Repeat("d", 64) + "  microgen_1.2.3_darwin_arm64.tar.gz",
		strings.Repeat("e", 64) + "  microgen_1.2.3_windows_amd64.zip",
		strings.Repeat("f", 64) + "  microgen_1.2.3_windows_arm64.zip",
	}, "\n")

	release, err := ParseRelease("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	if release.Tag != "v1.2.3" || release.Version != "1.2.3" {
		t.Fatalf("unexpected release identity: %+v", release)
	}
	for _, expected := range []struct {
		id   string
		name string
	}{
		{id: "linux-amd64", name: "microgen_1.2.3_linux_amd64.tar.gz"},
		{id: "linux-arm64", name: "microgen_1.2.3_linux_arm64.tar.gz"},
		{id: "darwin-amd64", name: "microgen_1.2.3_darwin_amd64.tar.gz"},
		{id: "darwin-arm64", name: "microgen_1.2.3_darwin_arm64.tar.gz"},
		{id: "windows-amd64", name: "microgen_1.2.3_windows_amd64.zip"},
		{id: "windows-arm64", name: "microgen_1.2.3_windows_arm64.zip"},
	} {
		artifact := release.Artifact(expected.id)
		if artifact.Name != expected.name {
			t.Errorf("artifact %s = %q, want %q", expected.id, artifact.Name, expected.name)
		}
		wantURL := "https://github.com/pozeydon-code/generator-microservices-go/releases/download/v1.2.3/" + expected.name
		if artifact.URL != wantURL {
			t.Errorf("artifact %s URL = %q, want %q", expected.id, artifact.URL, wantURL)
		}
	}
}

func TestParseReleaseRejectsInvalidVersions(t *testing.T) {
	for _, version := range []string{"", "latest", "1.2.3", "v1.2", "v01.2.3", "dev"} {
		t.Run(version, func(t *testing.T) {
			if _, err := ParseRelease(version, strings.NewReader(fixtureChecksums("v1.2.3"))); err == nil {
				t.Fatalf("ParseRelease(%q) unexpectedly succeeded", version)
			}
		})
	}
}

func TestParseReleaseRejectsMissingMismatchedAndPlaceholderChecksums(t *testing.T) {
	valid := fixtureChecksums("v1.2.3")
	missing := strings.Replace(valid, "microgen_1.2.3_windows_arm64.zip", "microgen_1.2.4_windows_arm64.zip", 1)
	placeholder := strings.Replace(valid, strings.Repeat("f", 64), strings.Repeat("0", 64), 1)
	malformed := strings.Replace(valid, strings.Repeat("a", 64), "not-a-sha", 1)

	for name, input := range map[string]string{
		"missing artifact from mismatched version": missing,
		"placeholder checksum":                     placeholder,
		"malformed checksum":                       malformed,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRelease("v1.2.3", strings.NewReader(input)); err == nil {
				t.Fatal("ParseRelease unexpectedly accepted invalid checksums")
			}
		})
	}
}

func TestValidateRejectsChecksumAssociationChanges(t *testing.T) {
	checksums := fixtureChecksums("v1.2.3")
	release, err := ParseRelease("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	bundle.WingetInstaller = strings.Replace(bundle.WingetInstaller, release.Artifact("windows-amd64").SHA256, strings.Repeat("b", 64), 1)
	if err := Validate(bundle, release); err == nil {
		t.Fatal("Validate unexpectedly accepted a mismatched installer checksum")
	}
}

func TestBundleFilesAreVersionedAndStable(t *testing.T) {
	checksums := fixtureChecksums("v1.2.3")
	release, err := ParseRelease("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	files := bundle.Files(release)
	if len(files) != 7 {
		t.Fatalf("got %d output files, want 7", len(files))
	}
	for _, file := range files {
		if strings.Contains(file.Path, "latest") || strings.Contains(file.Content, "{{") {
			t.Errorf("file contains dynamic or unrendered metadata: %+v", file)
		}
	}
	if !bytes.Contains([]byte(files[1].Path), []byte("1.2.3")) {
		t.Fatalf("winget output path is not versioned: %s", files[1].Path)
	}
}

func TestBundleWritesSevenReviewableFiles(t *testing.T) {
	checksums := fixtureChecksums("v1.2.3")
	release, err := ParseRelease("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Render("v1.2.3", strings.NewReader(checksums))
	if err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	if err := bundle.Write(outputDir, release); err != nil {
		t.Fatal(err)
	}
	for _, file := range bundle.Files(release) {
		content, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Errorf("read %s: %v", file.Path, err)
			continue
		}
		if string(content) != file.Content {
			t.Errorf("written %s differs from rendered content", file.Path)
		}
	}
	var document struct {
		Metadata struct {
			ID      string `xml:"id"`
			Version string `xml:"version"`
		} `xml:"metadata"`
	}
	if err := xml.Unmarshal([]byte(bundle.ChocolateyNuspec), &document); err != nil {
		t.Fatalf("rendered Chocolatey nuspec is not XML: %v", err)
	}
	if document.Metadata.ID != "microgen" || document.Metadata.Version != "1.2.3" {
		t.Fatalf("unexpected Chocolatey metadata: %+v", document.Metadata)
	}
}

func fixtureChecksums(version string) string {
	artifacts := releaseArtifacts(strings.TrimPrefix(version, "v"))
	var builder strings.Builder
	for index, artifact := range artifacts {
		builder.WriteString(strings.Repeat(string(rune('a'+index)), 64))
		builder.WriteString("  ")
		builder.WriteString(artifact.Name)
		builder.WriteByte('\n')
	}
	return builder.String()
}
