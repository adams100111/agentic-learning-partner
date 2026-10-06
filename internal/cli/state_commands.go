package cli

import (
	"context"
	"errors"
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
	tx, expected, activeSession, err := mutationTransaction(active, validator, "evidence")
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	engine := state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(expected), nil },
	}
	record, err = engine.AppendEvidence(string(expected), record)
	if err != nil {
		if !activeSession { _ = tx.Rollback() }
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if !activeSession {
		if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: append evidence "+record.ID); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
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
	tx, expected, activeSession, err := mutationTransaction(active, validator, "assessment")
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	engine := state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(expected), nil },
	}
	record, err = engine.AppendAssessment(string(expected), record)
	if err != nil {
		if !activeSession { _ = tx.Rollback() }
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if !activeSession {
		if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: append assessment "+record.ID); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
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
	tx, expected, activeSession, err := mutationTransaction(active, validator, "rebuild")
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	projection, err := (state.Store{
		Root: tx.StageRoot(), Catalog: domain.NewRegistry(), Validator: validator,
		RevisionProvider: func() (string, error) { return string(expected), nil },
	}).RebuildProjection()
	if err != nil {
		if !activeSession { _ = tx.Rollback() }
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if !activeSession {
		if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: rebuild learner state"); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
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

func mutationTransaction(active storepkg.Store, validator *workspace.Validator, kind string) (*storepkg.Transaction, storepkg.Revision, bool, error) {
	runtimeDir, err := lifecycle.DefaultRuntimeDir()
	if err != nil {
		return nil, "", false, err
	}
	coordinator := storepkg.NewCoordinator(active, validator, runtimeDir)
	manifest, err := workspace.ReadManifest(active.Root())
	if err != nil {
		return nil, "", false, err
	}
	if manifest.WorkspaceID != "" {
		recovery, recoveryErr := coordinator.InspectRecovery(manifest.WorkspaceID)
		if recoveryErr == nil {
			switch recovery.Status {
			case storepkg.RecoveryStaged:
				sessionHandle := filepath.Join(runtimeDir, "sessions", manifest.WorkspaceID+".json")
				if _, statErr := os.Stat(sessionHandle); statErr != nil {
					if errors.Is(statErr, os.ErrNotExist) {
						return nil, "", false, fmt.Errorf("workspace has an interrupted staged transaction; inspect/resume or discard recovery before new mutations")
					}
					return nil, "", false, statErr
				}
				tx, err := coordinator.Resume(manifest.WorkspaceID)
				if err != nil {
					return nil, "", false, err
				}
				return tx, tx.BaseRevision(), true, nil
			case storepkg.RecoverySyncPending, storepkg.RecoveryCheckpointed:
				return nil, "", false, fmt.Errorf("workspace has checkpointed session state awaiting synchronization; complete session recovery before new mutations")
			}
		} else if !errors.Is(recoveryErr, os.ErrNotExist) {
			return nil, "", false, recoveryErr
		}
	}
	revision, err := active.Revision(context.Background())
	if err != nil {
		return nil, "", false, err
	}
	sessionID := fmt.Sprintf("cli-%s-%d", kind, time.Now().UTC().UnixNano())
	tx, err := coordinator.Begin(context.Background(), sessionID, revision)
	if err != nil {
		return nil, "", false, err
	}
	return tx, revision, false, nil
}
