package configloader

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

func TestLoadJSONRejectsStrictJSONProblems(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedErr string
	}{
		{
			name:        "malformed JSON",
			content:     `{"solution":`,
			expectedErr: "parse config JSON",
		},
		{
			name:        "unknown field",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[],"unexpected":true}`,
			expectedErr: "unknown key",
		},
		{
			name:        "duplicate key",
			content:     `{"solution":{"name":"CommercePlatform","name":"Other"},"services":[]}`,
			expectedErr: "duplicate key \"name\"",
		},
		{
			name:        "incorrectly cased known key",
			content:     `{"Solution":{"name":"CommercePlatform"},"services":[]}`,
			expectedErr: "incorrectly cased key \"Solution\"",
		},
		{
			name:        "unknown generation key",
			content:     `{"schemaVersion":1,"generation":{"framework":"net9.0"},"solution":{"name":"CommercePlatform"},"services":[]}`,
			expectedErr: "unknown key \"framework\" in generation object",
		},
		{
			name:        "duplicate service key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","name":"CatalogService","entities":[]}]}`,
			expectedErr: "duplicate key \"name\" in service object",
		},
		{
			name:        "incorrectly cased service key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"Name":"ProductService","entities":[]}]}`,
			expectedErr: "incorrectly cased key \"Name\" in service object",
		},
		{
			name:        "duplicate entity key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","entities":[{"name":"Product","name":"Category","fields":[]}]}]}`,
			expectedErr: "duplicate key \"name\" in entity object",
		},
		{
			name:        "incorrectly cased entity key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","entities":[{"Name":"Product","fields":[]}]}]}`,
			expectedErr: "incorrectly cased key \"Name\" in entity object",
		},
		{
			name:        "duplicate field key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","entities":[{"name":"Product","fields":[{"name":"Id","name":"Code","type":"Guid"}]}]}]}`,
			expectedErr: "duplicate key \"name\" in field object",
		},
		{
			name:        "duplicate value object key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","valueObjects":[{"name":"ProductName","name":"Other","type":"string","validations":{}}],"entities":[]}]}`,
			expectedErr: "duplicate key \"name\" in valueObject object",
		},
		{
			name:        "unknown validation key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","valueObjects":[{"name":"ProductName","type":"string","validations":{"trim":true}}],"entities":[]}]}`,
			expectedErr: "unknown key \"trim\" in validations object",
		},
		{
			name:        "incorrectly cased validation key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","valueObjects":[{"name":"ProductName","type":"string","validations":{"Required":true}}],"entities":[]}]}`,
			expectedErr: "incorrectly cased key \"Required\" in validations object",
		},
		{
			name:        "incorrectly cased field key",
			content:     `{"solution":{"name":"CommercePlatform"},"services":[{"name":"ProductService","entities":[{"name":"Product","fields":[{"Name":"Id","type":"Guid"}]}]}]}`,
			expectedErr: "incorrectly cased key \"Name\" in field object",
		},
		{
			name:        "trailing JSON",
			content:     validConfigJSON + ` {}`,
			expectedErr: "trailing data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadJSON(writeConfig(t, tt.content))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Fatalf("expected %q, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestLoadJSONRejectsOversizedConfig(t *testing.T) {
	_, err := LoadJSON(writeConfig(t, strings.Repeat(" ", int(MaxConfigBytes)+1)))
	if err == nil {
		t.Fatal("expected oversized config error")
	}
	if !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("expected byte limit error, got %v", err)
	}
}

func TestLoadJSONAcceptsExactConfigByteLimit(t *testing.T) {
	paddingLength := int(MaxConfigBytes) - len(validConfigJSON)
	if paddingLength < 0 {
		t.Fatalf("valid config fixture is larger than max byte limit")
	}
	content := validConfigJSON + strings.Repeat(" ", paddingLength)

	if _, err := LoadJSON(writeConfig(t, content)); err != nil {
		t.Fatalf("expected exact byte limit config to load, got %v", err)
	}
}

func TestLoadJSONAcceptsSchemaVersionAndGenerationOptions(t *testing.T) {
	cfg, err := LoadJSON(writeConfig(t, `{
  "schemaVersion": 1,
  "generation": { "targetFramework": "net10.0", "solutionFormat": "slnx" },
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
}`))
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.SchemaVersion != spec.ConfigSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", spec.ConfigSchemaVersion, cfg.SchemaVersion)
	}
	if cfg.TargetFramework() != "net10.0" {
		t.Fatalf("expected net10.0 target framework, got %q", cfg.TargetFramework())
	}
	if cfg.SolutionFormat() != "slnx" {
		t.Fatalf("expected slnx solution format, got %q", cfg.SolutionFormat())
	}
}

func TestLoadJSONSchemaVersionCompatibility(t *testing.T) {
	tests := []struct {
		name                  string
		content               string
		expectedSchemaVersion int
		expectedTarget        string
		expectedErr           string
	}{
		{
			name:                  "missing schema version migrates to current schema",
			content:               validConfigJSON,
			expectedSchemaVersion: spec.ConfigSchemaVersion,
			expectedTarget:        spec.DefaultTargetFramework,
		},
		{
			name:        "explicit zero schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":0,`, 1),
			expectedErr: "schemaVersion must be 1 when present",
		},
		{
			name:                  "current schema version is valid",
			content:               strings.Replace(validConfigJSON, `{`, `{"schemaVersion":1,`, 1),
			expectedSchemaVersion: spec.ConfigSchemaVersion,
			expectedTarget:        spec.DefaultTargetFramework,
		},
		{
			name:                  "legacy config keeps selected target framework",
			content:               strings.Replace(validConfigJSON, `{`, `{"generation":{"targetFramework":"net9.0"},`, 1),
			expectedSchemaVersion: spec.ConfigSchemaVersion,
			expectedTarget:        "net9.0",
		},
		{
			name:        "future schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":2,`, 1),
			expectedErr: "unsupported schemaVersion 2; current schemaVersion is 1",
		},
		{
			name:        "non-integer schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":1.5,`, 1),
			expectedErr: "schemaVersion must be an integer",
		},
		{
			name:        "string schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":"1",`, 1),
			expectedErr: "schemaVersion must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadJSON(writeConfig(t, tt.content))
			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("expected config to load, got %v", err)
				}
				if cfg.SchemaVersion != tt.expectedSchemaVersion {
					t.Fatalf("expected schema version %d, got %d", tt.expectedSchemaVersion, cfg.SchemaVersion)
				}
				if cfg.TargetFramework() != tt.expectedTarget {
					t.Fatalf("expected target framework %q, got %q", tt.expectedTarget, cfg.TargetFramework())
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Fatalf("expected %q, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestSaveJSONLoadJSONRoundTripPreservesConfigData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microgen.json")
	cfg := spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation:    spec.GenerationOptions{TargetFramework: "net10.0", SolutionFormat: "slnx"},
		Solution:      spec.Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []spec.Service{
			{
				Name: "ProductService",
				ValueObjects: []spec.ValueObject{
					{Name: "ProductName", Type: "string"},
				},
				Entities: []spec.Entity{
					{
						Name: "Product",
						Fields: []spec.Field{
							{Name: "Id", Type: "Guid"},
							{Name: "Name", Type: "ProductName"},
						},
					},
				},
			},
		},
	}

	if err := SaveJSON(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.HasPrefix(string(content), "{\n  \"schemaVersion\": 1,") {
		t.Fatalf("expected deterministic pretty JSON with schemaVersion first, got %q", string(content))
	}
	if !strings.Contains(string(content), "\"targetFramework\": \"net10.0\"") || !strings.Contains(string(content), "\"solutionFormat\": \"slnx\"") {
		t.Fatalf("expected target framework to be saved, got %q", string(content))
	}
	loaded, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if !reflect.DeepEqual(loaded, cfg) {
		t.Fatalf("expected round trip to preserve config\nwant: %#v\n got: %#v", cfg, loaded)
	}
}

func TestSaveJSONRejectsInvalidConfigWithoutReplacingExistingFile(t *testing.T) {
	path := writeConfig(t, validConfigJSON)
	invalid := spec.Config{SchemaVersion: spec.ConfigSchemaVersion, Solution: spec.Solution{Name: "1Bad"}}

	err := SaveJSON(path, invalid)

	if err == nil || !strings.Contains(err.Error(), "solution.name must be a valid C# identifier") {
		t.Fatalf("expected validation error, got %v", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read config after failed save: %v", readErr)
	}
	if string(content) != validConfigJSON {
		t.Fatalf("expected failed save not to replace existing file, got %q", string(content))
	}
}

func TestSaveJSONPreservesExistingFileMode(t *testing.T) {
	path := writeConfig(t, validConfigJSON)
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	if err := SaveJSON(path, validConfig()); err != nil {
		t.Fatalf("save config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("expected config mode 0640, got %04o", got)
	}
}

func TestSaveJSONUsesDefaultModeForNewFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microgen.json")

	if err := SaveJSON(path, validConfig()); err != nil {
		t.Fatalf("save config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("expected new config mode 0644, got %04o", got)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func validConfig() spec.Config {
	return spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation:    spec.GenerationOptions{TargetFramework: "net9.0"},
		Solution:      spec.Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []spec.Service{{
			Name: "ProductService",
			Entities: []spec.Entity{{
				Name:   "Product",
				Fields: []spec.Field{{Name: "Id", Type: "Guid"}},
			}},
		}},
	}
}

const validConfigJSON = `{
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
