package version

import "fmt"

const (
	DevelopmentVersion = "dev"
	UnknownCommit      = "unknown"
	UnknownDate        = "unknown"
)

// These variables are replaced by release builds through linker flags.
var (
	Version = DevelopmentVersion
	Commit  = UnknownCommit
	Date    = UnknownDate
)

type Info struct {
	Version string
	Commit  string
	Date    string
}

func Current() Info {
	return Info{
		Version: fallback(Version, DevelopmentVersion),
		Commit:  fallback(Commit, UnknownCommit),
		Date:    fallback(Date, UnknownDate),
	}
}

func Format(info Info) string {
	return fmt.Sprintf("microgen version: %s\ncommit: %s\ndate: %s\n", info.Version, info.Commit, info.Date)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
