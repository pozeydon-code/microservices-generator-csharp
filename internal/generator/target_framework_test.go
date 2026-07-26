package generator

import (
	"strings"
	"testing"
)

func TestDependencyPolicyForTargetFramework(t *testing.T) {
	tests := []struct {
		name            string
		target          string
		wantOK          bool
		targetMajor     int
		aspNetCore      string
		aspNetCoreTest  string
		entityFramework string
		sqlClient       string
		cryptographyXML string
	}{
		{
			name:            "net8",
			target:          "net8.0",
			wantOK:          true,
			targetMajor:     8,
			aspNetCore:      "8.0.28",
			aspNetCoreTest:  "8.0.28",
			entityFramework: "8.0.28",
			sqlClient:       "6.1.1",
			cryptographyXML: "8.0.4",
		},
		{
			name:            "net9",
			target:          "net9.0",
			wantOK:          true,
			targetMajor:     9,
			aspNetCore:      "9.0.7",
			aspNetCoreTest:  "9.0.7",
			entityFramework: "9.0.7",
			sqlClient:       "6.1.1",
			cryptographyXML: "9.0.18",
		},
		{
			name:            "net10",
			target:          "net10.0",
			wantOK:          true,
			targetMajor:     10,
			aspNetCore:      "10.0.0",
			aspNetCoreTest:  "10.0.0",
			entityFramework: "10.0.0",
			sqlClient:       "6.1.1",
			cryptographyXML: "10.0.10",
		},
		{name: "net11", target: "net11.0"},
		{name: "unknown major", target: "net99.0"},
		{name: "invalid target", target: "netx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := dependencyPolicyForTargetFramework(tt.target)
			if ok != tt.wantOK {
				t.Fatalf("dependencyPolicyForTargetFramework(%q) found=%t, want %t", tt.target, ok, tt.wantOK)
			}
			if !tt.wantOK {
				if actual != (dependencyPolicy{}) {
					t.Fatalf("expected no policy for %q, got %#v", tt.target, actual)
				}
				return
			}
			expected := dependencyPolicy{
				TargetMajor:                       tt.targetMajor,
				MediatRVersion:                    "14.2.0",
				FluentValidationVersion:           "12.1.1",
				ErrorOrVersion:                    "2.1.1",
				AspNetCorePackageVersion:          tt.aspNetCore,
				AspNetCoreTestingPackageVersion:   tt.aspNetCoreTest,
				EntityFrameworkCorePackageVersion: tt.entityFramework,
				SqlClientPackageVersion:           tt.sqlClient,
				CryptographyXmlPackageVersion:     tt.cryptographyXML,
			}
			if actual != expected {
				t.Fatalf("unexpected dependency policy\nexpected: %#v\nactual:   %#v", expected, actual)
			}
		})
	}
}

func TestSupportedTargetFrameworksComeFromDependencyPolicies(t *testing.T) {
	want := []string{"net10.0", "net9.0", "net8.0"}
	got := SupportedTargetFrameworks()
	if len(got) != len(want) {
		t.Fatalf("expected supported targets %#v, got %#v", want, got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected supported targets %#v, got %#v", want, got)
		}
	}
}

func TestValidateTargetFrameworkPolicyExplainsUnverifiedTargets(t *testing.T) {
	if err := ValidateTargetFrameworkPolicy("net8.0"); err != nil {
		t.Fatalf("expected net8.0 policy to validate, got %v", err)
	}
	err := ValidateTargetFrameworkPolicy("net11.0")
	if err == nil {
		t.Fatal("expected net11.0 policy validation to fail")
	}
	if want := "a new target requires an explicit verified policy entry"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error to contain %q, got %v", want, err)
	}
}
