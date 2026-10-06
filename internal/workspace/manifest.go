package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const CurrentSchemaVersion = 2

type Manifest struct {
	SchemaVersion int    `yaml:"schemaVersion" json:"schemaVersion"`
	WorkspaceID   string `yaml:"workspaceId,omitempty" json:"workspaceId,omitempty"`
	LearnerID     string `yaml:"learnerId" json:"learnerId"`
}

func ReadManifest(root string) (Manifest, error) {
	data, err := os.ReadFile(root + string(os.PathSeparator) + "workspace.yaml")
	if err != nil {
		return Manifest{}, fmt.Errorf("read workspace manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse workspace manifest: %w", err)
	}
	if manifest.LearnerID == "" {
		return Manifest{}, fmt.Errorf("workspace learnerId is required")
	}
	if manifest.SchemaVersion >= 2 && manifest.WorkspaceID == "" {
		return Manifest{}, fmt.Errorf("workspaceId is required for schema v%d", manifest.SchemaVersion)
	}
	return manifest, nil
}

func NewWorkspaceID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate workspace id: %w", err)
	}
	return "ws_" + hex.EncodeToString(raw[:]), nil
}

func WriteManifest(root string, manifest Manifest) error {
	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = CurrentSchemaVersion
	}
	if manifest.WorkspaceID == "" || manifest.LearnerID == "" {
		return fmt.Errorf("workspaceId and learnerId are required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal workspace manifest: %w", err)
	}
	path := filepath.Join(root, "workspace.yaml")
	return os.WriteFile(path, data, 0o644)
}
