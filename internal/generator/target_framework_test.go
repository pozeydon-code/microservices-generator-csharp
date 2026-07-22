package generator

import "testing"

func TestDependencyPolicyForTargetFramework(t *testing.T) {
	tests := []struct {
		name            string
		target          string
		targetMajor     int
		aspNetCore      string
		aspNetCoreTest  string
		entityFramework string
		sqlClient       string
		cryptographyXML string
	}{
		{
			name:            "legacy defaults to net8 policy",
			target:          "net7.0",
			targetMajor:     8,
			aspNetCore:      "8.0.28",
			aspNetCoreTest:  "8.0.28",
			entityFramework: "8.0.28",
			sqlClient:       "6.1.1",
			cryptographyXML: "8.0.4",
		},
		{
			name:            "net8",
			target:          "net8.0",
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
			targetMajor:     10,
			aspNetCore:      "10.0.0",
			aspNetCoreTest:  "10.0.0",
			entityFramework: "10.0.0",
			sqlClient:       "6.1.1",
			cryptographyXML: "10.0.10",
		},
		{
			name:            "net11",
			target:          "net11.0",
			targetMajor:     11,
			aspNetCore:      "11.0.0",
			aspNetCoreTest:  "11.0.0",
			entityFramework: "11.0.0",
			sqlClient:       "6.1.1",
			cryptographyXML: "10.0.10",
		},
		{
			name:            "future",
			target:          "net12.0",
			targetMajor:     12,
			aspNetCore:      "12.0.0",
			aspNetCoreTest:  "12.0.0",
			entityFramework: "12.0.0",
			sqlClient:       "6.1.1",
			cryptographyXML: "10.0.10",
		},
		{
			name:            "invalid defaults to net8 policy",
			target:          "netx",
			targetMajor:     8,
			aspNetCore:      "8.0.28",
			aspNetCoreTest:  "8.0.28",
			entityFramework: "8.0.28",
			sqlClient:       "6.1.1",
			cryptographyXML: "8.0.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := dependencyPolicyForTargetFramework(tt.target)
			expected := dependencyPolicy{
				TargetMajor:                       tt.targetMajor,
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
