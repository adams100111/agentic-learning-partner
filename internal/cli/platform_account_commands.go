package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// Platform workspace error codes.
const (
	codeInvalidWorkspace              = "invalid-workspace"
	codeLearnerConfirmationRequired   = "learner-confirmation-required"
	codeAccountLinkedToAnotherLearner = "platform-account-linked-to-another-learner"
)

// platformWorkspace is a learner workspace opened for a platform mutation.
type platformWorkspace struct {
	learnerID string
	// packs is the domain pack registry every platform command in this
	// workspace reads: the Store catalog, projections and specifications.
	packs    domain.Registry
	engine   state.Store
	expected string
	finish   func(changed bool, message string) error
	abandon  func()
}

// openPlatformWorkspace resolves and validates the learner workspace and
// begins a Store transaction on it; every write goes through engine. On
// failure it has already written the error and returns the exit code.
func (a App) openPlatformWorkspace(explicit, kind string) (platformWorkspace, int, bool) {
	_, info, ok := a.checkWorkspace(explicit)
	if !ok {
		return platformWorkspace{}, a.platformFailure(&platform.Error{Code: codeInvalidWorkspace, Message: "learner workspace cannot be opened; see stderr"}), false
	}
	manifest, err := workspace.ReadManifest(info.Path)
	if err != nil {
		return platformWorkspace{}, a.platformFailure(&platform.Error{Code: codeInvalidWorkspace, Message: err.Error()}), false
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		return platformWorkspace{}, a.platformFailure(fmt.Errorf("initialize validation: %w", err)), false
	}
	active, err := openStateStore(info.Path, validator)
	if err != nil {
		return platformWorkspace{}, a.platformFailure(err), false
	}
	tx, expected, activeSession, err := mutationTransaction(active, validator, kind)
	if err != nil {
		return platformWorkspace{}, a.platformFailure(err), false
	}
	packs := domain.NewRegistry()
	abandon := func() {
		if !activeSession {
			_ = tx.Rollback()
		}
	}
	return platformWorkspace{
		learnerID: manifest.LearnerID,
		packs:     packs,
		engine: state.Store{
			Root: tx.StageRoot(), Catalog: packs, Validator: validator,
			RevisionProvider: func() (string, error) { return string(expected), nil },
		},
		expected: string(expected),
		finish: func(changed bool, message string) error {
			if activeSession {
				// An active session publishes staged records at its close.
				return nil
			}
			if !changed {
				return tx.Rollback()
			}
			_, err := tx.CheckpointWithMessage(context.Background(), false, message)
			return err
		},
		abandon: abandon,
	}, 0, true
}

type accountLinkOutput struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Status        string                    `json:"status"`
	Link          state.PlatformAccountLink `json:"link"`
}

func runPlatformAccountLink(a App, adapter platform.Adapter, _ string, flags platformFlags) int {
	if flags.instance == "" || flags.user == "" {
		return a.platformUsageError("--instance and --user are required: pass the platform instance and platform user ID from the platform's activity export")
	}
	if !flags.confirm {
		return a.platformFailure(&platform.Error{
			Code:    codeLearnerConfirmationRequired,
			Adapter: adapter.ID(),
			Message: fmt.Sprintf("linking platform account %q on %s instance %q requires the learner's explicit confirmation (--confirm); agents must not link accounts on their own", flags.user, adapter.ID(), flags.instance),
		})
	}
	ws, code, ok := a.openPlatformWorkspace(flags.workspace, "platform-account-link")
	if !ok {
		return code
	}
	existing, found, err := ws.engine.LinkedAccount(adapter.ID(), flags.instance, flags.user)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	if found {
		ws.abandon()
		if existing.LearnerID != ws.learnerID {
			return a.platformFailure(&platform.Error{
				Code:    codeAccountLinkedToAnotherLearner,
				Adapter: adapter.ID(),
				Message: fmt.Sprintf("platform account %q on %s instance %q is linked to learner %q, not workspace learner %q", flags.user, adapter.ID(), flags.instance, existing.LearnerID, ws.learnerID),
			})
		}
		return a.writePlatformJSON(accountLinkOutput{SchemaVersion: 1, Status: "already-linked", Link: existing}, 0)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	link, err := ws.engine.AppendPlatformAccountLink(ws.expected, state.PlatformAccountLink{
		RecordedAt:     now,
		Platform:       adapter.ID(),
		Instance:       flags.instance,
		PlatformUserID: flags.user,
		LearnerID:      ws.learnerID,
		Confirmation:   state.AccountLinkConfirmation{ConfirmedBy: state.LearnerConfirmation, ConfirmedAt: now},
	})
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	if err := ws.finish(true, "alp: link platform account "+link.ID); err != nil {
		return a.platformFailure(err)
	}
	return a.writePlatformJSON(accountLinkOutput{SchemaVersion: 1, Status: "linked", Link: link}, 0)
}
