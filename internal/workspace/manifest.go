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
	WorkspaceID   string `yaml:"workspaceId" json:"workspaceId"`
	LearnerID     string `yaml:"learnerId" json:"learnerId"`
}

func ReadManifest(root string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "workspace.yaml"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read workspace manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse workspace manifest: %w", err)
	}
	return manifest, nil
}

func WriteManifest(root string, manifest Manifest) error {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal workspace manifest: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create workspace root: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), data, 0o644); err != nil {
		return fmt.Errorf("write workspace manifest: %w", err)
	}
	return nil
}

func NewWorkspaceID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate workspace id: %w", err)
	}
	return "ws_" + hex.EncodeToString(data[:]), nil
}
