package pylearn

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

type Adapter struct {
	Domains domain.Registry
}

func NewAdapter() Adapter {
	return Adapter{Domains: domain.NewRegistry()}
}

func (a Adapter) ValidateMapping(mapping Mapping) error {
	if mapping.SchemaVersion != 1 {
		return fmt.Errorf("unsupported platform mapping schemaVersion %d", mapping.SchemaVersion)
	}
	if mapping.Platform != "pylearn" {
		return fmt.Errorf("expected platform %q, got %q", "pylearn", mapping.Platform)
	}

	seen := map[string]struct{}{}
	for _, entry := range mapping.Mappings {
		if _, duplicate := seen[entry.ContentID]; duplicate {
			return fmt.Errorf("duplicate PyLearn mapping for content %q", entry.ContentID)
		}
		seen[entry.ContentID] = struct{}{}
		if err := a.Domains.CheckCompatibility(entry.Domain, entry.PackVersion); err != nil {
			return fmt.Errorf("content %q: %w", entry.ContentID, err)
		}
		pack, err := a.Domains.Load(entry.Domain)
		if err != nil {
			return err
		}
		for _, competency := range entry.Competencies {
			if !pack.HasCompetency(competency) {
				return fmt.Errorf("content %q maps unknown competency %q", entry.ContentID, competency)
			}
		}
	}
	return nil
}

func (a Adapter) Normalize(export Export, mapping Mapping, currentProfile map[string]any) (Result, error) {
	if export.SchemaVersion != 1 {
		return Result{}, fmt.Errorf("unsupported PyLearn export schemaVersion %d", export.SchemaVersion)
	}
	if err := a.ValidateMapping(mapping); err != nil {
		return Result{}, err
	}

	byContent := make(map[string]ContentMapping, len(mapping.Mappings))
	for _, entry := range mapping.Mappings {
		byContent[entry.ContentID] = entry
	}

	result := Result{}
	seen := map[string]struct{}{}
	add := func(kind, sourceID, contentID, observedAt, evidenceType, observation, outcome, strength, failureClass string) error {
		entry, ok := byContent[contentID]
		if !ok {
			return nil
		}
		stable := stableID(kind, sourceID, contentID)
		if _, duplicate := seen[stable]; duplicate {
			return nil
		}
		seen[stable] = struct{}{}
		record := state.Evidence{
			SchemaVersion: 1,
			ID:            "ev_" + stable,
			RecordedAt:    fallbackTimestamp(observedAt, export.ExportedAt),
			Domain:        entry.Domain,
			Competencies:  append([]string(nil), entry.Competencies...),
			Type:          evidenceType,
			Source: state.EvidenceSource{
				Kind:      "platform",
				Ref:       "pylearn:" + kind + ":" + sourceID,
				ContentID: contentID,
			},
			Observation:  observation,
			Result:       outcome,
			Strength:     strength,
			FailureClass: failureClass,
			Metadata: map[string]any{
				"platform": "pylearn",
				"kind":     kind,
			},
		}
		result.Evidence = append(result.Evidence, NormalizedEvidence{StableKey: stable, Record: record})
		return nil
	}

	for _, progress := range export.Progress {
		_ = add("progress", progress.ID, progress.ContentID, progress.UpdatedAt, "platform-event",
			"PyLearn content progress is "+progress.Status+".", "neutral", "weak", "")
	}
	for _, attempt := range export.Attempts {
		outcome, strength, failureClass := "fail", "moderate", "implementation-miss"
		observation := "PyLearn exercise attempt did not pass."
		if attempt.Passed {
			outcome, strength, failureClass = "pass", "moderate", ""
			observation = "PyLearn exercise attempt passed its configured checks."
		}
		_ = add("attempt", attempt.ID, attempt.ContentID, attempt.CreatedAt, "exercise", observation, outcome, strength, failureClass)
	}
	for _, answer := range export.QuizAnswers {
		outcome, observation := "fail", "PyLearn quiz answer was incorrect."
		failureClass := "conceptual-miss"
		if answer.Correct {
			outcome, observation, failureClass = "pass", "PyLearn quiz answer was correct.", ""
		}
		_ = add("quiz", answer.ID, answer.ContentID, answer.CreatedAt, "quiz", observation, outcome, "weak", failureClass)
	}
	for _, reflection := range export.Reflections {
		_ = add("reflection", reflection.ID, reflection.ContentID, reflection.CreatedAt, "reflection",
			"PyLearn learner reflection: "+strings.TrimSpace(reflection.Text), "neutral", "weak", "")
	}
	for _, mastery := range export.ConceptMastery {
		result.DerivedSignals = append(result.DerivedSignals, DerivedSignal{
			Kind:      "concept-mastery-rollup",
			ContentID: mastery.ContentID,
			Ref:       "pylearn:concept-mastery:" + mastery.ID,
			Summary:   fmt.Sprintf("platform rollup attempts=%d passes=%d failures=%d", mastery.Attempts, mastery.Passes, mastery.Failures),
		})
	}
	for _, bookmark := range export.Bookmarks {
		result.DerivedSignals = append(result.DerivedSignals, DerivedSignal{
			Kind: "bookmark", ContentID: bookmark.ContentID, Ref: "pylearn:bookmark:" + bookmark.ID,
			Summary: "interest/friction signal only; not competency evidence",
		})
	}

	result.ProfileConflicts = DetectProfileConflicts(export.Learner, currentProfile)
	sort.Slice(result.Evidence, func(i, j int) bool { return result.Evidence[i].StableKey < result.Evidence[j].StableKey })
	sort.Slice(result.DerivedSignals, func(i, j int) bool { return result.DerivedSignals[i].Ref < result.DerivedSignals[j].Ref })
	return result, nil
}

func DetectProfileConflicts(platform, current map[string]any) []ProfileConflict {
	var conflicts []ProfileConflict
	platformExperience, _ := platform["experience"].(map[string]any)
	currentExperience, _ := current["experience"].(map[string]any)
	for stack, rawPlatform := range platformExperience {
		rawCurrent, exists := currentExperience[stack]
		if !exists {
			continue
		}
		platformLevel := nestedString(rawPlatform, "level")
		currentLevel := nestedString(rawCurrent, "level")
		if platformLevel != "" && currentLevel != "" && platformLevel != currentLevel {
			conflicts = append(conflicts, ProfileConflict{
				Field: "experience." + stack + ".level", Platform: platformLevel, Current: currentLevel,
			})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Field < conflicts[j].Field })
	return conflicts
}

func stableID(kind, sourceID, contentID string) string {
	sum := sha256.Sum256([]byte("pylearn|" + kind + "|" + sourceID + "|" + contentID))
	return hex.EncodeToString(sum[:12])
}

func fallbackTimestamp(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func nestedString(value any, key string) string {
	mapping, _ := value.(map[string]any)
	text, _ := mapping[key].(string)
	return text
}
