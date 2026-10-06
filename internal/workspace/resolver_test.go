package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolverPrecedence(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project", "nested")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	projectWorkspace := filepath.Join(root, "project-workspace")
	if err := os.WriteFile(filepath.Join(root, "project", ".alp.yaml"), []byte("workspace: ../project-workspace\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := Resolver{
		Getenv: func(key string) string {
			if key == EnvWorkspace {
				return filepath.Join(root, "env-workspace")
			}
			return ""
		},
		HomeDir: func() (string, error) { return filepath.Join(root, "home"), nil },
	}

	explicit := filepath.Join(root, "explicit")
	got, err := resolver.Resolve(explicit, project)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != explicit || got.Source != "explicit --workspace" {
		t.Fatalf("explicit resolution = %#v", got)
	}

	got, err = resolver.Resolve("", project)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != projectWorkspace || got.Source != "project .alp.yaml" {
		t.Fatalf("project resolution = %#v", got)
	}
}

func TestResolverUsesEnvironmentThenUserConfig(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".config", "alp"), 0o755); err != nil {
		t.Fatal(err)
	}
	userWorkspace := filepath.Join(root, "user-workspace")
	if err := os.WriteFile(filepath.Join(home, ".config", "alp", "config.yaml"), []byte("workspace: "+userWorkspace+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	envWorkspace := filepath.Join(root, "env-workspace")
	resolver := Resolver{
		Getenv: func(string) string { return envWorkspace },
		HomeDir: func() (string, error) { return home, nil },
	}
	got, err := resolver.Resolve("", root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != envWorkspace || got.Source != EnvWorkspace {
		t.Fatalf("environment resolution = %#v", got)
	}

	resolver.Getenv = func(string) string { return "" }
	got, err = resolver.Resolve("", root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != userWorkspace || got.Source != "user config" {
		t.Fatalf("user config resolution = %#v", got)
	}
}

func TestResolverUsesNamedWorkspace(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configDir := filepath.Join(home, ".config", "alp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	workspacePath := filepath.Join(root, "personal")
	config := `defaultWorkspace: personal
workspaces:
  personal:
    provider: git
    path: ` + workspacePath + `
    syncMode: session
    remote: origin
    branch: main
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := Resolver{
		Getenv: func(string) string { return "" },
		HomeDir: func() (string, error) { return home, nil },
	}
	got, err := resolver.Resolve("", root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "personal" || got.Path != workspacePath || got.Provider.Type != "git" {
		t.Fatalf("named resolution = %#v", got)
	}
}
