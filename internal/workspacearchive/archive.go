package workspacearchive

import (
	"archive/zip"
	"bytes"
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

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
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

type Validator func(root string) error

func Export(ctx context.Context, source storepkg.Store, destination string, createdAt time.Time) (Manifest, error) {
	identity, err := source.Workspace(ctx)
	if err != nil {
		return Manifest{}, err
	}
	manifestData, err := source.Read(ctx, "workspace.yaml")
	if err != nil {
		return Manifest{}, err
	}
	var workspaceManifest workspace.Manifest
	if err := yaml.Unmarshal(manifestData, &workspaceManifest); err != nil {
		return Manifest{}, fmt.Errorf("parse workspace manifest: %w", err)
	}

	entries, err := source.List(ctx, "")
	if err != nil {
		return Manifest{}, err
	}
	files := map[string][]byte{}
	var metadata []Entry
	for _, item := range entries {
		path := filepath.ToSlash(item.Path)
		if !isCanonicalExportPath(path) {
			continue
		}
		data, err := source.Read(ctx, path)
		if err != nil {
			return Manifest{}, err
		}
		sum := sha256.Sum256(data)
		files[path] = data
		metadata = append(metadata, Entry{Path: path, SHA256: hex.EncodeToString(sum[:]), Size: len(data)})
	}
	sort.Slice(metadata, func(i, j int) bool { return metadata[i].Path < metadata[j].Path })

	manifest := Manifest{
		FormatVersion: FormatVersion,
		WorkspaceID:   identity.ID,
		LearnerID:     identity.LearnerID,
		SchemaVersion: workspaceManifest.SchemaVersion,
		CreatedAt:     createdAt.UTC(),
		Entries:       metadata,
	}
	if err := writeArchive(destination, manifest, files); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Verify(path string, validator Validator) (Verified, error) {
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
		if !safeArchivePath(name) {
			return Verified{}, fmt.Errorf("archive contains unsafe path %q", name)
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
		return Verified{}, errors.New("archive manifest identity/schema is incomplete")
	}

	expected := map[string]Entry{}
	for _, entry := range manifest.Entries {
		if !isCanonicalExportPath(entry.Path) || !safeArchivePath(entry.Path) {
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
			return Verified{}, fmt.Errorf("manifest entry %q is missing from archive", path)
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
		return Verified{}, err
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
		for path, data := range files {
			target := filepath.Join(temp, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return Verified{}, err
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return Verified{}, err
			}
		}
		if err := validator(temp); err != nil {
			return Verified{}, fmt.Errorf("archive canonical state is invalid: %w", err)
		}
	}

	return Verified{Manifest: manifest, Files: files}, nil
}

func Restore(ctx context.Context, archive Verified, destination storepkg.Store, mode RestoreMode) (storepkg.Checkpoint, error) {
	target, err := destination.Workspace(ctx)
	if err != nil {
		return storepkg.Checkpoint{}, err
	}
	switch mode {
	case RestoreRecover:
		if archive.Manifest.WorkspaceID != target.ID {
			return storepkg.Checkpoint{}, fmt.Errorf("recover requires workspaceId %q, target is %q", archive.Manifest.WorkspaceID, target.ID)
		}
		if archive.Manifest.LearnerID != target.LearnerID {
			return storepkg.Checkpoint{}, errors.New("recover learner identity mismatch")
		}
	case RestoreClone:
		if archive.Manifest.LearnerID != target.LearnerID {
			return storepkg.Checkpoint{}, errors.New("clone learner identity mismatch")
		}
	case RestoreMerge:
		if archive.Manifest.LearnerID != target.LearnerID {
			return storepkg.Checkpoint{}, errors.New("merge requires the same learnerId")
		}
	default:
		return storepkg.Checkpoint{}, fmt.Errorf("unsupported restore mode %q", mode)
	}

	revision, err := destination.Revision(ctx)
	if err != nil {
		return storepkg.Checkpoint{}, err
	}
	tx, err := destination.Begin(ctx, revision)
	if err != nil {
		return storepkg.Checkpoint{}, err
	}
	defer tx.Rollback()

	files := cloneFiles(archive.Files)
	if mode == RestoreClone {
		manifest := workspace.Manifest{
			SchemaVersion: workspace.CurrentSchemaVersion,
			WorkspaceID: target.ID,
			LearnerID: target.LearnerID,
		}
		data, err := yaml.Marshal(manifest)
		if err != nil {
			return storepkg.Checkpoint{}, err
		}
		files["workspace.yaml"] = data
	}

	if mode == RestoreMerge {
		files, err = mergeIntoDestination(ctx, destination, files)
		if err != nil {
			return storepkg.Checkpoint{}, err
		}
	} else {
		existing, err := destination.List(ctx, "")
		if err != nil {
			return storepkg.Checkpoint{}, err
		}
		for _, item := range existing {
			path := filepath.ToSlash(item.Path)
			if isCanonicalExportPath(path) {
				if _, keep := files[path]; !keep {
					if err := tx.Delete(path); err != nil {
						return storepkg.Checkpoint{}, err
					}
				}
			}
		}
	}
	for path, data := range files {
		if !isCanonicalExportPath(path) {
			continue
		}
		if err := tx.Put(path, data); err != nil {
			return storepkg.Checkpoint{}, err
		}
	}
	return tx.Commit(ctx, "state: restore ALP workspace")
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
	closeWithError := func(current error) error {
		if err := writer.Close(); current == nil && err != nil {
			current = err
		}
		if err := file.Close(); current == nil && err != nil {
			current = err
		}
		return current
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return closeWithError(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeZipEntry(writer, "manifest.json", manifestBytes, manifest.CreatedAt); err != nil {
		return closeWithError(err)
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := writeZipEntry(writer, path, files[path], manifest.CreatedAt); err != nil {
			return closeWithError(err)
		}
	}
	return closeWithError(nil)
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

func isCanonicalExportPath(path string) bool {
	path = filepath.ToSlash(path)
	if path == "workspace.yaml" {
		return true
	}
	for _, prefix := range []string{"profile/", "personas/", "evidence/", "assessments/", "sessions/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func safeArchivePath(path string) bool {
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && clean != ".." && !strings.HasPrefix(clean, "../")
}

func mergeIntoDestination(ctx context.Context, destination storepkg.Store, incoming map[string][]byte) (map[string][]byte, error) {
	result := map[string][]byte{}
	existing, err := destination.List(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, item := range existing {
		path := filepath.ToSlash(item.Path)
		if !isCanonicalExportPath(path) {
			continue
		}
		data, err := destination.Read(ctx, path)
		if err != nil {
			return nil, err
		}
		result[path] = data
	}
	for path, data := range incoming {
		current, exists := result[path]
		if !exists || bytes.Equal(current, data) {
			result[path] = data
			continue
		}
		if isAppendOnlyPath(path) {
			return nil, fmt.Errorf("append-only identity collision at %s", path)
		}
		if path == "workspace.yaml" {
			continue
		}
		merged, ok, err := mergeStructured(current, data)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("semantic restore conflict at %s", path)
		}
		result[path] = merged
	}
	return result, nil
}

func mergeStructured(existing, incoming []byte) ([]byte, bool, error) {
	var left, right map[string]any
	if err := yaml.Unmarshal(existing, &left); err != nil {
		return nil, false, nil
	}
	if err := yaml.Unmarshal(incoming, &right); err != nil {
		return nil, false, nil
	}
	merged, ok := mergeMaps(left, right)
	if !ok {
		return nil, false, nil
	}
	data, err := yaml.Marshal(merged)
	return data, err == nil, err
}

func mergeMaps(left, right map[string]any) (map[string]any, bool) {
	result := map[string]any{}
	for key, value := range left {
		result[key] = value
	}
	for key, incoming := range right {
		existing, exists := result[key]
		if !exists {
			result[key] = incoming
			continue
		}
		if fmt.Sprintf("%#v", existing) == fmt.Sprintf("%#v", incoming) {
			continue
		}
		leftMap, lok := existing.(map[string]any)
		rightMap, rok := incoming.(map[string]any)
		if lok && rok {
			merged, ok := mergeMaps(leftMap, rightMap)
			if !ok {
				return nil, false
			}
			result[key] = merged
			continue
		}
		return nil, false
	}
	return result, true
}

func isAppendOnlyPath(path string) bool {
	for _, prefix := range []string{"evidence/", "assessments/", "sessions/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func cloneFiles(files map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(files))
	for path, data := range files {
		result[path] = append([]byte(nil), data...)
	}
	return result
}
