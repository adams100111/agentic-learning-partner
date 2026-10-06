package workspace

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

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
