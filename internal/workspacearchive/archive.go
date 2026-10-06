package workspacearchive

import (
	"archive/zip"
	"context"
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
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

const FormatVersion = 1

type Entry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

type Manifest struct {
	FormatVersion int       `json:"formatVersion"`
	WorkspaceID   string    `json:"workspaceId"`
	LearnerID     string    `json:"learnerId"`
	SchemaVersion int       `json:"schemaVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	Entries       []Entry   `json:"entries"`
}

type Verified struct {
	Manifest Manifest
	Files    map[string][]byte
}

type RestoreMode string

const (
	RestoreRecover RestoreMode = "recover"
	RestoreClone   RestoreMode = "clone"
	RestoreMerge   RestoreMode = "merge"
)

type RestoreOptions struct {
	Mode       RestoreMode
	RuntimeDir string
	Rebuild    func(root string) error
}

func Export(ctx context.Context, source store.Store, destination string, createdAt time.Time) (Manifest, error) {
	if source == nil {
		return Manifest{}, errors.New("source store is required")
	}
	manifestData, err := source.Read(ctx, "workspace.yaml")
	if err != nil {
		return Manifest{}, err
	}
	var workspaceManifest workspace.Manifest
	if err := yaml.Unmarshal(manifestData, &workspaceManifest); err != nil {
		return Manifest{}, fmt.Errorf("parse workspace manifest: %w", err)
	}
	files, err := canonicalSnapshot(source.Root())
	if err != nil {
		return Manifest{}, err
	}
	entries := make([]Entry, 0, len(files))
	for path, data := range files {
		sum := sha256.Sum256(data)
		entries = append(entries, Entry{Path: path, SHA256: hex.EncodeToString(sum[:]), Size: len(data)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	manifest := Manifest{
		FormatVersion: FormatVersion,
		WorkspaceID:   workspaceManifest.WorkspaceID,
		LearnerID:     workspaceManifest.LearnerID,
		SchemaVersion: workspaceManifest.SchemaVersion,
		CreatedAt:     createdAt.UTC(),
		Entries:       entries,
	}
	if err := writeArchive(destination, manifest, files); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Verify(path string, validator *workspace.Validator) (Verified, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Verified{}, fmt.Errorf("open ALP archive: %w", err)
	}
	defer reader.Close()

	files := map[string][]byte{}
	var manifest Manifest
	foundManifest := false
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if name == "manifest.json" {
			if foundManifest {
				return Verified{}, errors.New("archive contains duplicate manifest.json")
			}
			foundManifest = true
			data, err := readZipFile(file)
			if err != nil {
				return Verified{}, err
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Verified{}, fmt.Errorf("parse archive manifest: %w", err)
			}
			continue
		}
		if !safeArchivePath(name) || !store.IsCanonicalRevisionPath(name) {
			return Verified{}, fmt.Errorf("archive contains invalid canonical path %q", name)
		}
		if _, duplicate := files[name]; duplicate {
			return Verified{}, fmt.Errorf("archive contains duplicate entry %q", name)
		}
		data, err := readZipFile(file)
		if err != nil {
			return Verified{}, err
		}
		files[name] = data
	}
	if !foundManifest {
		return Verified{}, errors.New("archive manifest.json is required")
	}
	if manifest.FormatVersion != FormatVersion {
		return Verified{}, fmt.Errorf("unsupported archive format version %d", manifest.FormatVersion)
	}
	if manifest.WorkspaceID == "" || manifest.LearnerID == "" || manifest.SchemaVersion <= 0 {
		return Verified{}, errors.New("archive identity/schema is incomplete")
	}

	expected := map[string]Entry{}
	for _, entry := range manifest.Entries {
		if !safeArchivePath(entry.Path) || !store.IsCanonicalRevisionPath(entry.Path) {
			return Verified{}, fmt.Errorf("manifest contains invalid canonical path %q", entry.Path)
		}
		if _, duplicate := expected[entry.Path]; duplicate {
			return Verified{}, fmt.Errorf("manifest contains duplicate entry %q", entry.Path)
		}
		expected[entry.Path] = entry
	}
	if len(expected) != len(files) {
		return Verified{}, fmt.Errorf("archive file count %d does not match manifest %d", len(files), len(expected))
	}
	for path, entry := range expected {
		data, ok := files[path]
		if !ok {
			return Verified{}, fmt.Errorf("manifest entry %q is missing", path)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != entry.SHA256 || len(data) != entry.Size {
			return Verified{}, fmt.Errorf("archive entry %q failed integrity verification", path)
		}
	}

	workspaceData, ok := files["workspace.yaml"]
	if !ok {
		return Verified{}, errors.New("archive workspace.yaml is required")
	}
	var workspaceManifest workspace.Manifest
	if err := yaml.Unmarshal(workspaceData, &workspaceManifest); err != nil {
		return Verified{}, fmt.Errorf("parse archived workspace manifest: %w", err)
	}
	if workspaceManifest.WorkspaceID != manifest.WorkspaceID ||
		workspaceManifest.LearnerID != manifest.LearnerID ||
		workspaceManifest.SchemaVersion != manifest.SchemaVersion {
		return Verified{}, errors.New("archive manifest and workspace identity disagree")
	}

	if validator != nil {
		temp, err := os.MkdirTemp("", "alp-archive-verify-*")
		if err != nil {
			return Verified{}, err
		}
		defer os.RemoveAll(temp)
		if err := writeSnapshot(temp, files); err != nil {
			return Verified{}, err
		}
		if issues := validator.ValidateWorkspace(temp); len(issues) > 0 {
			return Verified{}, fmt.Errorf("archive canonical state is invalid: %s", issues[0].Error())
		}
	}
	return Verified{Manifest: manifest, Files: cloneFiles(files)}, nil
}

func Restore(ctx context.Context, archive Verified, destination store.Store, validator *workspace.Validator, options RestoreOptions) (store.Revision, error) {
	if destination == nil || validator == nil {
		return "", errors.New("destination store and validator are required")
	}
	if options.RuntimeDir == "" {
		return "", errors.New("restore runtime directory is required")
	}
	targetData, err := destination.Read(ctx, "workspace.yaml")
	if err != nil {
		return "", err
	}
	var targetManifest workspace.Manifest
	if err := yaml.Unmarshal(targetData, &targetManifest); err != nil {
		return "", err
	}

	switch options.Mode {
	case RestoreRecover:
		if archive.Manifest.WorkspaceID != targetManifest.WorkspaceID || archive.Manifest.LearnerID != targetManifest.LearnerID {
			return "", errors.New("recover requires the same workspaceId and learnerId")
		}
	case RestoreClone:
		if archive.Manifest.LearnerID != targetManifest.LearnerID {
			return "", errors.New("clone requires the same learnerId")
		}
	case RestoreMerge:
		if archive.Manifest.LearnerID != targetManifest.LearnerID {
			return "", errors.New("merge requires the same learnerId")
		}
	default:
		return "", fmt.Errorf("unsupported restore mode %q", options.Mode)
	}

	incoming := cloneFiles(archive.Files)
	if options.Mode == RestoreClone || options.Mode == RestoreMerge {
		incoming["workspace.yaml"] = targetData
	}
	if options.Mode == RestoreMerge {
		current, err := canonicalSnapshot(destination.Root())
		if err != nil {
			return "", err
		}
		delete(current, "workspace.yaml")
		mergeIncoming := cloneFiles(incoming)
		delete(mergeIncoming, "workspace.yaml")
		merged, err := store.ReconcileCanonical(map[string][]byte{}, current, mergeIncoming)
		if err != nil {
			return "", err
		}
		merged["workspace.yaml"] = targetData
		incoming = merged
	}

	revision, err := destination.Revision(ctx)
	if err != nil {
		return "", err
	}
	manifest := targetManifest
	coordinator := store.NewCoordinator(destination, validator, options.RuntimeDir)
	tx, err := coordinator.Begin(ctx, "restore-"+manifest.WorkspaceID, revision)
	if err != nil {
		return "", err
	}

	current, err := canonicalSnapshot(destination.Root())
	if err != nil {
		return "", err
	}
	for path := range current {
		if _, keep := incoming[path]; !keep {
			if err := tx.Delete(path); err != nil {
				return "", err
			}
		}
	}
	paths := make([]string, 0, len(incoming))
	for path := range incoming {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := tx.Put(path, incoming[path]); err != nil {
			return "", err
		}
	}
	if err := os.RemoveAll(filepath.Join(tx.StageRoot(), "state")); err != nil {
		return "", fmt.Errorf("discard stale derived state before restore rebuild: %w", err)
	}
	if options.Rebuild != nil {
		if err := options.Rebuild(tx.StageRoot()); err != nil {
			return "", fmt.Errorf("rebuild restored workspace: %w", err)
		}
	}
	return tx.CheckpointWithMessage(ctx, false, "alp: restore workspace")
}

func canonicalSnapshot(root string) (map[string][]byte, error) {
	result := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".alp-") {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !store.IsCanonicalRevisionPath(relative) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[relative] = data
		return nil
	})
	return result, err
}

func writeArchive(destination string, manifest Manifest, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	closeAll := func(current error) error {
		if err := writer.Close(); current == nil && err != nil {
			current = err
		}
		if err := file.Close(); current == nil && err != nil {
			current = err
		}
		return current
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return closeAll(err)
	}
	manifestData = append(manifestData, '\n')
	if err := writeZipEntry(writer, "manifest.json", manifestData, manifest.CreatedAt); err != nil {
		return closeAll(err)
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := writeZipEntry(writer, path, files[path], manifest.CreatedAt); err != nil {
			return closeAll(err)
		}
	}
	return closeAll(nil)
}

func writeZipEntry(writer *zip.Writer, path string, data []byte, timestamp time.Time) error {
	header := &zip.FileHeader{Name: path, Method: zip.Deflate}
	header.SetModTime(timestamp.UTC())
	header.SetMode(0o644)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func safeArchivePath(path string) bool {
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && clean != ".." && !strings.HasPrefix(clean, "../")
}

func writeSnapshot(root string, files map[string][]byte) error {
	for path, data := range files {
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func cloneFiles(files map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(files))
	for path, data := range files {
		result[path] = append([]byte(nil), data...)
	}
	return result
}
