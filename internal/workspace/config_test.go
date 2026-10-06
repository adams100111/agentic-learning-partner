package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigSupportsNamedWorkspacesAndLegacyDefault(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")
	data := []byte(`defaultWorkspace: personal
workspaces:
  personal:
    path: ~/state/personal
    provider: git
  offline:
    path: ~/state/offline
    provider: local
`)
	if err := os.WriteFile(path, data, 0o644); err != nil { t.Fatal(err) }
	config, err := LoadConfig(path)
	if err != nil { t.Fatal(err) }
	if config.DefaultWorkspace != "personal" || config.Workspaces["offline"].Provider != "local" {
		t.Fatalf("config = %#v", config)
	}
}

func TestConfigResolveNamedWorkspace(t *testing.T) {
	config := Config{
		DefaultWorkspace: "personal",
		Workspaces: map[string]WorkspaceConfig{
			"personal": {Path: "/tmp/personal", Provider: "git"},
		},
	}
	entry, err := config.Resolve("")
	if err != nil { t.Fatal(err) }
	if entry.Name != "personal" || entry.Provider != "git" {
		t.Fatalf("entry = %#v", entry)
	}
}
