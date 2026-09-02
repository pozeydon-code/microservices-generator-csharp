package release_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/configloader"
)

func TestV041HardeningDocsDescribeDefensiveScope(t *testing.T) {
	repo := repositoryRoot(t)
	tests := []struct {
		name string
		path string
	}{
		{name: "readme", path: "README.md"},
		{name: "changelog", path: "CHANGELOG.md"},
		{name: "release", path: "RELEASE.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readFile(t, filepath.Join(repo, tt.path))
			assertContainsText(t, content, "v0.4.1")
			assertContainsText(t, strings.ToLower(content), "hardening")
			assertDoesNotClaimUnsupportedRelationshipRelease(t, content)
		})
	}
}

func TestRelationshipServiceFixtureUsesOnlySupportedOneToManyRelationships(t *testing.T) {
	fixture := filepath.Join(repositoryRoot(t), "examples", "relationship-service.json")
	cfg, err := configloader.LoadJSON(fixture)
	if err != nil {
		t.Fatalf("expected relationship fixture to load: %v", err)
	}
	if len(cfg.Services) != 1 {
		t.Fatalf("expected one service in fixture, got %d", len(cfg.Services))
	}
	relationships := cfg.Services[0].Relationships
	if len(relationships) != 1 {
		t.Fatalf("expected one relationship in fixture, got %#v", relationships)
	}
	relationship := relationships[0]
	if relationship.Multiplicity != "one-to-many" {
		t.Fatalf("expected one-to-many relationship, got %q", relationship.Multiplicity)
	}
	if relationship.PrincipalEntity != "Customer" || relationship.DependentEntity != "Order" {
		t.Fatalf("expected Customer -> Order relationship, got %#v", relationship)
	}
	if relationship.ForeignKeyName != "CustomerId" || relationship.PrincipalNavigation != "Orders" || relationship.DependentNavigation != "Customer" {
		t.Fatalf("expected supported navigation/FK fields, got %#v", relationship)
	}
}

func assertDoesNotClaimUnsupportedRelationshipRelease(t *testing.T, content string) {
	t.Helper()
	for _, forbidden := range []string{"one-to-one", "many-to-many", "cross-service"} {
		if strings.Contains(strings.ToLower(content), forbidden) && !strings.Contains(strings.ToLower(content), "not "+forbidden) && !strings.Contains(strings.ToLower(content), "no "+forbidden) && !strings.Contains(strings.ToLower(content), "does not add "+forbidden) {
			t.Fatalf("expected unsupported relationship scope to remain excluded, found unqualified %q in %q", forbidden, content)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertContainsText(t *testing.T, content, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected content to contain %q", expected)
	}
}
