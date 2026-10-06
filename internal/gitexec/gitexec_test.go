package gitexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandIgnoresInheritedRepositoryVariables(t *testing.T) {
	target := t.TempDir()
	decoy := t.TempDir()
	for _, dir := range []string{target, decoy} {
		if output, err := Command(context.Background(), "", "init", "-q", dir).CombinedOutput(); err != nil {
			t.Fatalf("git init %s: %s: %v", dir, output, err)
		}
	}
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(decoy, ".git", "index"))

	output, err := Command(context.Background(), target, "rev-parse", "--absolute-git-dir").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse: %s: %v", output, err)
	}
	got, _ := filepath.EvalSymlinks(strings.TrimSpace(string(output)))
	want, _ := filepath.EvalSymlinks(filepath.Join(target, ".git"))
	if got != want {
		t.Fatalf("git dir = %s, want %s (inherited GIT_DIR leaked)", got, want)
	}
}

func TestEnvKeepsUnrelatedVariables(t *testing.T) {
	t.Setenv("GIT_DIR", "/elsewhere")
	t.Setenv("GIT_CONFIG_KEY_0", "user.name")
	t.Setenv("GIT_AUTHOR_NAME", "Kept")
	t.Setenv("ALP_PROBE", "kept")
	env := strings.Join(Env(), "\n")
	for _, banned := range []string{"GIT_DIR=", "GIT_CONFIG_KEY_0="} {
		if strings.Contains(env, banned) {
			t.Fatalf("Env() kept %s", banned)
		}
	}
	for _, kept := range []string{"GIT_AUTHOR_NAME=Kept", "ALP_PROBE=kept", "PATH=" + os.Getenv("PATH")} {
		if !strings.Contains(env, kept) {
			t.Fatalf("Env() dropped %s", kept)
		}
	}
}
