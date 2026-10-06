package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/learning"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

func (a App) runPlan(args []string) int {
	if len(args) == 0 || args[0] != "build" {
		fmt.Fprintln(a.ErrOut, "usage: alp plan build --domain go [--limit N] [--as-of RFC3339] [--workspace PATH]")
		return 2
	}
	flags := flag.NewFlagSet("plan build", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	domainName := flags.String("domain", "go", "domain")
	limit := flags.Int("limit", 10, "maximum plan items")
	asOfText := flags.String("as-of", "", "planning timestamp in RFC3339")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *domainName != "go" {
		fmt.Fprintf(a.ErrOut, "domain %q does not yet provide an adaptive planner\n", *domainName)
		return 1
	}
	_, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}
	asOf := time.Now().UTC()
	if *asOfText != "" {
		parsed, err := time.Parse(time.RFC3339, *asOfText)
		if err != nil {
			fmt.Fprintf(a.ErrOut, "invalid --as-of: %v\n", err)
			return 2
		}
		asOf = parsed
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	queue, plan, err := (learning.GoPlanner{Root: info.Path, Validator: validator, Registry: domain.NewRegistry()}).Build(asOf, *limit)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintf(a.Out, "review items: %d\nplan items: %d\n", len(queue.Items), len(plan.Items))
	fmt.Fprint(a.Out, learning.ExplainPlan(plan))
	return 0
}

func (a App) runDiagnostic(args []string) int {
	flags := flag.NewFlagSet("diagnostic", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	domainName := flags.String("domain", "go", "domain")
	limit := flags.Int("limit", 15, "maximum activities")
	format := flags.String("format", "yaml", "output format: yaml or json")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *domainName != "go" {
		fmt.Fprintf(a.ErrOut, "domain %q does not yet provide a diagnostic catalog\n", *domainName)
		return 1
	}
	_, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}
	diagnostic, err := (learning.GoPlanner{Root: info.Path, Registry: domain.NewRegistry()}).Diagnostic(*limit)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	var output []byte
	switch *format {
	case "yaml":
		output, err = yaml.Marshal(diagnostic)
	case "json":
		output, err = json.MarshalIndent(diagnostic, "", "  ")
	default:
		fmt.Fprintf(a.ErrOut, "unsupported diagnostic format %q\n", *format)
		return 2
	}
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprintln(a.Out, string(output))
	return 0
}
