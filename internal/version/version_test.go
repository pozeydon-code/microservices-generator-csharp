package version

import "testing"

func TestFormatUsesBuildMetadata(t *testing.T) {
	info := Info{Version: "1.2.3", Commit: "abc123", Date: "2026-07-26"}
	want := "microgen version: 1.2.3\ncommit: abc123\ndate: 2026-07-26\n"
	if got := Format(info); got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestCurrentUsesDeterministicFallbacks(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	Version, Commit, Date = "", "", ""
	t.Cleanup(func() { Version, Commit, Date = originalVersion, originalCommit, originalDate })

	want := Info{Version: DevelopmentVersion, Commit: UnknownCommit, Date: UnknownDate}
	if got := Current(); got != want {
		t.Fatalf("Current() = %#v, want %#v", got, want)
	}
}
