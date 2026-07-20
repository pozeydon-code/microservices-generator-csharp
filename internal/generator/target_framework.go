package generator

func dotNetPackageVersion(targetFramework string) string {
	switch targetFramework {
	case "net9.0":
		return "9.0.7"
	default:
		return "8.0.28"
	}
}
