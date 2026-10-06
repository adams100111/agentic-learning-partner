package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestWorkspaceV1ToV2AssignsStableWorkspaceIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 1\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := NewWorkspaceMigrator(nil)
	plan, err := m.Plan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].From != 1 || plan.Steps[0].To != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	step := m.Steps[1]
	if err := step.Apply(root); err != nil {
		t.Fatal(err)
	}
	if err := setWorkspaceSchemaVersion(root, 2); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "workspace.yaml"))
	var manifest struct {
		SchemaVersion int    `yaml:"schemaVersion"`
		WorkspaceID   string `yaml:"workspaceId"`
		LearnerID     string `yaml:"learnerId"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || manifest.WorkspaceID == "" || manifest.LearnerID != "learner" {
		t.Fatalf("manifest = %#v", manifest)
	}
}
