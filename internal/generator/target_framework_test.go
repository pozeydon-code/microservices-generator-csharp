package generator

import (
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestDependencyPolicyManifestLoads(t *testing.T) {
	policies, commonPackages, packageOrder, err := loadDependencyPolicies(dependencyPolicyManifestJSON)
	if err != nil {
		t.Fatalf("load dependency policy manifest: %v", err)
	}
	if len(policies) != 3 {
		t.Fatalf("expected three manifest policies, got %d", len(policies))
	}
	for _, targetMajor := range []int{8, 9, 10} {
		if _, ok := policies[targetMajor]; !ok {
			t.Fatalf("manifest is missing targetMajor %d", targetMajor)
		}
	}
	if len(commonPackages) != 12 {
		t.Fatalf("expected twelve common package pins, got %d", len(commonPackages))
	}
	if len(packageOrder) != 18 {
		t.Fatalf("expected eighteen package names in manifest order, got %d", len(packageOrder))
	}
}

func TestDependencyPolicyManifestMatchesRenderedPackageProps(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	for _, targetFramework := range []string{"net8.0", "net9.0", "net10.0"} {
		t.Run(targetFramework, func(t *testing.T) {
			cfg := testConfig()
			cfg.Generation.TargetFramework = targetFramework
			files, err := gen.Generate(cfg)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			var props struct {
				ItemGroups []struct {
					Packages []struct {
						Include string `xml:"Include,attr"`
						Version string `xml:"Version,attr"`
					} `xml:"PackageVersion"`
				} `xml:"ItemGroup"`
			}
			if err := xml.Unmarshal(generatedContent(t, files, "Directory.Packages.props"), &props); err != nil {
				t.Fatalf("parse generated package props: %v", err)
			}
			actual := map[string]string{}
			for _, group := range props.ItemGroups {
				for _, packageVersion := range group.Packages {
					actual[packageVersion.Include] = packageVersion.Version
				}
			}
			policy, ok := dependencyPolicyForTargetFramework(targetFramework)
			if !ok {
				t.Fatalf("expected %s policy", targetFramework)
			}
			expected := clonePackageMap(dependencyPolicyCommonPackages)
			for packageName, version := range policy.Packages {
				expected[packageName] = version
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("rendered package set mismatch\nexpected: %#v\nactual:   %#v", expected, actual)
			}
			if len(actual) != len(dependencyPolicyPackageOrder) {
				t.Fatalf("expected %d rendered packages, got %d", len(dependencyPolicyPackageOrder), len(actual))
			}
		})
	}
}

func TestLoadDependencyPoliciesRejectsInvalidManifests(t *testing.T) {
	validPolicy := dependencyPolicy{
		TargetMajor: 8,
		Verified:    true,
		Packages: map[string]string{
			"Microsoft.AspNetCore.Authentication.JwtBearer": "8.0.28",
			"Microsoft.AspNetCore.Mvc.Testing":              "8.0.28",
			"Microsoft.EntityFrameworkCore.Design":          "8.0.28",
			"Microsoft.EntityFrameworkCore.SqlServer":       "8.0.28",
			"Microsoft.Data.SqlClient":                      "6.1.1",
			"System.Security.Cryptography.Xml":              "8.0.4",
		},
	}
	tests := []struct {
		name     string
		manifest []byte
		wantErr  string
	}{
		{name: "malformed JSON", manifest: []byte(`{`), wantErr: "decode manifest"},
		{name: "unsupported schema", manifest: marshalDependencyPolicyManifest(t, 2, validPolicy), wantErr: "unsupported schemaVersion"},
		{name: "missing policies", manifest: marshalDependencyPolicyManifest(t, 1), wantErr: "at least one entry"},
		{name: "duplicate target", manifest: marshalDependencyPolicyManifest(t, 1, validPolicy, validPolicy), wantErr: "duplicate targetMajor"},
		{name: "unverified entry", manifest: marshalDependencyPolicyManifest(t, 1, func() dependencyPolicy { policy := validPolicy; policy.Verified = false; return policy }()), wantErr: "not marked verified"},
		{name: "missing version", manifest: marshalDependencyPolicyManifest(t, 1, func() dependencyPolicy {
			policy := cloneDependencyPolicy(validPolicy)
			policy.Packages["Microsoft.Data.SqlClient"] = ""
			return policy
		}()), wantErr: "semantic version"},
		{name: "invalid version", manifest: marshalDependencyPolicyManifest(t, 1, func() dependencyPolicy {
			policy := cloneDependencyPolicy(validPolicy)
			policy.Packages["Microsoft.Data.SqlClient"] = "latest"
			return policy
		}()), wantErr: "semantic version"},
		{name: "ASP.NET and EF drift", manifest: marshalDependencyPolicyManifest(t, 1, func() dependencyPolicy {
			policy := cloneDependencyPolicy(validPolicy)
			policy.Packages["Microsoft.EntityFrameworkCore.Design"] = "8.0.27"
			return policy
		}()), wantErr: "align"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := loadDependencyPolicies(tt.manifest)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("load dependency policies error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDependencyPoliciesRejectsInvalidCommonPackages(t *testing.T) {
	validPolicy := dependencyPolicy{
		TargetMajor: 8,
		Verified:    true,
		Packages: map[string]string{
			"Microsoft.AspNetCore.Authentication.JwtBearer": "8.0.28",
			"Microsoft.AspNetCore.Mvc.Testing":              "8.0.28",
			"Microsoft.EntityFrameworkCore.Design":          "8.0.28",
			"Microsoft.EntityFrameworkCore.SqlServer":       "8.0.28",
			"Microsoft.Data.SqlClient":                      "6.1.1",
			"System.Security.Cryptography.Xml":              "8.0.4",
		},
	}
	tests := []struct {
		name   string
		mutate func(dependencyPolicyManifest)
		want   string
	}{
		{name: "missing common pin", mutate: func(manifest dependencyPolicyManifest) { delete(manifest.CommonPackages, "MediatR") }, want: "missing from targetMajor"},
		{name: "empty common pin", mutate: func(manifest dependencyPolicyManifest) { manifest.CommonPackages["MediatR"] = "" }, want: "commonPackages"},
		{name: "invalid common pin", mutate: func(manifest dependencyPolicyManifest) { manifest.CommonPackages["MediatR"] = "latest" }, want: "commonPackages"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validDependencyPolicyManifest(1, validPolicy)
			tt.mutate(manifest)
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatalf("marshal manifest: %v", err)
			}
			_, _, _, err = loadDependencyPolicies(data)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("load dependency policies error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func marshalDependencyPolicyManifest(t *testing.T, schemaVersion int, policies ...dependencyPolicy) []byte {
	t.Helper()
	data, err := json.Marshal(validDependencyPolicyManifest(schemaVersion, policies...))
	if err != nil {
		t.Fatalf("marshal dependency policy manifest: %v", err)
	}
	return data
}

func validDependencyPolicyManifest(schemaVersion int, policies ...dependencyPolicy) dependencyPolicyManifest {
	return dependencyPolicyManifest{
		SchemaVersion:  schemaVersion,
		PackageOrder:   append([]string(nil), dependencyPolicyPackageOrder...),
		CommonPackages: clonePackageMap(dependencyPolicyCommonPackages),
		Policies:       policies,
	}
}

func clonePackageMap(packages map[string]string) map[string]string {
	clone := make(map[string]string, len(packages))
	for name, version := range packages {
		clone[name] = version
	}
	return clone
}

func cloneDependencyPolicy(policy dependencyPolicy) dependencyPolicy {
	policy.Packages = clonePackageMap(policy.Packages)
	return policy
}

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
				if !reflect.DeepEqual(actual, dependencyPolicy{}) {
					t.Fatalf("expected no policy for %q, got %#v", tt.target, actual)
				}
				return
			}
			expected := dependencyPolicy{
				TargetMajor: tt.targetMajor,
				Verified:    true,
				Packages: map[string]string{
					"Microsoft.AspNetCore.Authentication.JwtBearer": tt.aspNetCore,
					"Microsoft.AspNetCore.Mvc.Testing":              tt.aspNetCoreTest,
					"Microsoft.EntityFrameworkCore.Design":          tt.entityFramework,
					"Microsoft.EntityFrameworkCore.SqlServer":       tt.entityFramework,
					"Microsoft.Data.SqlClient":                      tt.sqlClient,
					"System.Security.Cryptography.Xml":              tt.cryptographyXML,
				},
			}
			if !reflect.DeepEqual(actual, expected) {
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
