package output

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/pozeydon-code/generator-microservices-go/internal/generator"
)

const (
	manifestDirName   = ".microgen"
	manifestFileName  = "manifest.json"
	lockFileName      = ".microgen.publish.lock"
	manifestVersion   = 2
	manifestGenerator = "microgen"
	lockStaleAfter    = time.Hour
)

var currentHostIdentity = localHostIdentity

type Manifest struct {
	Generator string         `json:"generator"`
	Version   int            `json:"version"`
	Files     []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type WriteResult struct {
	OutputDir     string
	Action        PlanAction
	ForceRequired bool
	ForceUsed     bool
	Warning       string
}

type PlanAction string

const (
	PlanActionCreate    PlanAction = "create"
	PlanActionReplace   PlanAction = "replace"
	PlanActionUnchanged PlanAction = "unchanged"
)

type Plan struct {
	OutputDir     string
	Action        PlanAction
	ForceRequired bool
	ForceUsed     bool
	Files         []PlannedFile
}

type PlannedFile struct {
	Path   string
	Action PlanAction
}

type FilesystemWriter struct {
	rename    func(oldPath, newPath string) error
	removeAll func(path string) error
	remove    func(path string) error
}

func NewFilesystemWriter() FilesystemWriter {
	return FilesystemWriter{rename: os.Rename, removeAll: os.RemoveAll, remove: os.Remove}
}

func PlanOutput(outputDir string, files []generator.GeneratedFile, force bool) (Plan, error) {
	root, err := CanonicalPublishPath(outputDir)
	if err != nil {
		return Plan{}, err
	}
	state, err := inspectRoot(root)
	if err != nil {
		return Plan{OutputDir: root}, err
	}
	if state.exists {
		if !state.isDir {
			return Plan{OutputDir: root}, validatePublicationState(root, state, force)
		}
		if err := validatePublicationState(root, state, true); err != nil {
			return Plan{OutputDir: root}, err
		}
	} else if err := validatePublicationState(root, state, force); err != nil {
		return Plan{OutputDir: root}, err
	}

	action := PlanActionCreate
	if state.exists {
		action = PlanActionReplace
	}
	plannedFiles := make([]PlannedFile, 0, len(files))
	for _, file := range files {
		cleanPath, err := cleanGeneratedPath(file.Path)
		if err != nil {
			return Plan{OutputDir: root}, err
		}
		fileAction := action
		if state.exists {
			fileAction, err = planExistingFileAction(root, cleanPath, file.Content)
			if err != nil {
				return Plan{OutputDir: root}, err
			}
		}
		plannedFiles = append(plannedFiles, PlannedFile{Path: cleanPath, Action: fileAction})
	}
	sort.Slice(plannedFiles, func(i, j int) bool { return plannedFiles[i].Path < plannedFiles[j].Path })

	return Plan{
		OutputDir:     root,
		Action:        action,
		ForceRequired: state.exists,
		ForceUsed:     state.exists && force,
		Files:         plannedFiles,
	}, nil
}

func planExistingFileAction(root, cleanPath string, content []byte) (PlanAction, error) {
	target := filepath.Join(root, filepath.FromSlash(cleanPath))
	existing, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return PlanActionCreate, nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect generated file %s: %w", cleanPath, err)
	}
	if bytes.Equal(existing, content) {
		return PlanActionUnchanged, nil
	}
	return PlanActionReplace, nil
}

func (w FilesystemWriter) Write(outputDir string, files []generator.GeneratedFile, force bool) error {
	_, err := w.WriteDetailed(outputDir, files, force)
	return err
}

func (w FilesystemWriter) WriteDetailed(outputDir string, files []generator.GeneratedFile, force bool) (result WriteResult, err error) {
	if w.rename == nil {
		w.rename = os.Rename
	}
	if w.removeAll == nil {
		w.removeAll = os.RemoveAll
	}
	if w.remove == nil {
		w.remove = os.Remove
	}

	root, err := CanonicalPublishPath(outputDir)
	if err != nil {
		return WriteResult{}, err
	}
	result = WriteResult{OutputDir: root}
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return result, fmt.Errorf("create output parent directory: %w", err)
	}
	unlock, err := acquireLock(parent, filepath.Base(root), root, w.remove)
	if err != nil {
		return result, err
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil {
			result.Warning = appendWarning(result.Warning, unlockErr.Error())
		}
	}()

	state, err := inspectRoot(root)
	if err != nil {
		return result, err
	}
	if err := validatePublicationState(root, state, force); err != nil {
		return result, err
	}
	result.Action = PlanActionCreate
	if state.exists {
		result.Action = PlanActionReplace
	}
	result.ForceRequired = state.exists
	result.ForceUsed = state.exists && force

	staging, err := os.MkdirTemp(parent, "."+filepath.Base(root)+".microgen-staging-*")
	if err != nil {
		return result, fmt.Errorf("create staging directory: %w", err)
	}
	stagingPublished := false
	defer func() {
		if !stagingPublished {
			_ = w.removeAll(staging)
		}
	}()

	if err := writeFiles(staging, files); err != nil {
		return result, err
	}
	if err := writeManifest(staging, files); err != nil {
		return result, err
	}

	if !state.exists {
		if refreshed, err := inspectRoot(root); err != nil {
			return result, err
		} else if refreshed.exists {
			return result, errors.New("output directory changed while preparing publication; retry the command")
		}
		if err := w.rename(staging, root); err != nil {
			return result, fmt.Errorf("publish generated directory: %w", err)
		}
		stagingPublished = true
		return result, nil
	}
	if refreshed, err := inspectRoot(root); err != nil {
		return result, err
	} else if err := validatePublicationState(root, refreshed, force); err != nil {
		return result, fmt.Errorf("output directory changed while preparing publication: %w", err)
	}

	backup, err := os.MkdirTemp(parent, "."+filepath.Base(root)+".microgen-backup-*")
	if err != nil {
		return result, fmt.Errorf("create backup placeholder: %w", err)
	}
	if err := w.removeAll(backup); err != nil {
		return result, fmt.Errorf("prepare backup path: %w", err)
	}

	if err := w.rename(root, backup); err != nil {
		return result, fmt.Errorf("preserve existing generated directory: %w", err)
	}

	if err := w.rename(staging, root); err != nil {
		restoreErr := w.rename(backup, root)
		if restoreErr != nil {
			return result, fmt.Errorf("publish generated directory: %w; rollback failed: %v; previous generated directory remains at %s; recover by moving that backup path back to %s after inspecting both locations", err, restoreErr, backup, root)
		}
		return result, fmt.Errorf("publish generated directory: %w; previous generated directory restored", err)
	}
	stagingPublished = true
	if err := w.removeAll(backup); err != nil {
		result.Warning = fmt.Sprintf("generated output was published successfully, but cleanup of previous backup failed at %s: %v; inspect and remove that backup manually", backup, err)
	}
	return result, nil
}

type lockMetadata struct {
	Generator    string `json:"generator"`
	Version      int    `json:"version"`
	PID          int    `json:"pid"`
	Hostname     string `json:"hostname"`
	HostID       string `json:"hostId"`
	OwnerToken   string `json:"ownerToken"`
	OutputDir    string `json:"outputDir"`
	CreatedAtUTC string `json:"createdAtUtc"`
}

type lockHandle struct {
	path  string
	token string
}

func acquireLock(parent, base, outputDir string, remove func(string) error) (func() error, error) {
	lockPath := filepath.Join(parent, "."+base+"."+lockFileName)
	handle, file, err := createLockFile(lockPath, outputDir)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			if recovered, recoverErr := recoverStaleLock(lockPath, remove); recoverErr != nil {
				return nil, recoverErr
			} else if recovered {
				handle, file, err = createLockFile(lockPath, outputDir)
				if err != nil {
					return nil, fmt.Errorf("create output publication lock after stale recovery: %w", err)
				}
			} else {
				return nil, fmt.Errorf("output target is locked by another microgen publication: %s", lockPath)
			}
		} else {
			return nil, fmt.Errorf("create output publication lock: %w", err)
		}
	}
	_ = file.Close()
	return func() error {
		metadata, err := readLockMetadata(handle.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("generated output was published, but publication lock at %s could not be verified before cleanup: %w; inspect and remove that lock manually", handle.path, err)
		}
		if metadata.OwnerToken != handle.token {
			return fmt.Errorf("generated output was published, but publication lock ownership changed at %s; leaving lock in place for manual inspection", handle.path)
		}
		if err := remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("generated output was published, but cleanup of publication lock failed at %s: %w; inspect and remove that lock manually", lockPath, err)
		}
		return nil
	}, nil
}

func createLockFile(lockPath, outputDir string) (lockHandle, *os.File, error) {
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return lockHandle{}, nil, err
	}
	hostname, _ := os.Hostname()
	hostID := currentHostIdentity()
	ownerToken, err := randomOwnerToken()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return lockHandle{}, nil, fmt.Errorf("create output publication lock owner token: %w", err)
	}
	metadata := lockMetadata{Generator: manifestGenerator, Version: manifestVersion, PID: os.Getpid(), Hostname: hostname, HostID: hostID, OwnerToken: ownerToken, OutputDir: outputDir, CreatedAtUTC: time.Now().UTC().Format(time.RFC3339Nano)}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return lockHandle{}, nil, fmt.Errorf("write output publication lock metadata: %w", err)
	}
	return lockHandle{path: lockPath, token: ownerToken}, file, nil
}

func recoverStaleLock(lockPath string, remove func(string) error) (bool, error) {
	metadata, err := readLockMetadata(lockPath)
	if err != nil {
		return false, fmt.Errorf("output target has an unreadable publication lock at %s; inspect and remove it manually if no microgen process is active: %w", lockPath, err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, metadata.CreatedAtUTC)
	if err != nil {
		return false, fmt.Errorf("output target has a publication lock with invalid metadata at %s; inspect and remove it manually if no microgen process is active: %w", lockPath, err)
	}
	if time.Since(createdAt) <= lockStaleAfter || processExists(metadata.PID) {
		return false, nil
	}
	hostname, _ := os.Hostname()
	hostID := currentHostIdentity()
	if hostID == "" || metadata.HostID == "" {
		return false, fmt.Errorf("stale output publication lock found at %s for pid %d, but host identity is unavailable; microgen will not remove an unknown-host lock automatically. Confirm no publication is active, then remove %s manually", lockPath, metadata.PID, lockPath)
	}
	if metadata.Hostname != hostname || metadata.HostID != hostID {
		return false, fmt.Errorf("stale output publication lock found at %s for pid %d on host %q, but this host is %q; microgen will not remove a foreign-host lock automatically. Confirm no publication is active on %q, then remove %s manually", lockPath, metadata.PID, metadata.Hostname, hostname, metadata.Hostname, lockPath)
	}
	if err := remove(lockPath); err != nil {
		return false, fmt.Errorf("stale output publication lock found at %s for pid %d, but cleanup failed: %w; inspect and remove that lock manually", lockPath, metadata.PID, err)
	}
	return true, nil
}

func readLockMetadata(lockPath string) (lockMetadata, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return lockMetadata{}, err
	}
	var metadata lockMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return lockMetadata{}, err
	}
	if metadata.Generator != manifestGenerator || metadata.Version != manifestVersion || metadata.PID <= 0 || strings.TrimSpace(metadata.OutputDir) == "" || strings.TrimSpace(metadata.OwnerToken) == "" {
		return lockMetadata{}, errors.New("metadata does not describe a microgen publication lock")
	}
	return metadata, nil
}

func localHostIdentity() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		content, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(content)) != "" {
			sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
			return hex.EncodeToString(sum[:])
		}
	}
	return ""
}

func randomOwnerToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func appendWarning(existing, warning string) string {
	if existing == "" {
		return warning
	}
	return existing + "; " + warning
}

type rootState struct {
	exists bool
	isDir  bool
	owned  bool
}

// CanonicalPublishPath resolves the output path using the same safety contract
// enforced before publishing generated files.
func CanonicalPublishPath(outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", errors.New("output directory is required")
	}
	abs, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	abs = filepath.Clean(abs)
	existing := abs
	var missing []string
	for {
		info, err := os.Lstat(existing)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("refusing output path %s because existing ancestor %s is a symlink", outputDir, existing)
			}
			realExisting, err := filepath.EvalSymlinks(existing)
			if err != nil {
				return "", fmt.Errorf("resolve real output ancestor: %w", err)
			}
			for left, right := 0, len(missing)-1; left < right; left, right = left+1, right-1 {
				missing[left], missing[right] = missing[right], missing[left]
			}
			parts := append([]string{realExisting}, missing...)
			return filepath.Join(parts...), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect output ancestor: %w", err)
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", fmt.Errorf("no existing ancestor found for output directory %s", outputDir)
		}
		missing = append(missing, filepath.Base(existing))
		existing = parent
	}
}

func inspectRoot(root string) (rootState, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return rootState{}, nil
	}
	if err != nil {
		return rootState{}, fmt.Errorf("inspect output directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return rootState{}, fmt.Errorf("refusing output directory %s because it is a symlink", root)
	}
	state := rootState{exists: true, isDir: info.IsDir()}
	if state.isDir {
		manifest, err := readManifest(root)
		state.owned = err == nil && manifest.Generator == manifestGenerator && manifest.Version == manifestVersion
	}
	return state, nil
}

func validatePublicationState(root string, state rootState, force bool) error {
	if !state.exists {
		return nil
	}
	if !state.isDir {
		return fmt.Errorf("output path %s exists and is not a directory", root)
	}
	if !force {
		return fmt.Errorf("refusing to overwrite existing directory %s; pass --force only for verified directories previously generated by microgen", root)
	}
	manifest, err := readManifest(root)
	if err != nil || manifest.Generator != manifestGenerator || manifest.Version != manifestVersion {
		return fmt.Errorf("refusing to force replace %s because it is not owned by microgen", root)
	}
	if err := verifyOwnedTree(root, manifest); err != nil {
		return fmt.Errorf("refusing to force replace %s because generated output drifted: %w", root, err)
	}
	return nil
}

func writeFiles(root string, files []generator.GeneratedFile) error {
	for _, file := range files {
		target, err := safeTarget(root, file.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create output directory for %s: %w", target, err)
		}
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	return nil
}

func writeManifest(root string, files []generator.GeneratedFile) error {
	manifest := Manifest{Generator: manifestGenerator, Version: manifestVersion, Files: make([]ManifestFile, 0, len(files))}
	for _, file := range files {
		cleanPath, err := cleanGeneratedPath(file.Path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(file.Content)
		manifest.Files = append(manifest.Files, ManifestFile{Path: cleanPath, SHA256: hex.EncodeToString(sum[:])})
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("render ownership manifest: %w", err)
	}
	content = append(content, '\n')
	manifestPath := filepath.Join(root, manifestDirName, manifestFileName)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("create ownership manifest directory: %w", err)
	}
	if err := os.WriteFile(manifestPath, content, 0o644); err != nil {
		return fmt.Errorf("write ownership manifest: %w", err)
	}
	return nil
}

func readManifest(root string) (Manifest, error) {
	content, err := os.ReadFile(filepath.Join(root, manifestDirName, manifestFileName))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func hasValidManifest(root string) bool {
	manifest, err := readManifest(root)
	return err == nil && manifest.Generator == manifestGenerator && manifest.Version == manifestVersion
}

func verifyOwnedTree(root string, manifest Manifest) error {
	expected := map[string]string{}
	for _, file := range manifest.Files {
		cleanPath, err := cleanGeneratedPath(file.Path)
		if err != nil {
			return fmt.Errorf("manifest contains unsafe path %q", file.Path)
		}
		if file.SHA256 == "" {
			return fmt.Errorf("manifest entry %s has empty sha256", cleanPath)
		}
		expected[cleanPath] = file.SHA256
	}
	seen := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink found at %s", rel)
		}
		if rel == filepath.ToSlash(filepath.Join(manifestDirName, manifestFileName)) {
			return nil
		}
		if strings.HasPrefix(rel, manifestDirName+"/") {
			return fmt.Errorf("unknown file in manifest directory: %s", rel)
		}
		if entry.IsDir() {
			return nil
		}
		expectedHash, ok := expected[rel]
		if !ok {
			return fmt.Errorf("unknown file found: %s", rel)
		}
		actualHash, err := hashFile(path)
		if err != nil {
			return err
		}
		if actualHash != expectedHash {
			return fmt.Errorf("modified file found: %s", rel)
		}
		seen[rel] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	for path := range expected {
		if _, ok := seen[path]; !ok {
			return fmt.Errorf("manifest file missing from output tree: %s", path)
		}
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func safeTarget(root, relativePath string) (string, error) {
	cleanRelative, err := cleanGeneratedPath(relativePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(cleanRelative)), nil
}

func cleanGeneratedPath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("generated path %q must be relative", relativePath)
	}
	cleanRelative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	if cleanRelative == "." || strings.HasPrefix(cleanRelative, "../") || cleanRelative == ".." {
		return "", fmt.Errorf("generated path %q escapes the output directory", relativePath)
	}
	return cleanRelative, nil
}
