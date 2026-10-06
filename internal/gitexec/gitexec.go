// Package gitexec runs git against an explicit repository, isolated from
// repository-selecting environment variables inherited from the caller.
//
// Git hooks, `git -C` wrappers, and some editors export variables such as
// GIT_DIR and GIT_INDEX_FILE. A child git process that inherits them targets
// that repository instead of the directory ALP passes with -C, which can
// corrupt an unrelated repository. Every git subprocess ALP starts goes
// through this package.
package gitexec

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// localEnvVars is the output of `git rev-parse --local-env-vars` (git 2.55):
// the variables git itself clears when it switches to another repository.
var localEnvVars = map[string]struct{}{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
	"GIT_CONFIG":                       {},
	"GIT_CONFIG_PARAMETERS":            {},
	"GIT_CONFIG_COUNT":                 {},
	"GIT_OBJECT_DIRECTORY":             {},
	"GIT_DIR":                          {},
	"GIT_WORK_TREE":                    {},
	"GIT_IMPLICIT_WORK_TREE":           {},
	"GIT_GRAFT_FILE":                   {},
	"GIT_INDEX_FILE":                   {},
	"GIT_NO_REPLACE_OBJECTS":           {},
	"GIT_REPLACE_REF_BASE":             {},
	"GIT_PREFIX":                       {},
	"GIT_SHALLOW_FILE":                 {},
	"GIT_COMMON_DIR":                   {},
}

// Env returns the current environment without repository-local git variables.
// GIT_CONFIG_KEY_<n>/GIT_CONFIG_VALUE_<n> are dropped with GIT_CONFIG_COUNT.
func Env() []string {
	environ := os.Environ()
	result := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if _, local := localEnvVars[name]; local {
			continue
		}
		if strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_") {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// Command returns a git command run in dir (via -C when dir is non-empty)
// with an isolated environment.
func Command(ctx context.Context, dir string, args ...string) *exec.Cmd {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	command := exec.CommandContext(ctx, "git", args...)
	command.Env = Env()
	return command
}
