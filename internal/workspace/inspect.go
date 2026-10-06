package workspace

import (
	"context"
	"fmt"
	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"os"
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
	if !samePath(absoluteRoot, absolutePath) {
		return Info{}, fmt.Errorf("workspace %s must be the Git repository root (%s)", absolutePath, absoluteRoot)
	}

	revision, err := git(path, "rev-parse", "HEAD")
	if err != nil {
		return Info{}, fmt.Errorf("workspace Git repository has no committed revision: %w", err)
	}
	return Info{Path: absolutePath, Revision: strings.TrimSpace(revision)}, nil
}

func git(dir string, args ...string) (string, error) {
	command := gitexec.Command(context.Background(), dir, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// samePath compares paths after resolving symlinks, so macOS /var and
// /private/var refer to the same directory.
func samePath(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
