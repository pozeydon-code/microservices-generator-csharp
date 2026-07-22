package generator

import (
	"fmt"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

type dependencyPolicy struct {
	TargetMajor                       int
	AspNetCorePackageVersion          string
	AspNetCoreTestingPackageVersion   string
	EntityFrameworkCorePackageVersion string
	SqlClientPackageVersion           string
	CryptographyXmlPackageVersion     string
}

var dependencyPoliciesByTargetMajor = map[int]dependencyPolicy{
	8: {
		TargetMajor:                       8,
		AspNetCorePackageVersion:          "8.0.28",
		AspNetCoreTestingPackageVersion:   "8.0.28",
		EntityFrameworkCorePackageVersion: "8.0.28",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "8.0.4",
	},
	9: {
		TargetMajor:                       9,
		AspNetCorePackageVersion:          "9.0.7",
		AspNetCoreTestingPackageVersion:   "9.0.7",
		EntityFrameworkCorePackageVersion: "9.0.7",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "9.0.18",
	},
	10: {
		TargetMajor:                       10,
		AspNetCorePackageVersion:          "10.0.0",
		AspNetCoreTestingPackageVersion:   "10.0.0",
		EntityFrameworkCorePackageVersion: "10.0.0",
		SqlClientPackageVersion:           "6.1.1",
		CryptographyXmlPackageVersion:     "10.0.10",
	},
}

func dependencyPolicyForTargetFramework(targetFramework string) dependencyPolicy {
	major, ok := spec.TargetFrameworkMajor(targetFramework)
	if !ok || major <= 8 {
		return dependencyPoliciesByTargetMajor[8]
	}
	if policy, ok := dependencyPoliciesByTargetMajor[major]; ok {
		return policy
	}
	majorAlignedVersion := fmt.Sprintf("%d.0.0", major)
	return dependencyPolicy{
		TargetMajor:                       major,
		AspNetCorePackageVersion:          majorAlignedVersion,
		AspNetCoreTestingPackageVersion:   majorAlignedVersion,
		EntityFrameworkCorePackageVersion: majorAlignedVersion,
		SqlClientPackageVersion:           dependencyPoliciesByTargetMajor[10].SqlClientPackageVersion,
		CryptographyXmlPackageVersion:     dependencyPoliciesByTargetMajor[10].CryptographyXmlPackageVersion,
	}
}
