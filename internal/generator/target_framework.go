package generator

import (
	"fmt"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

func dotNetPackageVersion(targetFramework string) string {
	major, ok := spec.TargetFrameworkMajor(targetFramework)
	if !ok || major <= 8 {
		return "8.0.28"
	}
	if major == 9 {
		return "9.0.7"
	}
	return fmt.Sprintf("%d.0.0", major)
}
