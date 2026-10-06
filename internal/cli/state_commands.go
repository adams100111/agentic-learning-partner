package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/lifecycle"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

func (a App) appendEvidence(explicitWorkspace, file string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(a.ErrOut, "read evidence input: %v\n", err)
		return 1
	}
	var record state.Evidence
	if err := yaml.Unmarshal(data, &record); err != nil {
		fmt.Fprintf(a.ErrOut, "parse evidence input: %v\n", err)
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	active, err := openStateStore(info.Path, validator)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	revision, err := active.Revision(context.Background())
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	coordinator, tx, err := beginOneShotMutation(active, validator, "evidence", revision)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	_ = coordinator
	engine := state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(revision), nil },
	}
	record, err = engine.AppendEvidence(string(revision), record)
	if err != nil {
		_ = tx.Rollback()
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: append evidence "+record.ID); err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintln(a.Out, record.ID)
	return 0
}

func (a App) appendAssessment(explicitWorkspace, file string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(a.ErrOut, "read assessment input: %v\n", err)
		return 1
	}
	var record state.Assessment
	if err := yaml.Unmarshal(data, &record); err != nil {
		fmt.Fprintf(a.ErrOut, "parse assessment input: %v\n", err)
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	active, err := openStateStore(info.Path, validator)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	revision, err := active.Revision(context.Background())
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	_, tx, err := beginOneShotMutation(active, validator, "assessment", revision)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	engine := state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(revision), nil },
	}
	record, err = engine.AppendAssessment(string(revision), record)
	if err != nil {
		_ = tx.Rollback()
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: append assessment "+record.ID); err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintln(a.Out, record.ID)
	return 0
}

func (a App) rebuildState(explicitWorkspace string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	active, err := openStateStore(info.Path, validator)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	revision, err := active.Revision(context.Background())
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	_, tx, err := beginOneShotMutation(active, validator, "rebuild", revision)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	projection, err := (state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(revision), nil },
	}).RebuildProjection()
	if err != nil {
		_ = tx.Rollback()
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: rebuild learner state"); err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintf(a.Out, "rebuilt %d competencies\n", len(projection.Competencies))
	return 0
}

func openStateStore(root string, validator *workspace.Validator) (storepkg.Store, error) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return storepkg.OpenGit(root, validator)
	}
	return storepkg.OpenLocal(root, validator)
}

func beginOneShotMutation(active storepkg.Store, validator *workspace.Validator, kind string, revision storepkg.Revision) (*storepkg.Coordinator, *storepkg.Transaction, error) {
	runtimeDir, err := lifecycle.DefaultRuntimeDir()
	if err != nil {
		return nil, nil, err
	}
	coordinator := storepkg.NewCoordinator(active, validator, runtimeDir)
	sessionID := fmt.Sprintf("cli-%s-%d", kind, time.Now().UTC().UnixNano())
	tx, err := coordinator.Begin(context.Background(), sessionID, revision)
	if err != nil {
		return nil, nil, err
	}
	return coordinator, tx, nil
}
