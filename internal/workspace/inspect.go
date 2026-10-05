package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Info struct {
	Path     string
	Revision string
}

func Inspect(path string) (Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Info{}, fmt.Errorf("inspect workspace %s: %w", path, err)
	}
	if !info.IsDir() {
		return Info{}, fmt.Errorf("workspace %s is not a directory", path)
	}

	root, err := git(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return Info{}, fmt.Errorf("workspace %s must be inside a Git repository: %w", path, err)
	}
	absoluteRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return Info{}, fmt.Errorf("resolve Git root: %w", err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return Info{}, fmt.Errorf("resolve workspace path: %w", err)
	}
	if filepath.Clean(absoluteRoot) != filepath.Clean(absolutePath) {
		return Info{}, fmt.Errorf("workspace %s must be the Git repository root (%s)", absolutePath, absoluteRoot)
	}

	revision, err := git(path, "rev-parse", "HEAD")
	if err != nil {
		return Info{}, fmt.Errorf("workspace Git repository has no committed revision: %w", err)
	}
	return Info{Path: absolutePath, Revision: strings.TrimSpace(revision)}, nil
}

func git(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}
