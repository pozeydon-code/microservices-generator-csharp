package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pozeydon-code/generator-microservices-go/internal/application"
	"github.com/pozeydon-code/generator-microservices-go/internal/tui"
)

func TestRunGenerateSucceeds(t *testing.T) {
	configPath := writeTempConfig(t, validJSONConfig)
	outputDir := filepath.Join(t.TempDir(), "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"generate", "--config", configPath, "--output", outputDir}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected success, got code %d stderr %q", code, stderr.String())
	}
	expectedStdout := "Generated 44 files in " + outputDir + "\n"
	if stdout.String() != expectedStdout {
		t.Fatalf("expected stdout %q, got %q", expectedStdout, stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outputDir, "src", "ProductService", "ProductService.Domain", "Features", "Products", "Product.cs")); err != nil {
		t.Fatalf("expected entity file: %v", err)
	}
}

func TestRunGenerateReturnsNonZeroForInvalidConfig(t *testing.T) {
	configPath := writeTempConfig(t, `{"solution":{"name":"1Bad"},"services":[]}`)
	parent := t.TempDir()
	outputDir := filepath.Join(parent, "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"generate", "--config", configPath, "--output", outputDir}, &stdout, &stderr)

	if code != ExitError {
		t.Fatalf("expected domain error exit code %d, got %d", ExitError, code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	expectedStderr := "invalid config:\n- solution.name must be a valid C# identifier\n- services must contain at least 1 item\n"
	if stderr.String() != expectedStderr {
		t.Fatalf("expected stderr %q, got %q", expectedStderr, stderr.String())
	}
	for _, path := range []string{outputDir, filepath.Join(parent, ".generated.microgen.publish.lock")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be absent after invalid config, stat err=%v", path, err)
		}
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("read output parent: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "microgen-staging") {
			t.Fatalf("expected no staging path after invalid config, found %s", entry.Name())
		}
	}
}

func TestRunGenerateReturnsUsageErrorForMissingFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"generate", "--config", "config.json"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.String() != "missing required --output directory\n" {
		t.Fatalf("expected missing output message, got %q", stderr.String())
	}
}

func TestRunTUIReturnsUsageErrorForMissingFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"tui", "--config", "config.json"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.String() != "missing required --output directory\n" {
		t.Fatalf("expected missing output message, got %q", stderr.String())
	}
}

func TestRunTUIReturnsNonZeroForInvalidConfigBeforeStartingProgram(t *testing.T) {
	configPath := writeTempConfig(t, `{"solution":{"name":"1Bad"},"services":[]}`)
	outputDir := filepath.Join(t.TempDir(), "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	programStarted := false
	originalRunTUIProgram := runTUIProgram
	runTUIProgram = func(plan application.GenerationPlan, request application.GenerateRequest, planFunc tui.PlanFunc, generate tui.GenerateFunc, update tui.UpdateSettingsFunc, targetFrameworkSuggestions []string) error {
		programStarted = true
		return nil
	}
	t.Cleanup(func() { runTUIProgram = originalRunTUIProgram })

	code := Run([]string{"tui", "--config", configPath, "--output", outputDir}, &stdout, &stderr)

	if code != ExitError {
		t.Fatalf("expected domain error exit code %d, got %d", ExitError, code)
	}
	if programStarted {
		t.Fatal("expected invalid config to stop before starting TUI program")
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	expectedStderr := "invalid config:\n- solution.name must be a valid C# identifier\n- services must contain at least 1 item\n"
	if stderr.String() != expectedStderr {
		t.Fatalf("expected stderr %q, got %q", expectedStderr, stderr.String())
	}
}

func TestRunTUISucceedsWithRunnerSeam(t *testing.T) {
	configPath := writeTempConfig(t, validJSONConfig)
	outputDir := filepath.Join(t.TempDir(), "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var capturedPlan application.GenerationPlan
	var capturedRequest application.GenerateRequest
	var capturedSuggestions []string
	refreshCalled := false
	generateCalled := false
	updateCalled := false
	programStarted := false
	originalRunTUIProgram := runTUIProgram
	runTUIProgram = func(plan application.GenerationPlan, request application.GenerateRequest, planFunc tui.PlanFunc, generate tui.GenerateFunc, update tui.UpdateSettingsFunc, targetFrameworkSuggestions []string) error {
		programStarted = true
		capturedPlan = plan
		capturedRequest = request
		capturedSuggestions = append([]string(nil), targetFrameworkSuggestions...)
		refreshedPlan, err := planFunc(request)
		if err != nil {
			t.Fatalf("expected refresh action to succeed: %v", err)
		}
		if refreshedPlan.FileCount != 44 || refreshedPlan.OutputDir != outputDir {
			t.Fatalf("expected refreshed plan for output dir %s, got %#v", outputDir, refreshedPlan)
		}
		refreshCalled = true
		result, err := generate(request)
		if err != nil {
			t.Fatalf("expected generation action to succeed: %v", err)
		}
		if result.Plan.FileCount != 44 || result.OutputDir != outputDir {
			t.Fatalf("expected generation result for output dir %s, got %#v", outputDir, result)
		}
		generateCalled = true
		updateResult, err := update(request, application.SolutionSettings{SolutionName: "CatalogPlatform", SolutionDescription: "Catalog management.", TargetFramework: "net9.0"})
		if err != nil {
			t.Fatalf("expected settings update action to succeed: %v", err)
		}
		if !updateResult.Saved || updateResult.PlanError != nil || updateResult.Plan.Config.SolutionName != "CatalogPlatform" || updateResult.Plan.Config.TargetFramework != "net9.0" || updateResult.Plan.FileCount != 44 {
			t.Fatalf("expected updated plan from settings callback, got %#v", updateResult)
		}
		updateCalled = true
		return nil
	}
	t.Cleanup(func() { runTUIProgram = originalRunTUIProgram })

	code := Run([]string{"tui", "--config", configPath, "--output", outputDir, "--force"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected success, got code %d stderr %q", code, stderr.String())
	}
	if !programStarted {
		t.Fatal("expected TUI program to start")
	}
	if capturedPlan.OutputDir != outputDir || capturedPlan.FileCount != 44 || capturedPlan.OutputAction != "create" || capturedPlan.ForceUsed {
		t.Fatalf("expected planned generation to be passed to TUI, got %#v", capturedPlan)
	}
	if capturedRequest.ConfigPath != configPath || capturedRequest.OutputDir != outputDir || !capturedRequest.Force {
		t.Fatalf("expected generation request to be passed to TUI, got %#v", capturedRequest)
	}
	if len(capturedSuggestions) == 0 {
		t.Fatal("expected target framework suggestions to be passed to TUI")
	}
	if !refreshCalled || !generateCalled || !updateCalled {
		t.Fatalf("expected refresh, generation, and settings actions to be passed to TUI, refresh=%t generate=%t update=%t", refreshCalled, generateCalled, updateCalled)
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("expected no CLI output around TUI, got stdout %q stderr %q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outputDir, "src", "ProductService", "ProductService.Domain", "Features", "Products", "Product.cs")); err != nil {
		t.Fatalf("expected TUI generation action to write output: %v", err)
	}
}

func TestRunReturnsUsageErrorForUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"unknown"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	expectedStderr := "unknown command \"unknown\"\nUsage: microgen generate --config <path> --output <dir> [--force]\n       microgen tui --config <path> --output <dir> [--force]\n  --force replaces only a verified microgen-owned generated directory.\n"
	if stderr.String() != expectedStderr {
		t.Fatalf("expected stderr %q, got %q", expectedStderr, stderr.String())
	}
}

func TestRunGenerateHelpExitsOK(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "root help", args: []string{"--help"}},
		{name: "generate help", args: []string{"generate", "--help"}},
		{name: "tui help", args: []string{"tui", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)

			if code != ExitOK {
				t.Fatalf("expected OK exit code, got %d stderr %q", code, stderr.String())
			}
			expectedStdout := "Usage: microgen generate --config <path> --output <dir> [--force]\n       microgen tui --config <path> --output <dir> [--force]\n  --force replaces only a verified microgen-owned generated directory.\n"
			if stdout.String() != expectedStdout {
				t.Fatalf("expected stdout %q, got %q", expectedStdout, stdout.String())
			}
			if stderr.String() != "" {
				t.Fatalf("expected empty stderr for help, got %q", stderr.String())
			}
		})
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

const validJSONConfig = `{
  "solution": { "name": "CommercePlatform", "description": "Product management." },
  "services": [
    {
      "name": "ProductService",
      "entities": [
        {
          "name": "Product",
          "fields": [
            { "name": "Id", "type": "Guid" },
            { "name": "Name", "type": "string" }
          ]
        }
      ]
    }
  ]
}`
