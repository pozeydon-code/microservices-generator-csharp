package configloader

import (
	"os"
	"path/filepath"
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
  "generation": { "targetFramework": "net9.0" },
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
	if cfg.TargetFramework() != "net9.0" {
		t.Fatalf("expected net9.0 target framework, got %q", cfg.TargetFramework())
	}
}

func TestLoadJSONSchemaVersionCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedErr string
	}{
		{
			name:    "missing schema version remains valid",
			content: validConfigJSON,
		},
		{
			name:        "explicit zero schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":0,`, 1),
			expectedErr: "schemaVersion must be 1",
		},
		{
			name:    "current schema version is valid",
			content: strings.Replace(validConfigJSON, `{`, `{"schemaVersion":1,`, 1),
		},
		{
			name:        "unsupported schema version is rejected",
			content:     strings.Replace(validConfigJSON, `{`, `{"schemaVersion":2,`, 1),
			expectedErr: "schemaVersion must be 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadJSON(writeConfig(t, tt.content))
			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("expected config to load, got %v", err)
				}
				if cfg.TargetFramework() != spec.DefaultTargetFramework {
					t.Fatalf("expected default target framework %q, got %q", spec.DefaultTargetFramework, cfg.TargetFramework())
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

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "microgen.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
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
