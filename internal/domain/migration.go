package domain

import "fmt"

type CompetencyResolution struct {
	From                 string
	Targets              []string
	Strategy             string
	RequiresReassessment bool
	Removed              bool
}

func (p Pack) ResolveCompetency(id string) (CompetencyResolution, error) {
	if p.HasCompetency(id) {
		return CompetencyResolution{From: id, Targets: []string{id}, Strategy: "unchanged"}, nil
	}
	for _, migration := range p.Migrations {
		if migration.From != id {
			continue
		}
		resolution := CompetencyResolution{
			From:     id,
			Targets:  append([]string(nil), migration.To...),
			Strategy: migration.Strategy,
		}
		switch migration.Strategy {
		case "rename":
			if len(migration.To) != 1 {
				return CompetencyResolution{}, fmt.Errorf("rename migration from %q requires exactly one target", id)
			}
		case "merge":
			if len(migration.To) != 1 {
				return CompetencyResolution{}, fmt.Errorf("merge migration from %q requires exactly one target", id)
			}
			resolution.RequiresReassessment = true
		case "split":
			if len(migration.To) < 2 {
				return CompetencyResolution{}, fmt.Errorf("split migration from %q requires at least two targets", id)
			}
			resolution.RequiresReassessment = true
		case "reassess":
			if len(migration.To) == 0 {
				return CompetencyResolution{}, fmt.Errorf("reassess migration from %q requires targets", id)
			}
			resolution.RequiresReassessment = true
		case "remove":
			if len(migration.To) != 0 {
				return CompetencyResolution{}, fmt.Errorf("remove migration from %q cannot have targets", id)
			}
			resolution.Removed = true
		default:
			return CompetencyResolution{}, fmt.Errorf("unsupported migration strategy %q for %q", migration.Strategy, id)
		}
		return resolution, nil
	}
	return CompetencyResolution{}, fmt.Errorf("competency %q is neither current nor covered by a migration", id)
}
