package cli

import (
	"fmt"
	"os"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

func (a App) appendEvidence(explicitWorkspace, file string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(a.ErrOut, "read evidence input: %v\n", err)
		return 1
	}
	var record state.Evidence
	if err := yaml.Unmarshal(data, &record); err != nil {
		fmt.Fprintf(a.ErrOut, "parse evidence input: %v\n", err)
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	store := state.Store{
		Root: info.Path, Catalog: domain.NewRegistry(), Validator: validator,
	}
	record, err = store.AppendEvidence(info.Revision, record)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintln(a.Out, record.ID)
	return 0
}

func (a App) appendAssessment(explicitWorkspace, file string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(a.ErrOut, "read assessment input: %v\n", err)
		return 1
	}
	var record state.Assessment
	if err := yaml.Unmarshal(data, &record); err != nil {
		fmt.Fprintf(a.ErrOut, "parse assessment input: %v\n", err)
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	store := state.Store{
		Root: info.Path, Catalog: domain.NewRegistry(), Validator: validator,
	}
	record, err = store.AppendAssessment(info.Revision, record)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintln(a.Out, record.ID)
	return 0
}

func (a App) rebuildState(explicitWorkspace string) int {
	_, info, ok := a.checkWorkspace(explicitWorkspace)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	projection, err := (state.Store{
		Root: info.Path, Catalog: domain.NewRegistry(), Validator: validator,
	}).RebuildProjection()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintf(a.Out, "rebuilt %d competencies\n", len(projection.Competencies))
	return 0
}
