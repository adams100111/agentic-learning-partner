package workspacearchive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type ConvertToGitOptions struct {
	Destination string
	Branch      string
	Remote      string
	RuntimeDir  string
	Rebuild     func(root string) error
}

func ConvertToGit(ctx context.Context, source store.Store, validator *workspace.Validator, options ConvertToGitOptions) (*store.Git, error) {
	if source == nil || validator == nil {
		return nil, errors.New("source store and validator are required")
	}
	if options.Destination == "" || options.RuntimeDir == "" {
		return nil, errors.New("destination and runtime directory are required")
	}
	if info, err := os.Stat(options.Destination); err == nil {
		if !info.IsDir() {
			return nil, errors.New("Git Store destination must be a directory")
		}
		entries, readErr := os.ReadDir(options.Destination)
		if readErr != nil {
			return nil, readErr
		}
		if len(entries) != 0 {
			return nil, errors.New("Git Store destination must be empty")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	} else if err := os.MkdirAll(options.Destination, 0o755); err != nil {
		return nil, err
	}

	temp, err := os.MkdirTemp("", "alp-convert-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	archivePath := filepath.Join(temp, "workspace.alp")
	if _, err := Export(ctx, source, archivePath, time.Unix(0, 0).UTC()); err != nil {
		return nil, err
	}
	verified, err := Verify(archivePath, validator)
	if err != nil {
		return nil, err
	}

	workspaceData := verified.Files["workspace.yaml"]
	if err := os.WriteFile(filepath.Join(options.Destination, "workspace.yaml"), workspaceData, 0o644); err != nil {
		return nil, err
	}
	target, err := store.InitializeGit(ctx, options.Destination, options.Branch, options.Remote, validator)
	if err != nil {
		return nil, err
	}
	if _, err := Restore(ctx, verified, target, validator, RestoreOptions{
		Mode: RestoreRecover, RuntimeDir: options.RuntimeDir, Rebuild: options.Rebuild,
	}); err != nil {
		return nil, err
	}
	return target, nil
}
