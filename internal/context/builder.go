package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type Builder struct {
	Root      string
	Validator *workspace.Validator
}

func (b Builder) Build(request Request) (Bundle, error) {
	if request.Task == "" {
		return Bundle{}, errors.New("context task is required")
	}
	if b.Validator == nil {
		return Bundle{}, errors.New("workspace validator is required")
	}

	bundle := Bundle{
		SchemaVersion: 1,
		Task:          request.Task,
		Learner:       map[string]any{},
		Focus:         map[string]any{"task": request.Task},
		Persona:       map[string]any{},
		Competencies:  map[string]any{},
	}
	if request.Domain != "" {
		bundle.Domain = &request.Domain
		bundle.Focus["domain"] = request.Domain
	}
	if request.Competency != "" {
		bundle.Focus["competency"] = request.Competency
	}

	profile, err := readMap(filepath.Join(b.Root, "profile", "profile.yaml"))
	if err != nil {
		return Bundle{}, fmt.Errorf("load learner profile: %w", err)
	}
	bundle.Learner = projectProfile(profile)
	bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
		Ref: "profile/profile.yaml", Reason: "durable learner identity and compact experience/preferences",
	})

	globalPersona, err := readOptionalMap(filepath.Join(b.Root, "personas", "global.yaml"))
	if err != nil {
		return Bundle{}, err
	}
	if globalPersona != nil {
		bundle.Persona["global"] = projectPersona(globalPersona)
		bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
			Ref: "personas/global.yaml", Reason: "global teaching behavior applies to every domain",
		})
	} else {
		bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
			Ref: "personas/global.yaml", Reason: "not present",
		})
	}

	if request.Domain != "" {
		ref := filepath.ToSlash(filepath.Join("personas", "domains", request.Domain+".yaml"))
		domainPersona, err := readOptionalMap(filepath.Join(b.Root, filepath.FromSlash(ref)))
		if err != nil {
			return Bundle{}, err
		}
		if domainPersona != nil {
			bundle.Persona["domain"] = projectPersona(domainPersona)
			bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
				Ref: ref, Reason: "domain-specific teaching overrides for requested domain",
			})
		} else {
			bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
				Ref: ref, Reason: "domain persona not present",
			})
		}
	}

	projection, err := readOptionalMap(filepath.Join(b.Root, "state", "competencies.yaml"))
	if err != nil {
		return Bundle{}, err
	}
	if projection != nil {
		relevant := projectCompetencies(projection, request.Domain, request.Competency)
		if len(relevant) > 0 {
			bundle.Competencies["items"] = relevant
			bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
				Ref: "state/competencies.yaml", Reason: "current competency projection relevant to requested task/domain",
			})
		} else {
			bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
				Ref: "state/competencies.yaml", Reason: "no projected competencies matched requested focus",
			})
		}
	} else {
		bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
			Ref: "state/competencies.yaml", Reason: "projection not present",
		})
	}

	review, err := readOptionalMap(filepath.Join(b.Root, "state", "review-queue.yaml"))
	if err != nil {
		return Bundle{}, err
	}
	if review != nil {
		bundle.Reinforcement = projectReview(review, request.Domain)
		if len(bundle.Reinforcement) > 0 {
			bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
				Ref: "state/review-queue.yaml", Reason: "due reinforcement relevant to requested domain",
			})
		} else {
			bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
				Ref: "state/review-queue.yaml", Reason: "no relevant due reinforcement",
			})
		}
	}

	project, err := readOptionalMap(filepath.Join(b.Root, "state", "current-project.yaml"))
	if err != nil {
		return Bundle{}, err
	}
	if project != nil && (request.Domain == "" || stringValue(project["domain"]) == request.Domain) {
		bundle.Project = project
		bundle.IncludedSources = append(bundle.IncludedSources, SourceSelection{
			Ref: "state/current-project.yaml", Reason: "current project is relevant to requested domain",
		})
	} else if project != nil {
		bundle.OmittedSources = append(bundle.OmittedSources, SourceSelection{
			Ref: "state/current-project.yaml", Reason: "current project belongs to another domain",
		})
	}

	bundle.OmittedSources = append(bundle.OmittedSources,
		SourceSelection{Ref: "evidence/", Reason: "full evidence history is excluded from ordinary context; load only for assessment/audit"},
		SourceSelection{Ref: "assessments/", Reason: "full assessment history is excluded from ordinary context; projection is sufficient"},
	)

	sort.Slice(bundle.IncludedSources, func(i, j int) bool { return bundle.IncludedSources[i].Ref < bundle.IncludedSources[j].Ref })
	sort.Slice(bundle.OmittedSources, func(i, j int) bool { return bundle.OmittedSources[i].Ref < bundle.OmittedSources[j].Ref })

	encoded, err := json.Marshal(bundleWithoutEstimate(bundle))
	if err != nil {
		return Bundle{}, fmt.Errorf("estimate context size: %w", err)
	}
	estimate := (len(encoded) + 3) / 4
	bundle.EstimatedTokens = &estimate

	data, err := yaml.Marshal(bundle)
	if err != nil {
		return Bundle{}, fmt.Errorf("marshal context bundle: %w", err)
	}
	if issue := b.Validator.ValidateDocument("context-bundle.schema.json", "context.yaml", data); issue != nil {
		return Bundle{}, issue
	}
	return bundle, nil
}

func projectProfile(profile map[string]any) map[string]any {
	result := map[string]any{}
	if learner, ok := profile["learner"]; ok {
		result["identity"] = learner
	}
	if experience, ok := profile["experience"].(map[string]any); ok {
		type ranked struct {
			name string
			rank int
			data any
		}
		var items []ranked
		for name, raw := range experience {
			rank := 999
			if entry, ok := raw.(map[string]any); ok {
				if value, ok := intValue(entry["relativeRank"]); ok {
					rank = value
				}
			}
			items = append(items, ranked{name: name, rank: rank, data: raw})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].rank == items[j].rank {
				return items[i].name < items[j].name
			}
			return items[i].rank < items[j].rank
		})
		compact := make([]map[string]any, 0, len(items))
		for _, item := range items {
			entry := map[string]any{"stack": item.name}
			if data, ok := item.data.(map[string]any); ok {
				for _, key := range []string{"level", "relativeRank", "frameworks", "newest"} {
					if value, exists := data[key]; exists {
						entry[key] = value
					}
				}
			}
			compact = append(compact, entry)
		}
		result["experience"] = compact
	}
	if preferences, ok := profile["preferences"]; ok {
		result["preferences"] = preferences
	}
	return result
}

func projectPersona(persona map[string]any) map[string]any {
	result := map[string]any{}
	for _, key := range []string{"teaching", "analogyPolicy", "risks"} {
		if value, ok := persona[key]; ok {
			result[key] = value
		}
	}
	return result
}

func projectCompetencies(projection map[string]any, domain, competency string) []any {
	raw, _ := projection["competencies"].([]any)
	result := make([]any, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if domain != "" && stringValue(entry["domain"]) != domain {
			continue
		}
		if competency != "" && stringValue(entry["id"]) != competency {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func projectReview(review map[string]any, domain string) []string {
	raw, _ := review["items"].([]any)
	var result []string
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if domain != "" && stringValue(entry["domain"]) != domain {
			continue
		}
		if id := stringValue(entry["competency"]); id != "" {
			result = append(result, id)
		}
	}
	sort.Strings(result)
	return result
}

func readMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return value, nil
}

func readOptionalMap(path string) (map[string]any, error) {
	value, err := readMap(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return value, err
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int64:
		return int(number), true
	case float64:
		return int(number), true
	default:
		return 0, false
	}
}

func bundleWithoutEstimate(bundle Bundle) Bundle {
	bundle.EstimatedTokens = nil
	return bundle
}
