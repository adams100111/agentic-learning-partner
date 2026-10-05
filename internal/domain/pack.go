package domain

import (
	"fmt"
	"sort"
	"strings"
)

type Pack struct {
	SchemaVersion   int               `yaml:"schemaVersion"`
	Domain          string            `yaml:"domain"`
	Version         string            `yaml:"version"`
	VerifiedAgainst map[string]string `yaml:"verifiedAgainst"`
	Competencies    []Competency      `yaml:"competencies"`
	Migrations      []Migration       `yaml:"migrations,omitempty"`
}

type Competency struct {
	ID                      string   `yaml:"id"`
	Name                    string   `yaml:"name"`
	Dimension               string   `yaml:"dimension"`
	Description             string   `yaml:"description"`
	Prerequisites           []string `yaml:"prerequisites,omitempty"`
	SharedScaffolds         []string `yaml:"sharedScaffolds,omitempty"`
	ProductionReadyRequires []string `yaml:"productionReadyRequires,omitempty"`
	FreshnessClass          string   `yaml:"freshnessClass"`
}

type Migration struct {
	From     string   `yaml:"from"`
	To       []string `yaml:"to,omitempty"`
	Strategy string   `yaml:"strategy"`
}

var validDimensions = map[string]struct{}{
	"recall": {}, "mental-model": {}, "idiomatic": {}, "runtime": {},
	"backend": {}, "database": {}, "testing": {}, "production": {},
	"architecture": {}, "tooling": {}, "performance": {},
}

var validFreshnessClasses = map[string]struct{}{
	"stable-concept":                    {},
	"version-sensitive-language-runtime": {},
	"ecosystem-choice":                  {},
	"operational-platform":              {},
	"security-sensitive":                {},
}

var validMigrationStrategies = map[string]struct{}{
	"rename":   {},
	"split":    {},
	"merge":    {},
	"reassess": {},
	"remove":   {},
}

func (p Pack) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("domain %q: unsupported schemaVersion %d", p.Domain, p.SchemaVersion)
	}
	if p.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if _, err := ParseVersion(p.Version); err != nil {
		return fmt.Errorf("domain %q version: %w", p.Domain, err)
	}
	if len(p.Competencies) == 0 {
		return fmt.Errorf("domain %q: at least one competency is required", p.Domain)
	}

	byID := make(map[string]Competency, len(p.Competencies))
	for _, competency := range p.Competencies {
		if competency.ID == "" {
			return fmt.Errorf("domain %q: competency id is required", p.Domain)
		}
		if !strings.HasPrefix(competency.ID, p.Domain+".") {
			return fmt.Errorf("competency %q must use domain prefix %q", competency.ID, p.Domain+".")
		}
		if _, exists := byID[competency.ID]; exists {
			return fmt.Errorf("duplicate competency id %q", competency.ID)
		}
		if competency.Name == "" || competency.Dimension == "" || competency.Description == "" {
			return fmt.Errorf("competency %q requires name, dimension, and description", competency.ID)
		}
		if _, ok := validDimensions[competency.Dimension]; !ok {
			return fmt.Errorf("competency %q has invalid dimension %q", competency.ID, competency.Dimension)
		}
		if _, ok := validFreshnessClasses[competency.FreshnessClass]; !ok {
			return fmt.Errorf("competency %q has invalid freshness class %q", competency.ID, competency.FreshnessClass)
		}
		byID[competency.ID] = competency
	}

	for _, competency := range p.Competencies {
		for _, prerequisite := range competency.Prerequisites {
			if _, ok := byID[prerequisite]; !ok {
				return fmt.Errorf("competency %q references unknown prerequisite %q", competency.ID, prerequisite)
			}
		}
		seenRequirement := map[string]struct{}{}
		for _, requirement := range competency.ProductionReadyRequires {
			if strings.TrimSpace(requirement) == "" {
				return fmt.Errorf("competency %q has an empty production-ready requirement", competency.ID)
			}
			if _, exists := seenRequirement[requirement]; exists {
				return fmt.Errorf("competency %q repeats production-ready requirement %q", competency.ID, requirement)
			}
			seenRequirement[requirement] = struct{}{}
		}
	}

	if err := validatePrerequisiteCycles(byID); err != nil {
		return err
	}

	seenMigration := map[string]struct{}{}
	for _, migration := range p.Migrations {
		if _, exists := seenMigration[migration.From]; exists {
			return fmt.Errorf("duplicate migration source %q", migration.From)
		}
		seenMigration[migration.From] = struct{}{}
		if migration.From == "" {
			return fmt.Errorf("domain %q: migration from is required", p.Domain)
		}
		if _, ok := validMigrationStrategies[migration.Strategy]; !ok {
			return fmt.Errorf("migration from %q has invalid strategy %q", migration.From, migration.Strategy)
		}
		if migration.Strategy != "remove" && len(migration.To) == 0 {
			return fmt.Errorf("migration from %q requires at least one target", migration.From)
		}
		for _, target := range migration.To {
			if _, ok := byID[target]; !ok {
				return fmt.Errorf("migration from %q references unknown target %q", migration.From, target)
			}
		}
	}

	return nil
}

func validatePrerequisiteCycles(byID map[string]Competency) error {
	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[string]int, len(byID))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("competency prerequisite cycle includes %q", id)
		case done:
			return nil
		}
		state[id] = visiting
		for _, dependency := range byID[id].Prerequisites {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = done
		return nil
	}
	for id := range byID {
		if state[id] == unseen {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Pack) CompetencyIDs() []string {
	ids := make([]string, 0, len(p.Competencies))
	for _, competency := range p.Competencies {
		ids = append(ids, competency.ID)
	}
	sort.Strings(ids)
	return ids
}

func (p Pack) HasCompetency(id string) bool {
	for _, competency := range p.Competencies {
		if competency.ID == id {
			return true
		}
	}
	return false
}

func (p Pack) Supports(versionRange string) (bool, error) {
	return Satisfies(p.Version, versionRange)
}
