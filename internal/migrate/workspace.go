package migrate

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"context"
	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

const CurrentWorkspaceSchema = workspace.CurrentSchemaVersion

type Step struct {
	From        int
	To          int
	Description string
	Apply       func(root string) error
}

type PlannedStep struct {
	From        int
	To          int
	Description string
}

type Plan struct {
	Current int
	Target  int
	Steps   []PlannedStep
}

func (p Plan) Empty() bool { return len(p.Steps) == 0 }

type Migrator struct {
	TargetVersion int
	Steps         map[int]Step
	Validator     *workspace.Validator
	Rebuild       func(root string) error
}

func NewWorkspaceMigrator(validator *workspace.Validator) Migrator {
	return Migrator{
		TargetVersion: CurrentWorkspaceSchema,
		Steps: map[int]Step{
			1: {From: 1, To: 2, Description: "assign immutable workspace identity", Apply: migrateWorkspaceV1ToV2},
		},
		Validator: validator,
	}
}

func (m Migrator) Plan(root string) (Plan, error) {
	current, err := workspaceSchemaVersion(root)
	if err != nil {
		return Plan{}, err
	}
	target := m.TargetVersion
	if target == 0 {
		target = CurrentWorkspaceSchema
	}
	if current > target {
		return Plan{}, fmt.Errorf("workspace schema %d is newer than supported target %d", current, target)
	}

	plan := Plan{Current: current, Target: target}
	for version := current; version < target; version++ {
		step, ok := m.Steps[version]
		if !ok {
			return Plan{}, fmt.Errorf("no migration defined from workspace schema %d to %d", version, version+1)
		}
		if step.From != version || step.To != version+1 {
			return Plan{}, fmt.Errorf("invalid migration registration at %d: step must be %d -> %d", version, version, version+1)
		}
		plan.Steps = append(plan.Steps, PlannedStep{From: step.From, To: step.To, Description: step.Description})
	}
	return plan, nil
}

func (m Migrator) ApplyStaged(root string) (Plan, error) {
	plan, err := m.Plan(root)
	if err != nil {
		return Plan{}, err
	}
	if plan.Empty() {
		return plan, nil
	}
	for _, planned := range plan.Steps {
		step := m.Steps[planned.From]
		if step.Apply == nil {
			return Plan{}, fmt.Errorf("migration %d -> %d has no apply function", step.From, step.To)
		}
		if err := step.Apply(root); err != nil {
			return Plan{}, fmt.Errorf("apply workspace migration %d -> %d: %w", step.From, step.To, err)
		}
		if err := setWorkspaceSchemaVersion(root, step.To); err != nil {
			return Plan{}, err
		}
	}
	if m.Rebuild != nil {
		if err := m.Rebuild(root); err != nil {
			return Plan{}, fmt.Errorf("rebuild derived state after migration: %w", err)
		}
	}
	if m.Validator != nil {
		if issues := m.Validator.ValidateWorkspace(root); len(issues) > 0 {
			return Plan{}, fmt.Errorf("migrated workspace is invalid: %s", issues[0].Error())
		}
	}
	return plan, nil
}

func (m Migrator) Apply(root string, expectedRevision string) (Plan, error) {
	if strings.TrimSpace(expectedRevision) == "" {
		return Plan{}, errors.New("expected workspace revision is required")
	}
	info, err := workspace.Inspect(root)
	if err != nil {
		return Plan{}, err
	}
	if info.Revision != expectedRevision {
		return Plan{}, fmt.Errorf("workspace revision changed: expected %s, found %s", expectedRevision, info.Revision)
	}
	clean, err := gitClean(root)
	if err != nil {
		return Plan{}, err
	}
	if !clean {
		return Plan{}, errors.New("workspace must be clean before migration; commit or stash changes first")
	}

	plan, err := m.Plan(root)
	if err != nil {
		return Plan{}, err
	}
	if plan.Empty() {
		return plan, nil
	}

	for _, planned := range plan.Steps {
		step := m.Steps[planned.From]
		if step.Apply == nil {
			return Plan{}, fmt.Errorf("migration %d -> %d has no apply function", step.From, step.To)
		}
		if err := step.Apply(root); err != nil {
			return Plan{}, fmt.Errorf("apply workspace migration %d -> %d: %w", step.From, step.To, err)
		}
		if err := setWorkspaceSchemaVersion(root, step.To); err != nil {
			return Plan{}, err
		}
	}
	if m.Rebuild != nil {
		if err := m.Rebuild(root); err != nil {
			return Plan{}, fmt.Errorf("rebuild derived state after migration: %w", err)
		}
	}
	if m.Validator != nil {
		if issues := m.Validator.ValidateWorkspace(root); len(issues) > 0 {
			return Plan{}, fmt.Errorf("migrated workspace is invalid: %s", issues[0].Error())
		}
	}
	return plan, nil
}

func workspaceSchemaVersion(root string) (int, error) {
	path := filepath.Join(root, "workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read workspace manifest: %w", err)
	}
	var manifest struct {
		SchemaVersion int `yaml:"schemaVersion"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return 0, fmt.Errorf("parse workspace manifest: %w", err)
	}
	if manifest.SchemaVersion <= 0 {
		return 0, errors.New("workspace manifest schemaVersion must be a positive integer")
	}
	return manifest.SchemaVersion, nil
}

func setWorkspaceSchemaVersion(root string, version int) error {
	path := filepath.Join(root, "workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return err
	}
	manifest["schemaVersion"] = version
	output, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("write workspace schema version %s: %w", strconv.Itoa(version), err)
	}
	return nil
}

func gitClean(root string) (bool, error) {
	command := gitexec.Command(context.Background(), root, "status", "--porcelain")
	output, err := command.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("inspect workspace Git status: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)) == "", nil
}

func SortedSteps(steps map[int]Step) []Step {
	result := make([]Step, 0, len(steps))
	for _, step := range steps {
		result = append(result, step)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].From < result[j].From })
	return result
}

func migrateWorkspaceV1ToV2(root string) error {
	path := filepath.Join(root, "workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read workspace manifest: %w", err)
	}
	var manifest map[string]any
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse workspace manifest: %w", err)
	}
	if value, _ := manifest["workspaceId"].(string); value == "" {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return fmt.Errorf("generate workspace id: %w", err)
		}
		manifest["workspaceId"] = "ws_" + hex.EncodeToString(random[:])
	}
	output, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal workspace manifest: %w", err)
	}
	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("write workspace manifest: %w", err)
	}
	return nil
}
