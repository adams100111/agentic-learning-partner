package workspacearchive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/store/gitstore"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

func ConvertToGit(
	ctx context.Context,
	source storepkg.Store,
	destination, branch, remote string,
	validator gitstore.Validator,
) (*gitstore.Store, error) {
	manifestBytes, err := source.Read(ctx, "workspace.yaml")
	if err != nil {
		return nil, err
	}
	var manifest workspace.Manifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("parse source workspace manifest: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "alp-convert-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	archivePath := filepath.Join(tempDir, "workspace.alp")
	if _, err := Export(ctx, source, archivePath, time.Unix(0, 0).UTC()); err != nil {
		return nil, err
	}
	verified, err := Verify(archivePath, nil)
	if err != nil {
		return nil, err
	}

	target, err := gitstore.InitializeWithManifest(destination, manifest, branch, validator)
	if err != nil {
		return nil, err
	}
	if remote != "" {
		if err := target.ConfigureRemote("origin", remote); err != nil {
			return nil, err
		}
	}
	if _, err := Restore(ctx, verified, target, RestoreRecover); err != nil {
		return nil, err
	}
	return target, nil
}
