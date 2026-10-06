package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
)

type importOutput struct {
	SchemaVersion    int                       `json:"schemaVersion"`
	Adapter          string                    `json:"adapter"`
	Target           platform.ExternalID       `json:"target"`
	Account          importAccount             `json:"account"`
	CurriculumExport inspectSource             `json:"curriculumExport"`
	Mapping          platform.MappingSource    `json:"mapping"`
	Cursor           platform.CursorRange      `json:"cursor"`
	Counts           platform.ImportCounts     `json:"counts"`
	Records          []platform.ImportedRecord `json:"records"`
}

type importAccount struct {
	Instance       string `json:"instance"`
	PlatformUserID string `json:"platformUserId"`
	LearnerID      string `json:"learnerId"`
	Link           string `json:"link"`
}

// runPlatformImport imports one target's activity for a linked platform
// account. Evidence is graded through the target's validated mapping and
// written through one Store transaction; an import that adds nothing leaves
// the workspace revision unchanged.
func runPlatformImport(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.export == "" || flags.mapping == "" {
		return a.platformUsageError("--export and --mapping are required: pass the platform's activity export and content mapping")
	}
	reader, ok := adapter.(platform.ActivityReader)
	if !ok {
		return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.ActivitySource))
	}
	mapper, ok := adapter.(platform.ContentMappingValidator)
	if !ok {
		return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.ContentMapper))
	}
	curriculum, code, ok := a.readCurriculum(adapter, target, flags)
	if !ok {
		return code
	}
	mappingData, err := os.ReadFile(flags.mapping)
	if err != nil {
		return a.platformFailure(fmt.Errorf("read mapping: %w", err))
	}
	mapping, err := mapper.ValidateContentMapping(mappingData, filepath.Base(flags.mapping), curriculum)
	if err != nil {
		return a.platformFailure(err)
	}
	if !mapping.Valid {
		return a.platformFailure(&platform.Error{
			Code: platform.CodeInvalidMapping, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("platform mapping %s is not valid (%d errors); run alp platform mapping validate", filepath.Base(flags.mapping), mapping.Summary.Errors),
		})
	}
	exportData, err := os.ReadFile(flags.export)
	if err != nil {
		return a.platformFailure(fmt.Errorf("read activity export: %w", err))
	}
	var since *string
	if flags.cursorSet {
		since = &flags.cursor
	}
	batch, err := reader.ReadActivity(exportData, target, since)
	if err != nil {
		return a.platformFailure(err)
	}

	ws, code, ok := a.openPlatformWorkspace(flags.workspace, "platform-import")
	if !ok {
		return code
	}
	link, linked, err := ws.engine.LinkedAccount(adapter.ID(), batch.Instance, batch.PlatformUserID)
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	if !linked || link.LearnerID != ws.learnerID {
		ws.abandon()
		return a.platformFailure(&platform.Error{
			Code: platform.CodeAccountNotLinked, Adapter: adapter.ID(), Target: target,
			Message: fmt.Sprintf("platform user %q on %s instance %q is not linked to workspace learner %q; the learner must confirm the account with alp platform account link (learners are never matched by email or name)",
				batch.PlatformUserID, adapter.ID(), batch.Instance, ws.learnerID),
		})
	}
	existing, err := ws.engine.ListEvidence()
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	plan, err := platform.PlanImport(platform.ImportRequest{
		Batch:      batch,
		Curriculum: curriculum,
		Mapping:    mapping,
		Account:    platform.Account{Platform: adapter.ID(), Instance: batch.Instance, PlatformUserID: batch.PlatformUserID, LearnerID: ws.learnerID},
		Existing:   existing,
	})
	if err != nil {
		ws.abandon()
		return a.platformFailure(err)
	}
	for _, evidence := range plan.Evidence {
		if _, err := ws.engine.AppendEvidence(ws.expected, evidence); err != nil {
			ws.abandon()
			return a.platformFailure(err)
		}
	}
	message := fmt.Sprintf("alp: import %s %s activity (%d imported, %d superseded)", adapter.ID(), target, plan.Report.Counts.Imported, plan.Report.Counts.Superseded)
	if err := ws.finish(len(plan.Evidence) != 0, message); err != nil {
		return a.platformFailure(err)
	}
	return a.writePlatformJSON(importOutput{
		SchemaVersion:    1,
		Adapter:          adapter.ID(),
		Target:           curriculum.Target,
		Account:          importAccount{Instance: batch.Instance, PlatformUserID: batch.PlatformUserID, LearnerID: ws.learnerID, Link: link.ID},
		CurriculumExport: inspectSource{SchemaVersion: curriculum.SchemaVersion, ContentHash: curriculum.ContentHash},
		Mapping:          mapping.Mapping,
		Cursor:           batch.Cursor,
		Counts:           plan.Report.Counts,
		Records:          plan.Report.Records,
	}, 0)
}
