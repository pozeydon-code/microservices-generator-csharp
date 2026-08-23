package generator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

type dependencyPolicy struct {
	TargetMajor int               `json:"targetMajor"`
	Verified    bool              `json:"verified"`
	Packages    map[string]string `json:"packages"`
}

type dependencyPolicyManifest struct {
	SchemaVersion  int                `json:"schemaVersion"`
	PackageOrder   []string           `json:"packageOrder"`
	CommonPackages map[string]string  `json:"commonPackages"`
	Policies       []dependencyPolicy `json:"policies"`
}

//go:embed policy/dependency-policy.json
var dependencyPolicyManifestJSON []byte

var (
	packageVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	packageNamePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

	dependencyPoliciesByTargetMajor, dependencyPolicyCommonPackages, dependencyPolicyPackageOrder, dependencyPolicyManifestErr = loadDependencyPolicies(dependencyPolicyManifestJSON)
)

const dependencyPolicyManifestPath = "internal/generator/policy/dependency-policy.json"

var targetSpecificPackageNames = []string{
	"Microsoft.AspNetCore.Authentication.JwtBearer",
	"Microsoft.AspNetCore.OpenApi",
	"Microsoft.AspNetCore.Mvc.Testing",
	"Microsoft.EntityFrameworkCore.Design",
	"Microsoft.EntityFrameworkCore.Tools",
	"Microsoft.EntityFrameworkCore.SqlServer",
	"Microsoft.Data.SqlClient",
	"System.Security.Cryptography.Xml",
}

func loadDependencyPolicies(data []byte) (map[int]dependencyPolicy, map[string]string, []string, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest dependencyPolicyManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, nil, nil, fmt.Errorf("decode manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, nil, nil, fmt.Errorf("manifest contains trailing JSON")
		}
		return nil, nil, nil, fmt.Errorf("decode trailing manifest data: %w", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, nil, nil, fmt.Errorf("unsupported schemaVersion %d, want 1", manifest.SchemaVersion)
	}
	if err := validatePackageOrder(manifest.PackageOrder); err != nil {
		return nil, nil, nil, err
	}
	if err := validateCommonPackages(manifest.CommonPackages, manifest.PackageOrder); err != nil {
		return nil, nil, nil, err
	}
	if len(manifest.Policies) == 0 {
		return nil, nil, nil, fmt.Errorf("policies must contain at least one entry")
	}

	policies := make(map[int]dependencyPolicy, len(manifest.Policies))
	for index, policy := range manifest.Policies {
		if err := validateDependencyPolicy(policy, manifest.CommonPackages, manifest.PackageOrder); err != nil {
			return nil, nil, nil, fmt.Errorf("policies[%d]: %w", index, err)
		}
		if _, exists := policies[policy.TargetMajor]; exists {
			return nil, nil, nil, fmt.Errorf("policies[%d]: duplicate targetMajor %d", index, policy.TargetMajor)
		}
		policies[policy.TargetMajor] = policy
	}
	return policies, manifest.CommonPackages, manifest.PackageOrder, nil
}

func validatePackageOrder(packageOrder []string) error {
	if len(packageOrder) == 0 {
		return fmt.Errorf("packageOrder must contain at least one package")
	}
	seen := make(map[string]struct{}, len(packageOrder))
	for index, packageName := range packageOrder {
		if !packageNamePattern.MatchString(packageName) {
			return fmt.Errorf("packageOrder[%d] must contain a valid package name, got %q", index, packageName)
		}
		if _, exists := seen[packageName]; exists {
			return fmt.Errorf("packageOrder[%d]: duplicate package %q", index, packageName)
		}
		seen[packageName] = struct{}{}
	}
	return nil
}

func validateCommonPackages(commonPackages map[string]string, packageOrder []string) error {
	if len(commonPackages) == 0 {
		return fmt.Errorf("commonPackages must contain at least one package")
	}
	orderedPackages := make(map[string]struct{}, len(packageOrder))
	for _, packageName := range packageOrder {
		orderedPackages[packageName] = struct{}{}
	}
	for _, packageName := range sortedPackageNames(commonPackages) {
		version := commonPackages[packageName]
		if !packageNamePattern.MatchString(packageName) {
			return fmt.Errorf("commonPackages[%q] must contain a valid package name", packageName)
		}
		if _, ok := orderedPackages[packageName]; !ok {
			return fmt.Errorf("commonPackages[%q] is missing from packageOrder", packageName)
		}
		if !packageVersionPattern.MatchString(version) {
			return fmt.Errorf("commonPackages[%q] must be a verified semantic version, got %q", packageName, version)
		}
	}
	return nil
}

func validateDependencyPolicy(policy dependencyPolicy, commonPackages map[string]string, packageOrder []string) error {
	if policy.TargetMajor <= 0 {
		return fmt.Errorf("targetMajor must be positive")
	}
	if !policy.Verified {
		return fmt.Errorf("targetMajor %d is not marked verified", policy.TargetMajor)
	}
	if len(policy.Packages) == 0 {
		return fmt.Errorf("targetMajor %d packages must contain at least one package", policy.TargetMajor)
	}
	orderedPackages := make(map[string]struct{}, len(packageOrder))
	for _, packageName := range packageOrder {
		orderedPackages[packageName] = struct{}{}
	}
	for _, packageName := range sortedPackageNames(policy.Packages) {
		version := policy.Packages[packageName]
		if !packageNamePattern.MatchString(packageName) {
			return fmt.Errorf("packages[%q] must contain a valid package name", packageName)
		}
		if _, ok := orderedPackages[packageName]; !ok {
			return fmt.Errorf("packages[%q] is missing from packageOrder", packageName)
		}
		if _, ok := commonPackages[packageName]; ok {
			return fmt.Errorf("packages[%q] duplicates a common package", packageName)
		}
		if !packageVersionPattern.MatchString(version) {
			return fmt.Errorf("packages[%q] must be a verified semantic version, got %q", packageName, version)
		}
	}
	for _, packageName := range targetSpecificPackageNames {
		if _, common := commonPackages[packageName]; common {
			return fmt.Errorf("packages[%q] must remain target-specific", packageName)
		}
		if _, ok := policy.Packages[packageName]; !ok {
			return fmt.Errorf("packages[%q] is missing from targetMajor %d policy", packageName, policy.TargetMajor)
		}
	}
	for _, packageName := range packageOrder {
		if _, common := commonPackages[packageName]; common {
			continue
		}
		if _, targetSpecific := policy.Packages[packageName]; !targetSpecific {
			return fmt.Errorf("packages[%q] is missing from targetMajor %d policy", packageName, policy.TargetMajor)
		}
	}
	aspNetCoreVersion := policy.Packages["Microsoft.AspNetCore.Authentication.JwtBearer"]
	aspNetCoreOpenAPIVersion := policy.Packages["Microsoft.AspNetCore.OpenApi"]
	aspNetCoreTestingVersion := policy.Packages["Microsoft.AspNetCore.Mvc.Testing"]
	entityFrameworkVersion := policy.Packages["Microsoft.EntityFrameworkCore.Design"]
	entityFrameworkToolsVersion := policy.Packages["Microsoft.EntityFrameworkCore.Tools"]
	entityFrameworkSQLVersion := policy.Packages["Microsoft.EntityFrameworkCore.SqlServer"]
	for _, pkg := range []struct {
		name    string
		version string
	}{
		{name: "Microsoft.AspNetCore.Authentication.JwtBearer", version: aspNetCoreVersion},
		{name: "Microsoft.AspNetCore.OpenApi", version: aspNetCoreOpenAPIVersion},
		{name: "Microsoft.AspNetCore.Mvc.Testing", version: aspNetCoreTestingVersion},
		{name: "Microsoft.EntityFrameworkCore.Design", version: entityFrameworkVersion},
		{name: "Microsoft.EntityFrameworkCore.Tools", version: entityFrameworkToolsVersion},
		{name: "Microsoft.EntityFrameworkCore.SqlServer", version: entityFrameworkSQLVersion},
	} {
		major, ok := packageVersionMajor(pkg.version)
		if !ok || major != policy.TargetMajor {
			return fmt.Errorf("packages[%q] major %d does not match targetMajor %d", pkg.name, major, policy.TargetMajor)
		}
	}
	if aspNetCoreVersion != aspNetCoreOpenAPIVersion || aspNetCoreVersion != aspNetCoreTestingVersion {
		return fmt.Errorf("ASP.NET Core package versions must align: %q, %q, %q", aspNetCoreVersion, aspNetCoreOpenAPIVersion, aspNetCoreTestingVersion)
	}
	if entityFrameworkVersion != aspNetCoreVersion || entityFrameworkToolsVersion != aspNetCoreVersion || entityFrameworkSQLVersion != aspNetCoreVersion {
		return fmt.Errorf("Entity Framework Core and ASP.NET Core package versions must align: %q, %q, %q, %q", entityFrameworkVersion, entityFrameworkToolsVersion, entityFrameworkSQLVersion, aspNetCoreVersion)
	}
	return nil
}

func sortedPackageNames(packages map[string]string) []string {
	names := make([]string, 0, len(packages))
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func packageVersionMajor(version string) (int, bool) {
	core := strings.SplitN(version, "-", 2)[0]
	core = strings.SplitN(core, "+", 2)[0]
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return 0, false
	}
	major, err := strconv.Atoi(parts[0])
	return major, err == nil
}

func dependencyPolicyForTargetFramework(targetFramework string) (dependencyPolicy, bool) {
	if dependencyPolicyManifestErr != nil {
		return dependencyPolicy{}, false
	}
	major, ok := spec.TargetFrameworkMajor(targetFramework)
	if !ok {
		return dependencyPolicy{}, false
	}
	policy, ok := dependencyPoliciesByTargetMajor[major]
	return policy, ok
}

func dependencyPackageVersion(policy dependencyPolicy, packageName string) string {
	if version, ok := dependencyPolicyCommonPackages[packageName]; ok {
		return version
	}
	return policy.Packages[packageName]
}

func dependencyPackageVersions(policy dependencyPolicy, includeGateway bool) []PackageVersion {
	packages := make([]PackageVersion, 0, len(dependencyPolicyPackageOrder))
	for _, packageName := range dependencyPolicyPackageOrder {
		if packageName == "Yarp.ReverseProxy" && !includeGateway {
			continue
		}
		comment := ""
		if packageName == "System.Security.Cryptography.Xml" {
			comment = "<!-- Pinned for NuGet audit safety when EF/Core build transitives request vulnerable XML versions. -->"
		}
		if packageName == "Microsoft.OpenApi" {
			comment = "<!-- Pinned for NuGet audit safety when OpenAPI generators request vulnerable transitives. -->"
		}
		packages = append(packages, PackageVersion{Name: packageName, Version: dependencyPackageVersion(policy, packageName), Comment: comment})
	}
	return packages
}

func IsPolicyBackedTargetFramework(targetFramework string) bool {
	_, ok := dependencyPolicyForTargetFramework(targetFramework)
	return ok
}

func supportsOpenApiEndpoints(targetFramework string) bool {
	major, ok := spec.TargetFrameworkMajor(targetFramework)
	return ok && major >= 9
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
	if dependencyPolicyManifestErr != nil {
		return fmt.Errorf("dependency policy manifest %q is invalid: %w", dependencyPolicyManifestPath, dependencyPolicyManifestErr)
	}
	if IsPolicyBackedTargetFramework(targetFramework) {
		return nil
	}
	return fmt.Errorf("generation.targetFramework %q has no verified dependency policy entry; a new target requires an explicit verified policy entry in %s", targetFramework, dependencyPolicyManifestPath)
}
