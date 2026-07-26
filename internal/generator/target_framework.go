package generator

import (
	"fmt"
	"sort"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

type dependencyPolicy struct {
	TargetMajor                       int
	MediatRVersion                    string
	FluentValidationVersion           string
	ErrorOrVersion                    string
	AspNetCorePackageVersion          string
	AspNetCoreTestingPackageVersion   string
	EntityFrameworkCorePackageVersion string
	SqlClientPackageVersion           string
	CryptographyXmlPackageVersion     string
}

var dependencyPoliciesByTargetMajor = map[int]dependencyPolicy{
	8: {
		TargetMajor:                       8,
		MediatRVersion:                    "14.2.0",
		FluentValidationVersion:           "12.1.1",
		ErrorOrVersion:                    "2.1.1",
		AspNetCorePackageVersion:          "8.0.28",
		AspNetCoreTestingPackageVersion:   "8.0.28",
		EntityFrameworkCorePackageVersion: "8.0.28",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "8.0.4",
	},
	9: {
		TargetMajor:                       9,
		MediatRVersion:                    "14.2.0",
		FluentValidationVersion:           "12.1.1",
		ErrorOrVersion:                    "2.1.1",
		AspNetCorePackageVersion:          "9.0.7",
		AspNetCoreTestingPackageVersion:   "9.0.7",
		EntityFrameworkCorePackageVersion: "9.0.7",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "9.0.18",
	},
	10: {
		TargetMajor:                       10,
		MediatRVersion:                    "14.2.0",
		FluentValidationVersion:           "12.1.1",
		ErrorOrVersion:                    "2.1.1",
		AspNetCorePackageVersion:          "10.0.0",
		AspNetCoreTestingPackageVersion:   "10.0.0",
		EntityFrameworkCorePackageVersion: "10.0.0",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "10.0.10",
	},
}

func dependencyPolicyForTargetFramework(targetFramework string) (dependencyPolicy, bool) {
	major, ok := spec.TargetFrameworkMajor(targetFramework)
	if !ok {
		return dependencyPolicy{}, false
	}
	policy, ok := dependencyPoliciesByTargetMajor[major]
	return policy, ok
}

func IsPolicyBackedTargetFramework(targetFramework string) bool {
	_, ok := dependencyPolicyForTargetFramework(targetFramework)
	return ok
}

func SupportedTargetFrameworks() []string {
	majors := make([]int, 0, len(dependencyPoliciesByTargetMajor))
	for major := range dependencyPoliciesByTargetMajor {
		majors = append(majors, major)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(majors)))
	frameworks := make([]string, 0, len(majors))
	for _, major := range majors {
		frameworks = append(frameworks, fmt.Sprintf("net%d.0", major))
	}
	return frameworks
}

func ValidateTargetFrameworkPolicy(targetFramework string) error {
	if IsPolicyBackedTargetFramework(targetFramework) {
		return nil
	}
	return fmt.Errorf("generation.targetFramework %q has no verified dependency policy entry; a new target requires an explicit verified policy entry in internal/generator/target_framework.go", targetFramework)
}
