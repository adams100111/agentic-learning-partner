package pylearn

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

type Adapter struct {
	Domains domain.Registry
}

func NewAdapter() Adapter {
	return Adapter{Domains: domain.NewRegistry()}
}

// Normalize turns a PyLearn export into evidence using a validated v2 content
// mapping (see ValidateContentMapping). Every mapped competency contributes
// its resolved current ID; role-graded evidence is introduced with import
// (ADR-0058, ADR-0059).
func (a Adapter) Normalize(export Export, mapping platform.MappingReport, currentProfile map[string]any) (Result, error) {
	if export.SchemaVersion != 1 {
		return Result{}, fmt.Errorf("unsupported PyLearn export schemaVersion %d", export.SchemaVersion)
	}
	if !mapping.Valid {
		return Result{}, fmt.Errorf("platform mapping is not valid (%d errors); run alp platform mapping validate", mapping.Summary.Errors)
	}

	type mappedDomain struct {
		domain       string
		competencies []string
	}
	byContent := make(map[string][]mappedDomain, len(mapping.Entries))
	for _, entry := range mapping.Entries {
		var domains []mappedDomain
		for _, competency := range entry.Competencies {
			index := -1
			for i := range domains {
				if domains[i].domain == competency.Domain {
					index = i
				}
			}
			if index < 0 {
				domains = append(domains, mappedDomain{domain: competency.Domain})
				index = len(domains) - 1
			}
			domains[index].competencies = append(domains[index].competencies, competency.ResolvedID)
		}
		byContent[entry.Item.Item] = domains
	}

	result := Result{}
	seen := map[string]struct{}{}
	add := func(kind, sourceID, contentID, observedAt, evidenceType, observation, outcome, strength, failureClass string) {
		for _, mapped := range byContent[contentID] {
			stable := stableID(kind, sourceID, contentID, mapped.domain)
			if _, duplicate := seen[stable]; duplicate {
				continue
			}
			seen[stable] = struct{}{}
			record := state.Evidence{
				SchemaVersion: 1,
				ID:            "ev_" + stable,
				RecordedAt:    fallbackTimestamp(observedAt, export.ExportedAt),
				Domain:        mapped.domain,
				Competencies:  append([]string(nil), mapped.competencies...),
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
		}
	}

	for _, progress := range export.Progress {
		add("progress", progress.ID, progress.ContentID, progress.UpdatedAt, "platform-event",
			"PyLearn content progress is "+progress.Status+".", "neutral", "weak", "")
	}
	for _, attempt := range export.Attempts {
		outcome, strength, failureClass := "fail", "moderate", "implementation-miss"
		observation := "PyLearn exercise attempt did not pass."
		if attempt.Passed {
			outcome, strength, failureClass = "pass", "moderate", ""
			observation = "PyLearn exercise attempt passed its configured checks."
		}
		add("attempt", attempt.ID, attempt.ContentID, attempt.CreatedAt, "exercise", observation, outcome, strength, failureClass)
	}
	for _, answer := range export.QuizAnswers {
		outcome, observation := "fail", "PyLearn quiz answer was incorrect."
		failureClass := "conceptual-miss"
		if answer.Correct {
			outcome, observation, failureClass = "pass", "PyLearn quiz answer was correct.", ""
		}
		add("quiz", answer.ID, answer.ContentID, answer.CreatedAt, "quiz", observation, outcome, "weak", failureClass)
	}
	for _, reflection := range export.Reflections {
		add("reflection", reflection.ID, reflection.ContentID, reflection.CreatedAt, "reflection",
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

func stableID(kind, sourceID, contentID, domainName string) string {
	sum := sha256.Sum256([]byte("pylearn|" + kind + "|" + sourceID + "|" + contentID + "|" + domainName))
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
