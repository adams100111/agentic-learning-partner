package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// Gate record statuses.
const (
	gatesRecorded        = "recorded"
	gatesAlreadyRecorded = "already-recorded"
)

type gateRealization struct {
	Path string `json:"path"`
	platform.RealizationLink
}

type gatesRecordOutput struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Adapter       string              `json:"adapter"`
	Target        platform.ExternalID `json:"target"`
	Status        string              `json:"status"`
	Path          string              `json:"path"`
	Record        platform.GateRecord `json:"record"`
	Realizations  []gateRealization   `json:"realizations"`
}

// runPlatformGatesRecord attaches a Platform Gate Result to a recorded
// Authoring Plan (Q22) and records a Realization Link for each unit the
// result realizes: only when the platform reports it publishable, the
// authoring target reported the declared-stable items realizing the unit,
// and every claim of the unit has source provenance (Q27).
func runPlatformGatesRecord(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.plan == "" || flags.result == "" {
		return a.platformUsageError("--plan and --result are required: pass the Authoring Plan ID and the platform's gate result file")
	}
	reader, ok := adapter.(platform.GateResultReader)
	if !ok {
		return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.PlatformValidator))
	}
	curriculum, code, ok := a.readCurriculum(adapter, target, flags)
	if !ok {
		return code
	}
	data, err := os.ReadFile(flags.result)
	if err != nil {
		return a.platformFailure(fmt.Errorf("read gate result: %w", err))
	}
	result, err := reader.ReadGateResult(data, target)
	if err != nil {
		return a.platformFailure(err)
	}
	var report *platform.RealizationReport
	if flags.realization != "" {
		data, err := os.ReadFile(flags.realization)
		if err != nil {
			return a.platformFailure(fmt.Errorf("read realization report: %w", err))
		}
		read, err := platform.ReadRealizationReport(data, curriculum.Target)
		if err != nil {
			return a.platformFailure(err)
		}
		report = &read
	}

	ws, code, ok := a.openPlatformWorkspace(flags.workspace, "platform-gates")
	if !ok {
		return code
	}
	output, err := attachGateResult(ws, adapter, curriculum, result, report, flags.plan)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	changed := output.Status == gatesRecorded
	message := fmt.Sprintf("alp: record %s gate result %s for authoring plan %s", adapter.ID(), output.Record.ID, flags.plan)
	if err := ws.finish(changed, message); err != nil {
		return a.platformFailure(err)
	}
	return a.writePlatformJSON(output, 0)
}

func attachGateResult(ws platformWorkspace, adapter platform.Adapter, curriculum platform.Curriculum, result platform.GateResult, report *platform.RealizationReport, planID string) (gatesRecordOutput, error) {
	target := curriculum.Target
	data, path, found, err := ws.engine.AuthoringPlan(planID)
	if err != nil {
		return gatesRecordOutput{}, err
	}
	if !found {
		return gatesRecordOutput{}, &platform.Error{Code: platform.CodeUnknownAuthoringPlan, Adapter: adapter.ID(), Target: target.Target,
			Message: fmt.Sprintf("authoring plan %q is not recorded in this workspace; run alp platform plan first", planID)}
	}
	plan, err := platform.DecodeAuthoringPlan(path, data)
	if err != nil {
		return gatesRecordOutput{}, err
	}
	if plan.Target.Platform != target.Platform || plan.Target.Target != target.Target {
		return gatesRecordOutput{}, &platform.Error{Code: platform.CodeUnknownAuthoringPlan, Adapter: adapter.ID(), Target: target.Target,
			Message: fmt.Sprintf("authoring plan %s is for target %s/%s, not %s/%s", planID, plan.Target.Platform, plan.Target.Target, target.Platform, target.Target)}
	}
	if plan.LearnerID != ws.learnerID {
		return gatesRecordOutput{}, &platform.Error{Code: platform.CodeUnknownAuthoringPlan, Adapter: adapter.ID(), Target: target.Target,
			Message: fmt.Sprintf("authoring plan %s belongs to learner %q, not this workspace's learner %q", planID, plan.LearnerID, ws.learnerID)}
	}
	units := make([]platform.LearningUnitSpec, 0, len(plan.Units))
	for _, ref := range plan.Units {
		data, path, found, err := ws.engine.Specification(state.UnitSpecs, ref.ID, ref.Version)
		if err != nil {
			return gatesRecordOutput{}, err
		}
		if !found {
			return gatesRecordOutput{}, &platform.Error{Code: platform.CodeSpecIntegrity, Adapter: adapter.ID(), Target: target.Target,
				Message: fmt.Sprintf("authoring plan %s cites %s v%d, which is not stored in this workspace", planID, ref.ID, ref.Version)}
		}
		spec, err := platform.DecodeUnitSpec(path, data)
		if err != nil {
			return gatesRecordOutput{}, err
		}
		if spec.ContentHash != ref.ContentHash {
			return gatesRecordOutput{}, &platform.Error{Code: platform.CodeSpecIntegrity, Adapter: adapter.ID(), Target: target.Target,
				Message: fmt.Sprintf("authoring plan %s cites %s v%d with hash %s, but the stored version hashes to %s", planID, ref.ID, ref.Version, ref.ContentHash, spec.ContentHash)}
		}
		units = append(units, spec)
	}
	links, err := loadRealizations(ws)
	if err != nil {
		return gatesRecordOutput{}, err
	}

	outcome, err := platform.RecordGates(platform.GateRequest{
		Plan: plan, Units: units, Result: result, Report: report, Curriculum: curriculum,
		Realizations: platform.IndexRealizations(links, target),
		RecordedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return gatesRecordOutput{}, err
	}
	output := gatesRecordOutput{SchemaVersion: 1, Adapter: adapter.ID(), Target: target, Realizations: []gateRealization{}}

	encoded, err := platform.EncodeSpec(outcome.Record)
	if err != nil {
		return gatesRecordOutput{}, err
	}
	recordPath, existing, err := ws.engine.RecordGateResult(ws.expected, plan.ID, outcome.Record.ID, encoded)
	if err != nil {
		return gatesRecordOutput{}, err
	}
	output.Path = recordPath
	if existing != nil {
		// The same result and report were already recorded for this plan:
		// report that record and its links, and write nothing.
		record, err := platform.DecodeGateRecord(recordPath, existing)
		if err != nil {
			return gatesRecordOutput{}, err
		}
		output.Status, output.Record = gatesAlreadyRecorded, record
		for _, link := range links {
			if link.GateRecord == record.ID {
				output.Realizations = append(output.Realizations, gateRealization{Path: state.RealizationPath(link.ID), RealizationLink: link})
			}
		}
		return output, nil
	}
	output.Status, output.Record = gatesRecorded, outcome.Record
	for _, link := range outcome.Links {
		encoded, err := platform.EncodeSpec(link)
		if err != nil {
			return gatesRecordOutput{}, err
		}
		path, err := ws.engine.AppendRealization(ws.expected, link.ID, encoded)
		if err != nil {
			return gatesRecordOutput{}, err
		}
		output.Realizations = append(output.Realizations, gateRealization{Path: path, RealizationLink: link})
	}
	return output, nil
}

// loadRealizations reads every recorded Realization Link in the workspace.
func loadRealizations(ws platformWorkspace) ([]platform.RealizationLink, error) {
	records, err := ws.engine.Realizations()
	if err != nil {
		return nil, err
	}
	links := make([]platform.RealizationLink, 0, len(records))
	for _, record := range records {
		link, err := platform.DecodeRealizationLink(record.Path, record.Data)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}
