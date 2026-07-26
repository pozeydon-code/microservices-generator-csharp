package packaging

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const Repository = "pozeydon-code/generator-microservices-go"

var (
	releaseTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

//go:embed templates/homebrew/microgen.rb.tmpl templates/winget/*.tmpl templates/chocolatey/*.tmpl
var templateFiles embed.FS

type Artifact struct {
	ID     string
	OS     string
	Arch   string
	Name   string
	SHA256 string
	URL    string
}

type Release struct {
	Tag       string
	Version   string
	Artifacts []Artifact
}

type Bundle struct {
	HomebrewFormula     string
	WingetVersion       string
	WingetLocale        string
	WingetInstaller     string
	ChocolateyNuspec    string
	ChocolateyInstall   string
	ChocolateyUninstall string
}

type OutputFile struct {
	Path    string
	Content string
}

func Render(version string, checksums io.Reader) (Bundle, error) {
	release, err := ParseRelease(version, checksums)
	if err != nil {
		return Bundle{}, err
	}

	data := templateData{
		Tag:        release.Tag,
		Version:    release.Version,
		Repository: Repository,
		Artifacts:  release.Artifacts,
	}
	return Bundle{
		HomebrewFormula:     executeTemplate("templates/homebrew/microgen.rb.tmpl", data),
		WingetVersion:       executeTemplate("templates/winget/version.yaml.tmpl", data),
		WingetLocale:        executeTemplate("templates/winget/locale.en-US.yaml.tmpl", data),
		WingetInstaller:     executeTemplate("templates/winget/installer.yaml.tmpl", data),
		ChocolateyNuspec:    executeTemplate("templates/chocolatey/microgen.nuspec.tmpl", data),
		ChocolateyInstall:   executeTemplate("templates/chocolatey/chocolateyInstall.ps1.tmpl", data),
		ChocolateyUninstall: executeTemplate("templates/chocolatey/chocolateyUninstall.ps1.tmpl", data),
	}, nil
}

func ParseRelease(version string, checksums io.Reader) (Release, error) {
	if !releaseTagPattern.MatchString(version) {
		return Release{}, fmt.Errorf("release version must be an explicit semver tag such as v1.2.3, got %q", version)
	}
	checksumMap, err := parseChecksums(checksums)
	if err != nil {
		return Release{}, err
	}

	artifacts := releaseArtifacts(strings.TrimPrefix(version, "v"))
	for index := range artifacts {
		checksum, ok := checksumMap[artifacts[index].Name]
		if !ok {
			return Release{}, fmt.Errorf("checksums.txt is missing release artifact %q", artifacts[index].Name)
		}
		artifacts[index].SHA256 = checksum
		artifacts[index].URL = releaseURL(version, artifacts[index].Name)
	}
	return Release{Tag: version, Version: strings.TrimPrefix(version, "v"), Artifacts: artifacts}, nil
}

func (b Bundle) Files(release Release) []OutputFile {
	version := release.Version
	return []OutputFile{
		{Path: "homebrew/Formula/microgen.rb", Content: b.HomebrewFormula},
		{Path: fmt.Sprintf("winget/PozeydonCode.Microgen/%s/PozeydonCode.Microgen.yaml", version), Content: b.WingetVersion},
		{Path: fmt.Sprintf("winget/PozeydonCode.Microgen/%s/en-US/PozeydonCode.Microgen.locale.en-US.yaml", version), Content: b.WingetLocale},
		{Path: fmt.Sprintf("winget/PozeydonCode.Microgen/%s/PozeydonCode.Microgen.installer.yaml", version), Content: b.WingetInstaller},
		{Path: "chocolatey/microgen.nuspec", Content: b.ChocolateyNuspec},
		{Path: "chocolatey/tools/chocolateyInstall.ps1", Content: b.ChocolateyInstall},
		{Path: "chocolatey/tools/chocolateyUninstall.ps1", Content: b.ChocolateyUninstall},
	}
}

func (b Bundle) Write(outputDir string, release Release) error {
	for _, file := range b.Files(release) {
		path := filepath.Join(outputDir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create package metadata directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
			return fmt.Errorf("write package metadata %s: %w", file.Path, err)
		}
	}
	return nil
}

func Validate(bundle Bundle, release Release) error {
	if !releaseTagPattern.MatchString(release.Tag) || release.Version != strings.TrimPrefix(release.Tag, "v") {
		return fmt.Errorf("release metadata contains an invalid or non-explicit version")
	}
	for _, text := range []string{
		bundle.HomebrewFormula,
		bundle.WingetVersion,
		bundle.WingetLocale,
		bundle.WingetInstaller,
		bundle.ChocolateyNuspec,
		bundle.ChocolateyInstall,
		bundle.ChocolateyUninstall,
	} {
		if strings.Contains(strings.ToLower(text), "latest") || strings.Contains(text, "{{") {
			return fmt.Errorf("package metadata contains a dynamic latest reference or unrendered template")
		}
	}
	if err := validateReferences(bundle.HomebrewFormula, release, "darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64"); err != nil {
		return fmt.Errorf("validate Homebrew formula: %w", err)
	}
	if err := validateReferences(bundle.WingetInstaller, release, "windows-amd64", "windows-arm64"); err != nil {
		return fmt.Errorf("validate winget installer manifest: %w", err)
	}
	if err := validateReferences(bundle.ChocolateyInstall, release, "windows-amd64", "windows-arm64"); err != nil {
		return fmt.Errorf("validate Chocolatey install script: %w", err)
	}
	if !strings.Contains(bundle.WingetVersion, "ManifestType: version") ||
		!strings.Contains(bundle.WingetLocale, "ManifestType: defaultLocale") ||
		!strings.Contains(bundle.WingetInstaller, "ManifestType: installer") {
		return fmt.Errorf("winget manifests do not contain the standard version, locale, and installer manifests")
	}
	return nil
}

func parseChecksums(input io.Reader) (map[string]string, error) {
	if input == nil {
		return nil, fmt.Errorf("checksums.txt input is required")
	}
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(input)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !sha256Pattern.MatchString(fields[0]) || filepath.Base(fields[1]) != fields[1] {
			return nil, fmt.Errorf("checksums.txt line %d must contain a SHA-256 and a plain artifact filename", lineNumber)
		}
		checksum := strings.ToLower(fields[0])
		if checksum == strings.Repeat("0", 64) {
			return nil, fmt.Errorf("checksums.txt line %d contains a placeholder SHA-256", lineNumber)
		}
		if _, exists := checksums[fields[1]]; exists {
			return nil, fmt.Errorf("checksums.txt contains duplicate artifact %q", fields[1])
		}
		checksums[fields[1]] = checksum
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read checksums.txt: %w", err)
	}
	if len(checksums) == 0 {
		return nil, fmt.Errorf("checksums.txt contains no checksums")
	}
	return checksums, nil
}

func releaseArtifacts(normalizedVersion string) []Artifact {
	return []Artifact{
		{ID: "linux-amd64", OS: "linux", Arch: "amd64", Name: fmt.Sprintf("microgen_%s_linux_amd64.tar.gz", normalizedVersion)},
		{ID: "linux-arm64", OS: "linux", Arch: "arm64", Name: fmt.Sprintf("microgen_%s_linux_arm64.tar.gz", normalizedVersion)},
		{ID: "darwin-amd64", OS: "darwin", Arch: "amd64", Name: fmt.Sprintf("microgen_%s_darwin_amd64.tar.gz", normalizedVersion)},
		{ID: "darwin-arm64", OS: "darwin", Arch: "arm64", Name: fmt.Sprintf("microgen_%s_darwin_arm64.tar.gz", normalizedVersion)},
		{ID: "windows-amd64", OS: "windows", Arch: "amd64", Name: fmt.Sprintf("microgen_%s_windows_amd64.zip", normalizedVersion)},
		{ID: "windows-arm64", OS: "windows", Arch: "arm64", Name: fmt.Sprintf("microgen_%s_windows_arm64.zip", normalizedVersion)},
	}
}

func releaseURL(tag, name string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", Repository, tag, name)
}

func validateReferences(text string, release Release, ids ...string) error {
	for _, id := range ids {
		artifact := artifactByID(release.Artifacts, id)
		urlIndex := strings.Index(text, artifact.URL)
		if urlIndex < 0 {
			return fmt.Errorf("missing explicit URL for %s", id)
		}
		nextURL := strings.Index(text[urlIndex+len(artifact.URL):], "https://github.com/")
		end := len(text)
		if nextURL >= 0 {
			end = urlIndex + len(artifact.URL) + nextURL
		}
		if !strings.Contains(text[urlIndex:end], artifact.SHA256) {
			return fmt.Errorf("checksum for %s does not match the release checksums.txt entry", id)
		}
	}
	return nil
}

func artifactByID(artifacts []Artifact, id string) Artifact {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return artifact
		}
	}
	return Artifact{ID: id}
}

type templateData struct {
	Tag        string
	Version    string
	Repository string
	Artifacts  []Artifact
}

func executeTemplate(name string, data templateData) string {
	tmpl, err := template.ParseFS(templateFiles, name)
	if err != nil {
		panic(err)
	}
	var output strings.Builder
	if err := tmpl.Execute(&output, data); err != nil {
		panic(err)
	}
	return output.String()
}

func (r Release) Artifact(id string) Artifact {
	return artifactByID(r.Artifacts, id)
}
