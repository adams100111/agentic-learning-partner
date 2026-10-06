package view

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/state"
	"go.yaml.in/yaml/v3"
)

type Format string

const (
	Text     Format = "text"
	Markdown Format = "markdown"
)

type Renderer struct {
	Root string
}

func (r Renderer) Persona(domain string, format Format) (string, error) {
	profile, err := readMap(filepath.Join(r.Root, "profile", "profile.yaml"))
	if err != nil {
		return "", fmt.Errorf("read learner profile: %w", err)
	}
	global, _ := readOptionalMap(filepath.Join(r.Root, "personas", "global.yaml"))
	var domainPersona map[string]any
	if domain != "" {
		domainPersona, _ = readOptionalMap(filepath.Join(r.Root, "personas", "domains", domain+".yaml"))
	}

	var b strings.Builder
	if format == Markdown {
		b.WriteString("# Learner Persona\n\n")
	} else {
		b.WriteString("Learner Persona\n")
	}

	learner, _ := profile["learner"].(map[string]any)
	writeKV(&b, format, "ID", stringValue(learner["id"]))
	writeKV(&b, format, "Name", stringValue(learner["displayName"]))
	writeKV(&b, format, "Professional level", stringValue(learner["professionalLevel"]))

	if experience, ok := profile["experience"].(map[string]any); ok {
		if format == Markdown {
			b.WriteString("\n## Experience\n\n")
		} else {
			b.WriteString("\nExperience\n")
		}
		type item struct {
			name  string
			rank  int
			level string
		}
		var items []item
		for name, raw := range experience {
			entry, _ := raw.(map[string]any)
			rank := intValue(entry["relativeRank"], 999)
			items = append(items, item{name: name, rank: rank, level: stringValue(entry["level"])})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].rank == items[j].rank {
				return items[i].name < items[j].name
			}
			return items[i].rank < items[j].rank
		})
		for _, item := range items {
			if format == Markdown {
				fmt.Fprintf(&b, "- **%s** — %s", item.name, item.level)
			} else {
				fmt.Fprintf(&b, "- %s: %s", item.name, item.level)
			}
			if item.rank != 999 {
				fmt.Fprintf(&b, " (rank %d)", item.rank)
			}
			b.WriteString("\n")
		}
	}

	if preferences, ok := profile["preferences"].(map[string]any); ok && len(preferences) > 0 {
		writeMapSection(&b, format, "Preferences", preferences)
	}
	if global != nil {
		writePersonaSection(&b, format, "Global persona", global)
	}
	if domainPersona != nil {
		writePersonaSection(&b, format, "Domain persona: "+domain, domainPersona)
	}
	return b.String(), nil
}

func (r Renderer) Status(format Format) (string, error) {
	var b strings.Builder
	if format == Markdown {
		b.WriteString("# Learning Status\n\n")
	} else {
		b.WriteString("Learning Status\n")
	}

	current, _ := readOptionalMap(filepath.Join(r.Root, "state", "current.yaml"))
	if current != nil {
		writeMapSection(&b, format, "Current focus", current)
	} else {
		writeLine(&b, format, "Current focus", "not set")
	}

	projection, _ := readProjection(filepath.Join(r.Root, "state", "competencies.yaml"))
	if len(projection.Competencies) > 0 {
		if format == Markdown {
			b.WriteString("\n## Competencies\n\n")
		} else {
			b.WriteString("\nCompetencies\n")
		}
		for _, competency := range projection.Competencies {
			line := fmt.Sprintf("%s — %s / %s", competency.ID, competency.Level, competency.Confidence)
			if competency.NeedsReassessment {
				line += " [reassessment needed]"
			}
			if format == Markdown {
				fmt.Fprintf(&b, "- %s\n", line)
			} else {
				fmt.Fprintf(&b, "- %s\n", line)
			}
		}
	}

	review, _ := readOptionalMap(filepath.Join(r.Root, "state", "review-queue.yaml"))
	if review != nil {
		writeMapSection(&b, format, "Review queue", review)
	}
	return b.String(), nil
}

func (r Renderer) Competency(id string, format Format) (string, error) {
	projection, err := readProjection(filepath.Join(r.Root, "state", "competencies.yaml"))
	if err != nil {
		return "", err
	}
	var current *state.ProjectedCompetency
	for i := range projection.Competencies {
		if projection.Competencies[i].ID == id {
			current = &projection.Competencies[i]
			break
		}
	}
	if current == nil {
		return "", fmt.Errorf("competency %q not found in current projection", id)
	}

	assessments, err := readAssessments(filepath.Join(r.Root, "assessments"))
	if err != nil {
		return "", err
	}
	var relevant []state.Assessment
	for _, assessment := range assessments {
		if assessment.Competency == id {
			relevant = append(relevant, assessment)
		}
	}
	sort.Slice(relevant, func(i, j int) bool {
		if relevant[i].RecordedAt == relevant[j].RecordedAt {
			return relevant[i].ID > relevant[j].ID
		}
		return relevant[i].RecordedAt > relevant[j].RecordedAt
	})

	var b strings.Builder
	if format == Markdown {
		fmt.Fprintf(&b, "# %s\n\n", id)
	} else {
		fmt.Fprintf(&b, "%s\n", id)
	}
	writeLine(&b, format, "Level", current.Level)
	writeLine(&b, format, "Confidence", current.Confidence)
	writeLine(&b, format, "Last verified", current.LastVerified)
	writeLine(&b, format, "Assessment", current.AssessmentID)
	if current.NeedsReassessment {
		writeLine(&b, format, "Reassessment", "needed")
	}
	if len(current.Gaps) > 0 {
		writeLine(&b, format, "Gaps", strings.Join(current.Gaps, ", "))
	}

	if len(relevant) > 0 {
		if format == Markdown {
			b.WriteString("\n## Assessment rationale\n\n")
		} else {
			b.WriteString("\nAssessment rationale\n")
		}
		for _, assessment := range relevant {
			status := assessment.Status
			fmt.Fprintf(&b, "- %s [%s] %s / %s — %s\n", assessment.ID, status, assessment.Judgment.Level, assessment.Confidence, assessment.Rationale)
			if len(assessment.Evidence) > 0 {
				fmt.Fprintf(&b, "  evidence: %s\n", strings.Join(assessment.Evidence, ", "))
			}
		}
	}
	return b.String(), nil
}

func (r Renderer) Evidence(id string, format Format) (string, error) {
	path := filepath.Join(r.Root, "evidence", id+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read evidence %s: %w", id, err)
	}
	var evidence state.Evidence
	if err := yaml.Unmarshal(data, &evidence); err != nil {
		return "", fmt.Errorf("parse evidence %s: %w", id, err)
	}

	var b strings.Builder
	if format == Markdown {
		fmt.Fprintf(&b, "# Evidence %s\n\n", evidence.ID)
	} else {
		fmt.Fprintf(&b, "Evidence %s\n", evidence.ID)
	}
	writeLine(&b, format, "Domain", evidence.Domain)
	writeLine(&b, format, "Competencies", strings.Join(evidence.Competencies, ", "))
	writeLine(&b, format, "Type", evidence.Type)
	writeLine(&b, format, "Strength", evidence.Strength)
	writeLine(&b, format, "Result", evidence.Result)
	if evidence.FailureClass != "" {
		writeLine(&b, format, "Failure classification", evidence.FailureClass)
	}
	writeLine(&b, format, "Recorded at", evidence.RecordedAt)
	writeLine(&b, format, "Source", evidence.Source.Kind+" "+evidence.Source.Ref)
	writeLine(&b, format, "Observation", evidence.Observation)
	return b.String(), nil
}

func readProjection(path string) (state.Projection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return state.Projection{}, fmt.Errorf("read competency projection: %w", err)
	}
	var projection state.Projection
	if err := yaml.Unmarshal(data, &projection); err != nil {
		return state.Projection{}, fmt.Errorf("parse competency projection: %w", err)
	}
	return projection, nil
}

func readAssessments(dir string) ([]state.Assessment, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	result := make([]state.Assessment, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var assessment state.Assessment
		if err := yaml.Unmarshal(data, &assessment); err != nil {
			return nil, err
		}
		result = append(result, assessment)
	}
	return result, nil
}

func readMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func readOptionalMap(path string) (map[string]any, error) {
	value, err := readMap(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return value, err
}

func writePersonaSection(b *strings.Builder, format Format, title string, persona map[string]any) {
	compact := map[string]any{}
	for _, key := range []string{"teaching", "analogyPolicy", "risks"} {
		if value, ok := persona[key]; ok {
			compact[key] = value
		}
	}
	writeMapSection(b, format, title, compact)
}

func writeMapSection(b *strings.Builder, format Format, title string, value map[string]any) {
	if format == Markdown {
		fmt.Fprintf(b, "\n## %s\n\n", title)
	} else {
		fmt.Fprintf(b, "\n%s\n", title)
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(b, "- %s: %v\n", key, value[key])
	}
}

func writeKV(b *strings.Builder, format Format, key, value string) {
	if value == "" {
		return
	}
	if format == Markdown {
		fmt.Fprintf(b, "- **%s:** %s\n", key, value)
	} else {
		fmt.Fprintf(b, "%s: %s\n", key, value)
	}
}

func writeLine(b *strings.Builder, format Format, key, value string) {
	writeKV(b, format, key, value)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any, fallback int) int {
	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	default:
		return fallback
	}
}
