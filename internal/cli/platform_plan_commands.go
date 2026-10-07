package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// Target adaptation error codes.
const (
	codeStaleProjectionRevision = "stale-projection-revision"
	codeUnknownUnit             = "unknown-unit"
	codeModeNotAllowed          = "mode-not-allowed"
	codeUnknownDecision         = "unknown-decision"
)

// adaptationInputs are the target-side canonical inputs of a Target
// Adaptation Projection; learner state and decisions come from the workspace.
type adaptationInputs struct {
	curriculum  platform.Curriculum
	mapping     platform.MappingReport
	constraints *platform.TargetConstraints
}

// readAdaptationInputs reads and validates the curriculum export, mapping and
// optional target constraints. On failure it has already written the error.
func (a App) readAdaptationInputs(adapter platform.Adapter, target string, flags platformFlags) (adaptationInputs, int, bool) {
	if flags.mapping == "" {
		return adaptationInputs{}, a.platformUsageError("--mapping is required: pass the platform's content mapping file"), false
	}
	curriculum, code, ok := a.readCurriculum(adapter, target, flags)
	if !ok {
		return adaptationInputs{}, code, false
	}
	mapping, code, ok := a.readValidMapping(adapter, target, flags, curriculum)
	if !ok {
		return adaptationInputs{}, code, false
	}
	inputs := adaptationInputs{curriculum: curriculum, mapping: mapping}
	if flags.constraints != "" {
		data, err := os.ReadFile(flags.constraints)
		if err != nil {
			return adaptationInputs{}, a.platformFailure(fmt.Errorf("read target constraints: %w", err)), false
		}
		constraints, err := platform.ReadTargetConstraints(data, filepath.Base(flags.constraints), curriculum)
		if err != nil {
			return adaptationInputs{}, a.platformFailure(err), false
		}
		inputs.constraints = &constraints
	}
	return inputs, 0, true
}

// projectTarget builds the learner's Target Adaptation Projection from the
// workspace's competency state and Accepted Adaptation Decisions.
func projectTarget(ws platformWorkspace, inputs adaptationInputs) (platform.TargetAdaptationProjection, []state.AdaptationDecision, error) {
	learnerState, err := ws.engine.CompetencyProjection()
	if err != nil {
		return platform.TargetAdaptationProjection{}, nil, err
	}
	decisions, err := ws.engine.AdaptationDecisions()
	if err != nil {
		return platform.TargetAdaptationProjection{}, nil, err
	}
	projection, err := platform.ProjectTargetAdaptation(platform.AdaptationRequest{
		LearnerID:    ws.learnerID,
		Curriculum:   inputs.curriculum,
		Mapping:      inputs.mapping,
		Packs:        ws.packs,
		LearnerState: learnerState,
		Constraints:  inputs.constraints,
		Decisions:    decisions,
	})
	return projection, decisions, err
}

// writeProjection replaces the generated projection file in the workspace.
func writeProjection(ws platformWorkspace, projection platform.TargetAdaptationProjection) (string, error) {
	data, err := platform.EncodeTargetAdaptation(projection)
	if err != nil {
		return "", err
	}
	if err := ws.engine.WriteTargetAdaptation(projection.Target.Platform, projection.Target.Target, data); err != nil {
		return "", err
	}
	return state.TargetAdaptationPath(projection.Target.Platform, projection.Target.Target), nil
}

type planOutput struct {
	SchemaVersion  int                                 `json:"schemaVersion"`
	Adapter        string                              `json:"adapter"`
	Target         platform.ExternalID                 `json:"target"`
	Path           string                              `json:"path"`
	Projection     platform.TargetAdaptationProjection `json:"projection"`
	Specifications planSpecifications                  `json:"specifications"`
	Authoring      platform.AuthoringOutcome           `json:"authoring"`
	AuthoringPlan  *planAuthoringRecord                `json:"authoringPlan"`
}

// runPlatformPlan rebuilds the learner's Target Adaptation Projection for one
// shared Learning Target and writes it as a generated workspace file. Agents
// read proposedMode as their Adaptation Proposal; only the learner turns it
// into an Accepted Adaptation Decision. From the projection it reconciles the
// immutable Curriculum and Learning Unit Specification versions and records
// the Authoring Plan for the selected Authoring Intent.
func runPlatformPlan(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.intent != "" && !platform.ValidIntent(flags.intent) {
		return a.platformUsageError(fmt.Sprintf("unknown authoring intent %q (intents: target-skeleton, curriculum, unit, activity, patch)", flags.intent))
	}
	inputs, code, ok := a.readAdaptationInputs(adapter, target, flags)
	if !ok {
		return code
	}
	ws, code, ok := a.openPlatformWorkspace(flags.workspace, "platform-plan")
	if !ok {
		return code
	}
	projection, _, err := projectTarget(ws, inputs)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	path, err := writeProjection(ws, projection)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	specifications, authoring, plan, err := specifyTarget(ws, adapter, inputs, projection, flags)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	if err := ws.finish(true, fmt.Sprintf("alp: plan %s %s target adaptation", adapter.ID(), target)); err != nil {
		return a.platformFailure(err)
	}
	return a.writePlatformJSON(planOutput{
		SchemaVersion: 1, Adapter: adapter.ID(), Target: projection.Target, Path: path, Projection: projection,
		Specifications: specifications, Authoring: authoring, AuthoringPlan: plan,
	}, 0)
}

type decisionOutput struct {
	SchemaVersion int                                 `json:"schemaVersion"`
	Status        string                              `json:"status"`
	Decision      state.AdaptationDecision            `json:"decision"`
	Path          string                              `json:"path"`
	Projection    platform.TargetAdaptationProjection `json:"projection"`
}

// requireDecisionConfirmation refuses a decision the learner did not confirm.
func (a App) requireDecisionConfirmation(adapter platform.Adapter, target, what string, flags platformFlags) (int, bool) {
	if flags.basis == "" {
		return a.platformUsageError("--basis is required: pass the projection revision the learner confirmed against"), false
	}
	if !flags.confirm {
		return a.platformFailure(&platform.Error{
			Code: codeLearnerConfirmationRequired, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("%s requires the learner's explicit confirmation (--confirm); agents may only propose adaptations", what),
		}), false
	}
	return 0, true
}

// openDecisionBasis opens the workspace and rebuilds the projection the
// decision must be based on. On failure it has already written the error.
func (a App) openDecisionBasis(adapter platform.Adapter, target string, flags platformFlags, inputs adaptationInputs) (platformWorkspace, platform.TargetAdaptationProjection, []state.AdaptationDecision, int, bool) {
	ws, code, ok := a.openPlatformWorkspace(flags.workspace, "platform-decision")
	if !ok {
		return platformWorkspace{}, platform.TargetAdaptationProjection{}, nil, code, false
	}
	projection, decisions, err := projectTarget(ws, inputs)
	if err != nil {
		ws.abandon()
		return platformWorkspace{}, platform.TargetAdaptationProjection{}, nil, a.platformFailure(err), false
	}
	if flags.basis != projection.Revision {
		ws.abandon()
		return platformWorkspace{}, platform.TargetAdaptationProjection{}, nil, a.platformFailure(&platform.Error{
			Code: codeStaleProjectionRevision, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("basis projection revision %q is not the current Target Adaptation Projection revision %q; re-run alp platform plan and confirm against the current projection", flags.basis, projection.Revision),
		}), false
	}
	return ws, projection, decisions, 0, true
}

// recordDecision appends a confirmed decision, rebuilds the projection with
// it, and checkpoints both in one Store transaction.
func (a App) recordDecision(ws platformWorkspace, inputs adaptationInputs, decision state.AdaptationDecision, status string) int {
	now := time.Now().UTC().Format(time.RFC3339)
	decision.RecordedAt = now
	decision.LearnerID = ws.learnerID
	decision.Confirmation = state.DecisionConfirmation{ConfirmedBy: state.LearnerConfirmation, ConfirmedAt: now}
	recorded, err := ws.engine.AppendAdaptationDecision(ws.expected, decision)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	projection, _, err := projectTarget(ws, inputs)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	path, err := writeProjection(ws, projection)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	if err := ws.finish(true, fmt.Sprintf("alp: %s adaptation decision %s", recorded.Action, recorded.ID)); err != nil {
		return a.platformFailure(err)
	}
	return a.writePlatformJSON(decisionOutput{SchemaVersion: 1, Status: status, Decision: recorded, Path: path, Projection: projection}, 0)
}

func runPlatformDecisionAccept(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.unit == "" || flags.mode == "" {
		return a.platformUsageError("--unit and --mode are required: pass the unit item ID and the accepted adaptation mode")
	}
	if !validMode(flags.mode) {
		return a.platformUsageError(fmt.Sprintf("unknown adaptation mode %q (modes: skip, challenge, skim, full)", flags.mode))
	}
	if code, ok := a.requireDecisionConfirmation(adapter, target, fmt.Sprintf("accepting %s for unit %q", flags.mode, flags.unit), flags); !ok {
		return code
	}
	inputs, code, ok := a.readAdaptationInputs(adapter, target, flags)
	if !ok {
		return code
	}
	ws, projection, decisions, code, ok := a.openDecisionBasis(adapter, target, flags, inputs)
	if !ok {
		return code
	}
	known := false
	for _, unit := range projection.Units {
		known = known || unit.Item.Item == flags.unit
	}
	if !known {
		ws.abandon()
		return a.platformFailure(&platform.Error{
			Code: codeUnknownUnit, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("%q is not a unit of target %q", flags.unit, target),
		})
	}
	if !platform.AllowsMode(inputs.constraints, flags.unit, flags.mode) {
		ws.abandon()
		return a.platformFailure(&platform.Error{
			Code: codeModeNotAllowed, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("target constraints do not allow %s for unit %q", flags.mode, flags.unit),
		})
	}
	decision := state.AdaptationDecision{
		Target: state.DecisionTarget{Platform: projection.Target.Platform, Target: projection.Target.Target},
		Unit:   flags.unit, Action: state.DecisionAccept, Mode: flags.mode, Reason: flags.reason,
		Basis: state.DecisionBasis{ProjectionRevision: projection.Revision},
	}
	if current, ok := platform.ActiveAdaptationDecisions(decisions, projection.Target)[flags.unit]; ok {
		if current.Mode == flags.mode {
			ws.abandon()
			return a.writePlatformJSON(decisionOutput{
				SchemaVersion: 1, Status: "already-accepted", Decision: current,
				Path: state.TargetAdaptationPath(projection.Target.Platform, projection.Target.Target), Projection: projection,
			}, 0)
		}
		decision.Supersedes = []string{current.ID}
	}
	return a.recordDecision(ws, inputs, decision, "accepted")
}

func runPlatformDecisionRevoke(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.decision == "" {
		return a.platformUsageError("--decision is required: pass the ID of the accepted decision to revoke")
	}
	if code, ok := a.requireDecisionConfirmation(adapter, target, fmt.Sprintf("revoking adaptation decision %q", flags.decision), flags); !ok {
		return code
	}
	inputs, code, ok := a.readAdaptationInputs(adapter, target, flags)
	if !ok {
		return code
	}
	ws, projection, decisions, code, ok := a.openDecisionBasis(adapter, target, flags, inputs)
	if !ok {
		return code
	}
	var revoked *state.AdaptationDecision
	for _, active := range platform.ActiveAdaptationDecisions(decisions, projection.Target) {
		if active.ID == flags.decision {
			found := active
			revoked = &found
		}
	}
	if revoked == nil {
		ws.abandon()
		return a.platformFailure(&platform.Error{
			Code: codeUnknownDecision, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("%q is not an active accepted adaptation decision for target %q", flags.decision, target),
		})
	}
	return a.recordDecision(ws, inputs, state.AdaptationDecision{
		Target: revoked.Target, Unit: revoked.Unit, Action: state.DecisionRevoke, Reason: flags.reason,
		Basis:      state.DecisionBasis{ProjectionRevision: projection.Revision},
		Supersedes: []string{revoked.ID},
	}, "revoked")
}

func validMode(mode string) bool {
	for _, known := range platform.Modes {
		if known == mode {
			return true
		}
	}
	return false
}
