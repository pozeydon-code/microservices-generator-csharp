package application

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/configloader"
	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

func TestGenerateGatewaySampleBuildsWithDotnet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gateway sample dotnet build in short mode")
	}
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		t.Skipf("dotnet not installed: %v", err)
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	outputDir := filepath.Join(tempDir, "output")
	t.Cleanup(func() {
		shutdownDotnetBuildServers(t, dotnet, outputDir)
	})
	cfg := gatewayBuildSampleConfig()
	if err := configloader.SaveJSON(configPath, cfg); err != nil {
		t.Fatalf("write gateway sample config: %v", err)
	}

	service, err := DefaultService()
	if err != nil {
		t.Fatalf("create application service: %v", err)
	}
	if _, err := service.Generate(GenerateRequest{ConfigPath: configPath, OutputDir: outputDir}); err != nil {
		t.Fatalf("generate gateway sample: %v", err)
	}

	assertGeneratedPathExists(t, outputDir, "Gateway/GatewayBuildSample.Gateway.csproj")
	assertGeneratedPathExists(t, outputDir, "Gateway/Program.cs")
	assertGeneratedPathExists(t, outputDir, "Gateway/appsettings.json")
	assertGeneratedPathExists(t, outputDir, "Gateway/Directory.Build.props")
	assertGeneratedPathExists(t, outputDir, "Gateway/Directory.Packages.props")

	solutionPath := filepath.Join(outputDir, "GatewayBuildSample.slnx")
	solution := readGeneratedText(t, solutionPath)
	assertContainsText(t, solution, `Path="Gateway/GatewayBuildSample.Gateway.csproj"`)

	appsettings := readGeneratedText(t, filepath.Join(outputDir, "Gateway", "appsettings.json"))
	assertContainsText(t, appsettings, `/order-service/{**catch-all}`)
	assertContainsText(t, appsettings, `/product-service/{**catch-all}`)

	runDotnetBuildEvidenceCommand(t, dotnet, outputDir, "restore", solutionPath)
	runDotnetBuildEvidenceCommand(t, dotnet, outputDir, "build", solutionPath, "--no-restore")
}

func gatewayBuildSampleConfig() spec.Config {
	return spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation: spec.GenerationOptions{
			TargetFramework: "net10.0",
			SolutionFormat:  "slnx",
			Gateway:         spec.GatewayOptions{Enabled: true},
		},
		Solution: spec.Solution{
			Name:        "GatewayBuildSample",
			Description: "Gateway build verification sample.",
		},
		Services: []spec.Service{
			{
				Name: "OrderService",
				Entities: []spec.Entity{{
					Name:   "Order",
					Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}},
				}},
			},
			{
				Name: "ProductService",
				Entities: []spec.Entity{{
					Name:   "Product",
					Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "string"}},
				}},
			},
		},
	}
}

func assertGeneratedPathExists(t *testing.T, outputDir, relativePath string) {
	t.Helper()
	path := filepath.Join(outputDir, relativePath)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected generated path %s to exist: %v", relativePath, err)
	}
}

func readGeneratedText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}
	return string(content)
}

func assertContainsText(t *testing.T, content, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected generated content to contain %q", expected)
	}
}

func runDotnetBuildEvidenceCommand(t *testing.T, dotnet, workDir string, args ...string) {
	t.Helper()
	cmd := exec.Command(dotnet, args...)
	cmd.Dir = workDir
	cmd.Env = dotnetBuildEvidenceEnv(workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dotnet %s failed: %v\nworking directory: %s\noutput:\n%s", strings.Join(args, " "), err, workDir, output)
	}
}

func shutdownDotnetBuildServers(t *testing.T, dotnet, workDir string) {
	t.Helper()
	cmd := exec.Command(dotnet, "build-server", "shutdown")
	cmd.Dir = workDir
	cmd.Env = dotnetBuildEvidenceEnv(workDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("dotnet build-server shutdown failed during cleanup: %v\noutput:\n%s", err, output)
	}
}

func dotnetBuildEvidenceEnv(workDir string) []string {
	dotnetHome := filepath.Join(workDir, ".dotnet-home")
	nugetPackages := filepath.Join(workDir, ".nuget-packages")
	return append(os.Environ(),
		"DOTNET_CLI_HOME="+dotnetHome,
		"NUGET_PACKAGES="+nugetPackages,
		"DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1",
	)
}
