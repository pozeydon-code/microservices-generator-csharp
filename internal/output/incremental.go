package output

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/generator"
)

const manifestPath = manifestDirName + "/" + manifestFileName

type fileChange struct {
	directory    bool
	path, before string                   // An empty hash denotes an absent file.
	next         *generator.GeneratedFile // nil denotes deletion, except for the manifest.
}

func reconcile(root string, files []generator.GeneratedFile, force bool) (Plan, []fileChange, error) {
	plan := Plan{OutputDir: root, Action: PlanActionCreate}
	state, err := inspectRoot(root)
	if err != nil {
		return plan, nil, err
	}
	prior := map[string]string{}
	var manifestHash string
	if state.exists {
		if !state.isDir {
			return plan, nil, fmt.Errorf("output path %s exists and is not a directory", root)
		}
		manifestHash, err = diskHash(root, manifestPath)
		if err != nil {
			return plan, nil, err
		}
		manifest, err := readManifest(root)
		if err != nil || manifest.Generator != manifestGenerator || manifest.Version != manifestVersion {
			return plan, nil, fmt.Errorf("refusing to force replace %s because it is not owned by microgen", root)
		}
		for _, f := range manifest.Files {
			p, err := inventoryPath(f.Path)
			hash, decodeErr := hex.DecodeString(f.SHA256)
			if err != nil || decodeErr != nil || len(hash) != sha256.Size || prior[p] != "" {
				return plan, nil, fmt.Errorf("invalid manifest entry %q", f.Path)
			}
			prior[p] = strings.ToLower(f.SHA256)
		}
		if err := checkPrefixes(prior); err != nil {
			return plan, nil, err
		}
		plan.Action = PlanActionUnchanged
	}
	next := make(map[string]string, len(files))
	byPath := make(map[string]*generator.GeneratedFile, len(files))
	for i := range files {
		p, err := inventoryPath(files[i].Path)
		if err != nil {
			return plan, nil, err
		}
		if next[p] != "" {
			return plan, nil, fmt.Errorf("duplicate generated path %q", p)
		}
		next[p] = fmt.Sprintf("%x", sha256.Sum256(files[i].Content))
		byPath[p] = &files[i]
	}
	if err := checkPrefixes(next); err != nil {
		return plan, nil, err
	}
	paths := make([]string, 0, len(next)+len(prior))
	for p := range next {
		paths = append(paths, p)
	}
	for p := range prior {
		if next[p] == "" {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var deletes, directories, writes []fileChange
	var conflicts []error
	inventoryChanged := false
	for _, p := range paths {
		old, want := prior[p], next[p]
		action := PlanActionUnchanged
		if old != want {
			inventoryChanged = true
			switch {
			case old == "":
				action = PlanActionCreate
			case want != "":
				action = PlanActionReplace
			}
			if state.exists {
				actual, err := diskHash(root, p)
				if old == "" && err != nil {
					var dirs []fileChange
					actual, dirs, err = planPathTransition(root, p, prior, err)
					directories = append(directories, dirs...)
				}
				op := "update"
				if old == "" {
					op = "create"
				}
				if want == "" {
					op = "delete"
				}
				switch {
				case err != nil:
					conflicts = append(conflicts, fmt.Errorf("generation conflict: cannot %s %s: %w", op, p, err))
				case actual != old && !(want == "" && actual == ""):
					reason := "modified file found"
					if old == "" {
						reason = "unknown file found"
					}
					if actual == "" {
						reason = "generated file is missing"
					}
					conflicts = append(conflicts, fmt.Errorf("generation conflict: cannot %s %s: %s", op, p, reason))
				case want != "" || actual != "":
					change := fileChange{path: p, before: actual, next: byPath[p]}
					if want == "" {
						deletes = append(deletes, change)
						plan.DeletedFiles = append(plan.DeletedFiles, p)
					} else {
						writes = append(writes, change)
					}
				}
			}
		}
		if want != "" {
			plan.Files = append(plan.Files, PlannedFile{Path: p, Action: action})
		}
	}
	if len(conflicts) > 0 {
		return plan, nil, errors.Join(conflicts...)
	}
	changes := append(append(deletes, directories...), writes...)
	if state.exists && inventoryChanged {
		plan.Action, plan.ForceRequired, plan.ForceUsed = PlanActionReplace, true, force
		changes = append(changes, fileChange{path: manifestPath, before: manifestHash})
	}
	return plan, changes, nil
}

// Transitions may clear only paths accounted for by the prior inventory.
// Their obsolete files are checked separately by the reconciliation loop.
func planPathTransition(root, path string, prior map[string]string, original error) (string, []fileChange, error) {
	for dir := filepath.ToSlash(filepath.Dir(path)); dir != "."; dir = filepath.ToSlash(filepath.Dir(dir)) {
		if prior[dir] != "" {
			return "", nil, nil
		}
	}
	knownDirs := map[string]bool{}
	for old := range prior {
		if strings.HasPrefix(old, path+"/") {
			for dir := filepath.ToSlash(filepath.Dir(old)); dir != "." && !knownDirs[dir]; dir = filepath.ToSlash(filepath.Dir(dir)) {
				knownDirs[dir] = true
				if dir == path {
					break
				}
			}
		}
	}
	if !knownDirs[path] {
		return "", nil, original
	}
	target := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(target)
	if err != nil || !info.IsDir() {
		return "", nil, original
	}
	var dirs []fileChange
	err = filepath.WalkDir(target, func(abs string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return err
		}
		p := filepath.ToSlash(rel)
		if entry.IsDir() && knownDirs[p] {
			dirs = append(dirs, fileChange{path: p, directory: true})
		} else if !entry.Type().IsRegular() || prior[p] == "" {
			return fmt.Errorf("unknown or unsafe path found: %s", p)
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	// Children sort after their parent and must be removed first.
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].path > dirs[j].path })
	return "", dirs, nil
}

func inventoryPath(path string) (string, error) {
	p, err := cleanGeneratedPath(path)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(strings.Split(p, "/")[0], manifestDirName) {
		return "", fmt.Errorf("reserved generated path %q", path)
	}
	return p, nil
}

func checkPrefixes(paths map[string]string) error {
	for p := range paths {
		for dir := filepath.ToSlash(filepath.Dir(p)); dir != "."; dir = filepath.ToSlash(filepath.Dir(dir)) {
			if paths[dir] != "" {
				return fmt.Errorf("overlapping generated paths %q and %q", dir, p)
			}
		}
	}
	return nil
}

// Only inspect paths we may change; unrelated entries are not ours to manage.
func diskHash(root, path string) (string, error) {
	state, err := inspectRoot(root)
	if err != nil {
		return "", err
	}
	if !state.isDir {
		return "", fmt.Errorf("not a publication directory: %s", root)
	}
	target := root
	parts := strings.Split(path, "/")
	for i, part := range parts {
		target = filepath.Join(target, part)
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink found at %s", target)
		}
		if i+1 == len(parts) && info.Mode().IsRegular() {
			return hashFile(target)
		}
		if i+1 == len(parts) || !info.IsDir() {
			return "", fmt.Errorf("unexpected file type at %s", target)
		}
	}
	return "", nil
}

// Stage new bytes before touching live files. Move only affected originals into
// the journal, and install with a no-clobber link so a new file cannot be lost.
func (w FilesystemWriter) publishChanges(root string, files []generator.GeneratedFile, changes []fileChange) (warning string, err error) {
	parent, base := filepath.Dir(root), filepath.Base(root)
	stage, err := os.MkdirTemp(parent, "."+base+".microgen-staging-*")
	if err != nil {
		return "", err
	}
	defer w.removeAll(stage)
	for _, change := range changes {
		if change.next != nil {
			if err := writeFiles(stage, []generator.GeneratedFile{*change.next}); err != nil {
				return "", err
			}
		}
	}
	if err := writeManifest(stage, files); err != nil {
		return "", err
	}
	backup, err := os.MkdirTemp(parent, "."+base+".microgen-backup-*")
	if err != nil {
		return "", err
	}
	type appliedChange struct {
		dirs                               []string
		removedDir                         os.FileInfo
		path, saved, staged, installedHash string
		moved, installed                   bool
	}
	var applied []appliedChange
	defer func() {
		if err != nil {
			var failures []error
			for i := len(applied) - 1; i >= 0; i-- {
				a := applied[i]
				if a.removedDir != nil {
					if mkdirErr := os.Mkdir(a.path, a.removedDir.Mode().Perm()); mkdirErr != nil {
						failures = append(failures, mkdirErr)
					}
					continue
				}
				if a.installed {
					live, liveErr := os.Lstat(a.path)
					staged, stageErr := os.Stat(a.staged)
					if liveErr != nil || stageErr != nil || !os.SameFile(live, staged) {
						failures = append(failures, fmt.Errorf("cannot roll back changed path %s", a.path))
						continue
					}
					if current, hashErr := hashFile(a.path); hashErr != nil || current != a.installedHash {
						failures = append(failures, fmt.Errorf("cannot roll back edited path %s", a.path))
						continue
					}
					if removeErr := w.remove(a.path); removeErr != nil {
						failures = append(failures, removeErr)
						continue
					}
				}
				for j := len(a.dirs) - 1; j >= 0; j-- {
					_ = w.remove(a.dirs[j])
				}
				if a.moved {
					if restoreErr := w.link(a.saved, a.path); restoreErr != nil {
						failures = append(failures, restoreErr)
					}
				}
			}
			if len(failures) > 0 {
				err = fmt.Errorf("%w; rollback failed: %v; previous files remain at %s; inspect the backup before restoring", err, errors.Join(failures...), backup)
				return
			}
			err = fmt.Errorf("%w; previous files restored", err)
		}
		if cleanupErr := w.removeAll(backup); cleanupErr != nil {
			if err != nil {
				err = errors.Join(err, cleanupErr)
			} else {
				warning = fmt.Sprintf("generated output was published successfully, but cleanup of previous backup failed at %s: %v", backup, cleanupErr)
			}
		}
	}()
	for _, change := range changes {
		live := filepath.Join(root, filepath.FromSlash(change.path))
		if change.directory {
			info, err := os.Lstat(live)
			if err != nil || !info.IsDir() {
				return "", fmt.Errorf("generation conflict: directory %s changed while preparing publication", change.path)
			}
			// Remove only an empty directory; a newly inserted manual file aborts.
			if err := w.remove(live); err != nil {
				return "", fmt.Errorf("clear directory %s: %w", change.path, err)
			}
			applied = append(applied, appliedChange{path: live, removedDir: info})
			continue
		}
		actual, err := diskHash(root, change.path)
		if err != nil {
			return "", fmt.Errorf("recheck %s: %w", change.path, err)
		}
		if actual != change.before {
			return "", fmt.Errorf("generation conflict: %s changed while preparing publication", change.path)
		}
		a := appliedChange{path: live, saved: filepath.Join(backup, filepath.FromSlash(change.path)), staged: filepath.Join(stage, filepath.FromSlash(change.path))}
		applied = append(applied, a)
		entry := &applied[len(applied)-1]
		if actual != "" {
			if err := os.MkdirAll(filepath.Dir(entry.saved), 0o755); err != nil {
				return "", err
			}
			if err := w.rename(live, entry.saved); err != nil {
				return "", fmt.Errorf("preserve %s: %w", change.path, err)
			}
			entry.moved = true
			// Recheck the moved file, not merely the earlier snapshot of its path.
			if movedHash, err := diskHash(backup, change.path); err != nil || movedHash != actual {
				return "", fmt.Errorf("generation conflict: %s changed while preparing publication", change.path)
			}
		}
		if change.next == nil && change.path != manifestPath {
			continue
		}
		dirs, err := createPublishDirs(root, filepath.Dir(live))
		entry.dirs = dirs
		if err != nil {
			return "", err
		}
		entry.installedHash, err = hashFile(entry.staged)
		if err != nil {
			return "", err
		}
		if err := w.link(entry.staged, live); err != nil {
			return "", fmt.Errorf("publish %s: %w", change.path, err)
		}
		entry.installed = true
	}
	return "", nil
}

func createPublishDirs(root, dir string) ([]string, error) {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return nil, err
	}
	var created []string
	target := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		target = filepath.Join(target, part)
		if err := os.Mkdir(target, 0o755); err == nil {
			created = append(created, target)
		} else if !errors.Is(err, os.ErrExist) {
			return created, err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return created, err
		}
		if !info.IsDir() {
			return created, fmt.Errorf("not a publication directory: %s", target)
		}
	}
	return created, nil
}
