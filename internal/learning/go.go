package learning

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	godomain "github.com/adams100111/agentic-learning-partner/domains/go"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type GoPlanner struct {
	Root      string
	Validator *workspace.Validator
	Registry  domain.Registry
}

func (p GoPlanner) Build(asOf time.Time, limit int) (ReviewQueue, Plan, error) {
	if p.Registry.List() == nil {
		p.Registry = domain.NewRegistry()
	}
	pack, err := p.Registry.Load("go")
	if err != nil {
		return ReviewQueue{}, Plan{}, err
	}
	projection, err := readProjection(filepath.Join(p.Root, "state", "competencies.yaml"))
	if err != nil && !os.IsNotExist(err) {
		return ReviewQueue{}, Plan{}, err
	}

	byID := map[string]state.ProjectedCompetency{}
	for _, item := range projection.Competencies {
		if item.Domain == "go" {
			byID[item.ID] = item
		}
	}

	queue := ReviewQueue{SchemaVersion: 1, AsOf: asOf.UTC().Format(time.RFC3339)}
	plan := Plan{SchemaVersion: 1, Domain: "go", AsOf: queue.AsOf}

	for _, competency := range pack.Competencies {
		current, known := byID[competency.ID]
		priority, reasons, due := reviewPriority(competency, current, known, asOf)
		if due {
			queue.Items = append(queue.Items, ReviewItem{
				Domain: "go", Competency: competency.ID, Priority: priority,
				Reasons: reasons, LastVerified: current.LastVerified,
			})
		}
		mode, reason := consumptionMode(current, known, due)
		if mode != "skip" || due {
			plan.Items = append(plan.Items, PlanItem{
				Competency: competency.ID, Mode: mode, Reason: reason, Priority: priority,
			})
		}
	}

	sort.Slice(queue.Items, func(i, j int) bool {
		if queue.Items[i].Priority == queue.Items[j].Priority {
			return queue.Items[i].Competency < queue.Items[j].Competency
		}
		return queue.Items[i].Priority > queue.Items[j].Priority
	})
	sort.Slice(plan.Items, func(i, j int) bool {
		if plan.Items[i].Priority == plan.Items[j].Priority {
			return plan.Items[i].Competency < plan.Items[j].Competency
		}
		return plan.Items[i].Priority > plan.Items[j].Priority
	})
	if limit > 0 && len(plan.Items) > limit {
		plan.Items = plan.Items[:limit]
	}

	if err := p.persist(queue, plan); err != nil {
		return ReviewQueue{}, Plan{}, err
	}
	return queue, plan, nil
}

func (p GoPlanner) Diagnostic(limit int) (Diagnostic, error) {
	data, err := godomain.Files.ReadFile("diagnostic.yaml")
	if err != nil {
		return Diagnostic{}, fmt.Errorf("read Go diagnostic catalog: %w", err)
	}
	var diagnostic Diagnostic
	if err := yaml.Unmarshal(data, &diagnostic); err != nil {
		return Diagnostic{}, fmt.Errorf("parse Go diagnostic catalog: %w", err)
	}
	projection, _ := readProjection(filepath.Join(p.Root, "state", "competencies.yaml"))
	byID := map[string]state.ProjectedCompetency{}
	for _, item := range projection.Competencies {
		byID[item.ID] = item
	}

	var selected []DiagnosticActivity
	for _, activity := range diagnostic.Activities {
		if shouldDiagnose(activity, byID) {
			selected = append(selected, activity)
		}
	}
	if limit > 0 && len(selected) > limit {
		selected = selected[:limit]
	}
	diagnostic.Activities = selected
	return diagnostic, nil
}

func reviewPriority(competency domain.Competency, current state.ProjectedCompetency, known bool, asOf time.Time) (int, []string, bool) {
	if !known {
		return 8, []string{"no demonstrated competency assessment"}, true
	}
	score := 0
	var reasons []string
	switch current.Confidence {
	case "low":
		score += 4
		reasons = append(reasons, "low confidence")
	case "medium":
		score += 2
		reasons = append(reasons, "medium confidence")
	}
	if current.NeedsReassessment {
		score += 5
		reasons = append(reasons, "contradictory assessments require reassessment")
	}
	switch current.Level {
	case "unknown":
		score += 5
		reasons = append(reasons, "level unknown")
	case "rusty":
		score += 4
		reasons = append(reasons, "level rusty")
	case "functional":
		score += 2
		reasons = append(reasons, "functional level benefits from retrieval/challenge")
	}
	if competency.Dimension == "runtime" || competency.Dimension == "production" {
		score += 1
		reasons = append(reasons, "high-consequence competency")
	}
	if competency.FreshnessClass == "security-sensitive" {
		score += 3
		reasons = append(reasons, "security-sensitive")
	}

	if current.LastVerified != "" {
		if verified, err := time.Parse(time.RFC3339, current.LastVerified); err == nil {
			budget := freshnessBudget(competency.FreshnessClass)
			if asOf.Sub(verified) > budget {
				score += 3
				reasons = append(reasons, "evidence freshness budget exceeded")
			}
		}
	}
	return score, reasons, score > 0
}

func freshnessBudget(class string) time.Duration {
	switch class {
	case "security-sensitive":
		return 7 * 24 * time.Hour
	case "operational-platform":
		return 30 * 24 * time.Hour
	case "ecosystem-choice":
		return 90 * 24 * time.Hour
	case "version-sensitive-language-runtime":
		return 120 * 24 * time.Hour
	default:
		return 365 * 24 * time.Hour
	}
}

func consumptionMode(current state.ProjectedCompetency, known, due bool) (string, string) {
	if !known || current.Level == "unknown" || current.Level == "rusty" {
		return "full", "insufficient or rusty demonstrated evidence"
	}
	if current.NeedsReassessment {
		return "challenge-only", "contradictory evidence should be resolved with targeted challenge"
	}
	switch current.Level {
	case "functional":
		return "challenge-only", "functional competence should be strengthened through retrieval/implementation rather than lecture"
	case "strong":
		if due {
			return "challenge-only", "strong competence is due for targeted verification, not full reteaching"
		}
		return "skip", "strong and sufficiently fresh"
	case "production-ready":
		if due {
			return "skim", "production-ready competence only needs freshness confirmation unless contradictory evidence appears"
		}
		return "skip", "production-ready and sufficiently fresh"
	default:
		return "full", "no reliable demonstrated level"
	}
}

func shouldDiagnose(activity DiagnosticActivity, byID map[string]state.ProjectedCompetency) bool {
	for _, id := range activity.Competencies {
		current, ok := byID[id]
		if !ok || current.Level == "unknown" || current.Level == "rusty" || current.Level == "functional" || current.NeedsReassessment {
			return true
		}
	}
	return false
}

func (p GoPlanner) persist(queue ReviewQueue, plan Plan) error {
	if p.Validator == nil {
		return fmt.Errorf("workspace validator is required")
	}
	if err := os.MkdirAll(filepath.Join(p.Root, "state"), 0o755); err != nil {
		return err
	}
	queueData, err := yaml.Marshal(queue)
	if err != nil {
		return err
	}
	if issue := p.Validator.ValidateDocument("review-queue.schema.json", "state/review-queue.yaml", queueData); issue != nil {
		return issue
	}
	planData, err := yaml.Marshal(plan)
	if err != nil {
		return err
	}
	if issue := p.Validator.ValidateDocument("learning-plan.schema.json", "state/learning-plan.yaml", planData); issue != nil {
		return issue
	}
	if err := os.WriteFile(filepath.Join(p.Root, "state", "review-queue.yaml"), queueData, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(p.Root, "state", "learning-plan.yaml"), planData, 0o644); err != nil {
		return err
	}
	return nil
}

func readProjection(path string) (state.Projection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return state.Projection{}, err
	}
	var projection state.Projection
	if err := yaml.Unmarshal(data, &projection); err != nil {
		return state.Projection{}, err
	}
	return projection, nil
}

func ExplainPlan(plan Plan) string {
	var b strings.Builder
	for _, item := range plan.Items {
		fmt.Fprintf(&b, "%s: %s — %s\n", item.Competency, item.Mode, item.Reason)
	}
	return b.String()
}
