package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pozeydon-code/generator-microservices-go/internal/generator"
)

func TestFilesystemWriterRefusesExistingDirectoryUnlessForcedAndOwned(t *testing.T) {
	outputDir := t.TempDir()
	writer := NewFilesystemWriter()
	files := []generator.GeneratedFile{{Path: "README.md", Content: []byte("first\n")}}

	if err := writer.Write(outputDir, files, false); err == nil {
		t.Fatal("expected non-force existing directory error")
	} else if !strings.Contains(err.Error(), "refusing to overwrite existing directory") {
		t.Fatalf("expected overwrite guidance, got %v", err)
	}
	if err := writer.Write(outputDir, files, true); err == nil {
		t.Fatal("expected unrelated directory force error")
	} else if !strings.Contains(err.Error(), "not owned by microgen") {
		t.Fatalf("expected ownership guidance, got %v", err)
	}
}

func TestFilesystemWriterForceReplacesOwnedOutput(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	writer := NewFilesystemWriter()

	if err := writer.Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("first\n")}}, false); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	if err := writer.Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("second\n")}}, false); err == nil {
		t.Fatal("expected non-force replacement error")
	}
	if err := writer.Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("second\n")}}, true); err != nil {
		t.Fatalf("forced owned replacement: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if string(content) != "second\n" {
		t.Fatalf("expected forced content, got %q", content)
	}
	if !hasValidManifest(outputDir) {
		t.Fatal("expected ownership manifest")
	}
}

func TestFilesystemWriterRefusesForceWhenUserAddsFile(t *testing.T) {
	outputDir := writeInitialOutput(t)
	if err := os.WriteFile(filepath.Join(outputDir, "notes.txt"), []byte("user content\n"), 0o644); err != nil {
		t.Fatalf("add user file: %v", err)
	}

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if !strings.Contains(err.Error(), "unknown file found") {
		t.Fatalf("expected unknown file drift, got %v", err)
	}
}

func TestFilesystemWriterRefusesForceWhenGeneratedFileIsModified(t *testing.T) {
	outputDir := writeInitialOutput(t)
	if err := os.WriteFile(filepath.Join(outputDir, "README.md"), []byte("user edit\n"), 0o644); err != nil {
		t.Fatalf("modify generated file: %v", err)
	}

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if !strings.Contains(err.Error(), "modified file found") {
		t.Fatalf("expected modified file drift, got %v", err)
	}
}

func TestFilesystemWriterRefusesForceWithForgedIncompleteInventory(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	if err := os.MkdirAll(filepath.Join(outputDir, ".microgen"), 0o755); err != nil {
		t.Fatalf("create manifest dir: %v", err)
	}
	manifest := `{"generator":"microgen","version":2,"files":[]}`
	if err := os.WriteFile(filepath.Join(outputDir, ".microgen", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write forged manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "README.md"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected incomplete inventory error")
	}
	if !strings.Contains(err.Error(), "unknown file found") {
		t.Fatalf("expected unknown file from incomplete inventory, got %v", err)
	}
}

func TestFilesystemWriterRefusesForceWithNestedSymlink(t *testing.T) {
	outputDir := writeInitialOutput(t)
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(outputDir, "link")); err != nil {
		t.Fatalf("create nested symlink: %v", err)
	}

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected nested symlink error")
	}
	if !strings.Contains(err.Error(), "symlink found") {
		t.Fatalf("expected symlink drift, got %v", err)
	}
}

func TestFilesystemWriterRejectsOutputRootSymlink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	link := filepath.Join(base, "link")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	err := NewFilesystemWriter().Write(link, []generator.GeneratedFile{{Path: "README.md", Content: []byte("bad")}}, true)
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink message, got %v", err)
	}
}

func TestFilesystemWriterRejectsSymlinkExistingAncestor(t *testing.T) {
	base := t.TempDir()
	realParent := filepath.Join(base, "real-parent")
	linkParent := filepath.Join(base, "link-parent")
	if err := os.Mkdir(realParent, 0o755); err != nil {
		t.Fatalf("create real parent: %v", err)
	}
	if err := os.Symlink(realParent, linkParent); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	err := NewFilesystemWriter().Write(filepath.Join(linkParent, "generated"), []generator.GeneratedFile{{Path: "README.md", Content: []byte("bad")}}, false)
	if err == nil {
		t.Fatal("expected ancestor symlink rejection")
	}
	if !strings.Contains(err.Error(), "existing ancestor") || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected ancestor symlink message, got %v", err)
	}
}

func TestFilesystemWriterRejectsTraversal(t *testing.T) {
	writer := NewFilesystemWriter()
	err := writer.Write(filepath.Join(t.TempDir(), "generated"), []generator.GeneratedFile{{Path: "../escape.txt", Content: []byte("bad")}}, false)
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if !strings.Contains(err.Error(), "escapes the output directory") {
		t.Fatalf("expected traversal message, got %v", err)
	}
}

func TestFilesystemWriterFailsWhenPublicationLockExists(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writeLockMetadata(t, lockPath, outputDir, os.Getpid(), time.Now().UTC())

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false)
	if err == nil {
		t.Fatal("expected lock error")
	}
	if !strings.Contains(err.Error(), "locked by another microgen publication") {
		t.Fatalf("expected lock message, got %v", err)
	}
}

func TestFilesystemWriterWritesPublicationLockMetadata(t *testing.T) {
	withHostIdentity(t, "local-host-id")
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writer := NewFilesystemWriter()
	writer.remove = func(path string) error {
		if path != lockPath {
			t.Fatalf("unexpected lock cleanup path %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read lock metadata before cleanup: %v", err)
		}
		var metadata lockMetadata
		if err := json.Unmarshal(content, &metadata); err != nil {
			t.Fatalf("parse lock metadata: %v", err)
		}
		if metadata.Generator != manifestGenerator || metadata.Version != manifestVersion || metadata.PID != os.Getpid() || metadata.OutputDir != outputDir || metadata.CreatedAtUTC == "" || metadata.HostID == "" || metadata.OwnerToken == "" {
			t.Fatalf("unexpected lock metadata: %#v", metadata)
		}
		return os.Remove(path)
	}

	if _, err := writer.WriteDetailed(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false); err != nil {
		t.Fatalf("write output: %v", err)
	}
}

func TestFilesystemWriterRecoversGuardedStalePublicationLock(t *testing.T) {
	withHostIdentity(t, "local-host-id")
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writeLockMetadata(t, lockPath, outputDir, 999999, time.Now().UTC().Add(-2*lockStaleAfter))

	if err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false); err != nil {
		t.Fatalf("expected stale lock recovery, got %v", err)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected lock cleanup, got %v", err)
	}
}

func TestFilesystemWriterDoesNotRecoverForeignHostStalePublicationLock(t *testing.T) {
	withHostIdentity(t, "local-host-id")
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writeLockMetadataForHost(t, lockPath, outputDir, 999999, time.Now().UTC().Add(-2*lockStaleAfter), "other-host")

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false)
	if err == nil {
		t.Fatal("expected foreign-host stale lock error")
	}
	if !strings.Contains(err.Error(), "foreign-host lock") || !strings.Contains(err.Error(), "remove") {
		t.Fatalf("expected foreign-host recovery guidance, got %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected foreign-host lock to remain, got %v", err)
	}
}

func TestFilesystemWriterDoesNotRecoverSameHostnameDifferentHostIDStalePublicationLock(t *testing.T) {
	withHostIdentity(t, "local-host-id")
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	hostname, _ := os.Hostname()
	writeLockMetadataForIdentity(t, lockPath, outputDir, 999999, time.Now().UTC().Add(-2*lockStaleAfter), hostname, "different-host-id")

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false)
	if err == nil {
		t.Fatal("expected different-host-id stale lock error")
	}
	if !strings.Contains(err.Error(), "foreign-host lock") {
		t.Fatalf("expected foreign-host lock message, got %v", err)
	}
}

func TestFilesystemWriterDoesNotRecoverStalePublicationLockWhenHostIdentityUnavailable(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writeLockMetadata(t, lockPath, outputDir, 999999, time.Now().UTC().Add(-2*lockStaleAfter))
	withHostIdentity(t, "")

	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false)
	if err == nil {
		t.Fatal("expected unavailable-host-id stale lock error")
	}
	if !strings.Contains(err.Error(), "host identity is unavailable") {
		t.Fatalf("expected unavailable host identity guidance, got %v", err)
	}
}

func TestPublicationLockUnlockRefusesOwnershipTokenReplacement(t *testing.T) {
	withHostIdentity(t, "local-host-id")
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	unlock, err := acquireLock(base, "generated", outputDir, os.Remove)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	metadata, err := readLockMetadata(lockPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	metadata.OwnerToken = "replacement-token"
	content, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(lockPath, content, 0o600); err != nil {
		t.Fatalf("replace metadata: %v", err)
	}
	err = unlock()
	if err == nil || !strings.Contains(err.Error(), "ownership changed") {
		t.Fatalf("expected ownership changed error, got %v", err)
	}
}

func TestFilesystemWriterWarnsWhenPublicationLockCleanupFails(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	lockPath := filepath.Join(base, ".generated."+lockFileName)
	writer := NewFilesystemWriter()
	writer.remove = func(path string) error {
		if path == lockPath {
			return errors.New("simulated lock cleanup failure")
		}
		return os.Remove(path)
	}

	result, err := writer.WriteDetailed(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("content\n")}}, false)
	if err != nil {
		t.Fatalf("expected successful write with warning, got %v", err)
	}
	if !strings.Contains(result.Warning, "cleanup of publication lock failed") || !strings.Contains(result.Warning, lockPath) {
		t.Fatalf("expected lock cleanup warning with path, got %q", result.Warning)
	}
}

func TestFilesystemWriterCleansStagingOnWriteFailure(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "../escape.txt", Content: []byte("bad")}}, false)
	if err == nil {
		t.Fatal("expected write failure")
	}
	assertNoStagingDirectories(t, base)
	if _, err := os.Stat(outputDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no published output, got %v", err)
	}
}

func TestFilesystemWriterRestoresOwnedOutputWhenPublishFails(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	writer := NewFilesystemWriter()
	if err := writer.Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("original\n")}}, false); err != nil {
		t.Fatalf("initial write: %v", err)
	}

	renameCalls := 0
	failingWriter := NewFilesystemWriter()
	failingWriter.rename = func(oldPath, newPath string) error {
		renameCalls++
		if renameCalls == 2 {
			return errors.New("simulated publish failure")
		}
		return os.Rename(oldPath, newPath)
	}

	err := failingWriter.Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected publish failure")
	}
	if !strings.Contains(err.Error(), "previous generated directory restored") {
		t.Fatalf("expected rollback message, got %v", err)
	}
	content, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "original\n" {
		t.Fatalf("expected restored content, got %q", content)
	}
	assertNoStagingDirectories(t, base)
}

func TestFilesystemWriterReportsBackupPathWhenPublishAndRollbackFail(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	if err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("original\n")}}, false); err != nil {
		t.Fatalf("initial write: %v", err)
	}

	renameCalls := 0
	failingWriter := NewFilesystemWriter()
	failingWriter.rename = func(oldPath, newPath string) error {
		renameCalls++
		if renameCalls == 2 || renameCalls == 3 {
			return errors.New("simulated rename failure")
		}
		return os.Rename(oldPath, newPath)
	}

	_, err := failingWriter.WriteDetailed(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err == nil {
		t.Fatal("expected double-failure error")
	}
	message := err.Error()
	if !strings.Contains(message, "rollback failed") || !strings.Contains(message, "previous generated directory remains at") || !strings.Contains(message, "recover by moving") {
		t.Fatalf("expected actionable recovery instructions, got %v", err)
	}
}

func TestFilesystemWriterReportsWarningWhenBackupCleanupFailsAfterSuccessfulPublish(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "generated")
	if err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("original\n")}}, false); err != nil {
		t.Fatalf("initial write: %v", err)
	}

	failingWriter := NewFilesystemWriter()
	backupRemoveCalls := 0
	failingWriter.removeAll = func(path string) error {
		if strings.Contains(path, ".microgen-backup-") {
			backupRemoveCalls++
			if backupRemoveCalls == 1 {
				return os.RemoveAll(path)
			}
			return errors.New("simulated cleanup failure")
		}
		return os.RemoveAll(path)
	}

	result, err := failingWriter.WriteDetailed(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("replacement\n")}}, true)
	if err != nil {
		t.Fatalf("expected successful publish with warning, got %v", err)
	}
	if !strings.Contains(result.Warning, "published successfully") || !strings.Contains(result.Warning, "backup") {
		t.Fatalf("expected cleanup warning, got %q", result.Warning)
	}
	content, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("read published file: %v", err)
	}
	if string(content) != "replacement\n" {
		t.Fatalf("expected replacement content, got %q", content)
	}
}

func assertNoStagingDirectories(t *testing.T, base string) {
	t.Helper()
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".microgen-staging-") {
			t.Fatalf("expected staging cleanup, found %s", entry.Name())
		}
	}
}

func writeInitialOutput(t *testing.T) string {
	t.Helper()
	outputDir := filepath.Join(t.TempDir(), "generated")
	if err := NewFilesystemWriter().Write(outputDir, []generator.GeneratedFile{{Path: "README.md", Content: []byte("original\n")}}, false); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	return outputDir
}

func withHostIdentity(t *testing.T, hostID string) {
	t.Helper()
	original := currentHostIdentity
	currentHostIdentity = func() string { return hostID }
	t.Cleanup(func() { currentHostIdentity = original })
}

func writeLockMetadata(t *testing.T, lockPath, outputDir string, pid int, createdAt time.Time) {
	t.Helper()
	hostname, _ := os.Hostname()
	writeLockMetadataForHost(t, lockPath, outputDir, pid, createdAt, hostname)
}

func writeLockMetadataForHost(t *testing.T, lockPath, outputDir string, pid int, createdAt time.Time, hostname string) {
	t.Helper()
	writeLockMetadataForIdentity(t, lockPath, outputDir, pid, createdAt, hostname, currentHostIdentity())
}

func writeLockMetadataForIdentity(t *testing.T, lockPath, outputDir string, pid int, createdAt time.Time, hostname, hostID string) {
	t.Helper()
	metadata := lockMetadata{Generator: manifestGenerator, Version: manifestVersion, PID: pid, Hostname: hostname, HostID: hostID, OwnerToken: "test-owner-token", OutputDir: outputDir, CreatedAtUTC: createdAt.Format(time.RFC3339Nano)}
	content, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal lock metadata: %v", err)
	}
	if err := os.WriteFile(lockPath, content, 0o600); err != nil {
		t.Fatalf("write lock metadata: %v", err)
	}
}
