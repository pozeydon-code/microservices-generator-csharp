package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	expectedStderr := "unknown command \"unknown\"\nUsage: microgen generate --config <path> --output <dir> [--force]\n  --force replaces only a verified microgen-owned generated directory.\n"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)

			if code != ExitOK {
				t.Fatalf("expected OK exit code, got %d stderr %q", code, stderr.String())
			}
			expectedStdout := "Usage: microgen generate --config <path> --output <dir> [--force]\n  --force replaces only a verified microgen-owned generated directory.\n"
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
