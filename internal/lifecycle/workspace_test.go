package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestManagerInitializesListsAndSwitchesLocalWorkspaces(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	manager := New(configPath, validator)

	first, err := manager.Init("personal", "local", filepath.Join(root, "personal"), "learner", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Provider != "local" || first.WorkspaceID == "" || !first.Default {
		t.Fatalf("first status = %#v", first)
	}
	if containsCapability(first.Capabilities, storepkg.CapabilitySync) {
		t.Fatal("local workspace must not advertise sync")
	}

	second, err := manager.Init("research", "local", filepath.Join(root, "research"), "learner", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !second.Default {
		t.Fatal("new init is expected to become default")
	}

	if err := manager.Use("personal"); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status("")
	if err != nil {
		t.Fatal(err)
	}
	if status.Name != "personal" || !status.Default {
		t.Fatalf("default status = %#v", status)
	}

	items, err := manager.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Name != "personal" || items[1].Name != "research" {
		t.Fatalf("items = %#v", items)
	}
}

func TestManagerRejectsDuplicateWorkspaceName(t *testing.T) {
	root := t.TempDir()
	manager := New(filepath.Join(root, "config.yaml"), nil)
	if _, err := manager.Init("personal", "local", filepath.Join(root, "first"), "learner", "", ""); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Init("personal", "local", filepath.Join(root, "second"), "learner", "", "")
	if err == nil {
		t.Fatal("expected duplicate workspace name error")
	}
}

func TestMachineConfigDoesNotLiveInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "machine", "config.yaml")
	workspacePath := filepath.Join(root, "workspace")
	manager := New(configPath, nil)
	if _, err := manager.Init("personal", "local", workspacePath, "learner", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("machine config leaked into workspace: %v", err)
	}
}

func containsCapability(values []storepkg.Capability, target storepkg.Capability) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
