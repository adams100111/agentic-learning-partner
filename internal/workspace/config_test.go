package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUserConfigNamedWorkspacesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	want := UserConfig{
		DefaultWorkspace: "personal",
		Workspaces: map[string]ProviderConfig{
			"personal": {
				Type: "git", Path: "/tmp/personal", SyncMode: "session", Remote: "origin", Branch: "main",
			},
			"local": {Type: "local", Path: "/tmp/local"},
		},
	}
	if err := WriteUserConfig(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadUserConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config permissions = %o", info.Mode().Perm())
	}
}

func TestLegacySingleWorkspaceBecomesDefaultGitProvider(t *testing.T) {
	config := UserConfig{Workspace: "/tmp/legacy"}
	name, workspace, ok := config.Default()
	if !ok || name != "default" || workspace.Type != "git" || workspace.Path != "/tmp/legacy" {
		t.Fatalf("legacy default = %q %#v %v", name, workspace, ok)
	}
}
