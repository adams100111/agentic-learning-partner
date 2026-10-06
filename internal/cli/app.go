package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	contextbundle "github.com/adams100111/agentic-learning-partner/internal/context"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/migrate"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/view"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type App struct {
	Out    io.Writer
	ErrOut io.Writer
	Getwd  func() (string, error)
}

func New() App {
	return App{Out: os.Stdout, ErrOut: os.Stderr, Getwd: os.Getwd}
}

func (a App) Run(args []string) int {
	if len(args) == 0 {
		a.usage()
		return 2
	}

	switch args[0] {
	case "validate":
		return a.runValidate(args[1:])
	case "workspace":
		if len(args) < 2 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace <init|clone|list|use|status|sync|export|verify|restore|move|acknowledge-privacy|check|migrate> [options]")
			return 2
		}
		if args[1] == "check" {
			return a.runWorkspaceCheck(args[2:])
		}
		if args[1] == "migrate" {
			return a.runWorkspaceMigrate(args[2:])
		}
		return a.runWorkspaceLifecycle(args[1], args[2:])
	case "domain":
		return a.runDomain(args[1:])
	case "context":
		return a.runContext(args[1:])
	case "persona":
		return a.runPersona(args[1:])
	case "status":
		return a.runStatus(args[1:])
	case "competency":
		return a.runCompetency(args[1:])
	case "evidence":
		return a.runEvidence(args[1:])
	case "assessment":
		return a.runAssessment(args[1:])
	case "state":
		return a.runState(args[1:])
	case "plan":
		return a.runPlan(args[1:])
	case "diagnostic":
		return a.runDiagnostic(args[1:])
	default:
		fmt.Fprintf(a.ErrOut, "unknown command %q\n", args[0])
		a.usage()
		return 2
	}
}

func (a App) viewArgs(name string, args []string) (string, view.Format, bool) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	format := flags.String("format", "text", "output format: text or markdown")
	if err := flags.Parse(args); err != nil {
		return "", "", false
	}
	resolution, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return "", "", false
	}
	_ = resolution
	switch *format {
	case "text":
		return info.Path, view.Text, true
	case "markdown":
		return info.Path, view.Markdown, true
	default:
		fmt.Fprintf(a.ErrOut, "unsupported view format %q\n", *format)
		return "", "", false
	}
}

func (a App) runPersona(args []string) int {
	if len(args) == 0 || args[0] != "show" {
		fmt.Fprintln(a.ErrOut, "usage: alp persona show [--domain DOMAIN] [--format text|markdown] [--workspace PATH]")
		return 2
	}
	flags := flag.NewFlagSet("persona show", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	domainName := flags.String("domain", "", "domain")
	format := flags.String("format", "text", "output format: text or markdown")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	_, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}
	outputFormat, ok := parseViewFormat(*format)
	if !ok {
		fmt.Fprintf(a.ErrOut, "unsupported view format %q\n", *format)
		return 2
	}
	output, err := (view.Renderer{Root: info.Path}).Persona(*domainName, outputFormat)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprint(a.Out, output)
	return 0
}

func (a App) runStatus(args []string) int {
	root, format, ok := a.viewArgs("status", args)
	if !ok {
		return 1
	}
	output, err := (view.Renderer{Root: root}).Status(format)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprint(a.Out, output)
	return 0
}

func (a App) runCompetency(args []string) int {
	if len(args) == 0 || args[0] != "show" || len(args) < 2 {
		fmt.Fprintln(a.ErrOut, "usage: alp competency show <id> [--format text|markdown] [--workspace PATH]")
		return 2
	}
	id := args[1]
	root, format, ok := a.viewArgs("competency show", args[2:])
	if !ok {
		return 1
	}
	output, err := (view.Renderer{Root: root}).Competency(id, format)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	fmt.Fprint(a.Out, output)
	return 0
}

func (a App) runEvidence(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.ErrOut, "usage: alp evidence <show|add> ...")
		return 2
	}
	switch args[0] {
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(a.ErrOut, "usage: alp evidence show <id> [--format text|markdown] [--workspace PATH]")
			return 2
		}
		id := args[1]
		root, format, ok := a.viewArgs("evidence show", args[2:])
		if !ok {
			return 1
		}
		output, err := (view.Renderer{Root: root}).Evidence(id, format)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprint(a.Out, output)
		return 0
	case "add":
		flags := flag.NewFlagSet("evidence add", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		explicit := flags.String("workspace", "", "learner workspace path")
		file := flags.String("file", "", "evidence YAML or JSON file")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if *file == "" {
			fmt.Fprintln(a.ErrOut, "--file is required")
			return 2
		}
		return a.appendEvidence(*explicit, *file)
	default:
		fmt.Fprintf(a.ErrOut, "unknown evidence command %q\n", args[0])
		return 2
	}
}

func (a App) runAssessment(args []string) int {
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(a.ErrOut, "usage: alp assessment add --file FILE [--workspace PATH]")
		return 2
	}
	flags := flag.NewFlagSet("assessment add", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	file := flags.String("file", "", "assessment YAML or JSON file")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *file == "" {
		fmt.Fprintln(a.ErrOut, "--file is required")
		return 2
	}
	return a.appendAssessment(*explicit, *file)
}

func (a App) runState(args []string) int {
	if len(args) == 0 || args[0] != "rebuild" {
		fmt.Fprintln(a.ErrOut, "usage: alp state rebuild [--workspace PATH]")
		return 2
	}
	flags := flag.NewFlagSet("state rebuild", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	return a.rebuildState(*explicit)
}

func parseViewFormat(value string) (view.Format, bool) {
	switch value {
	case "text":
		return view.Text, true
	case "markdown":
		return view.Markdown, true
	default:
		return "", false
	}
}

func (a App) runContext(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.ErrOut, "usage: alp context <build|inspect> --task TASK [--domain DOMAIN] [--competency ID] [--workspace PATH]")
		return 2
	}
	command := args[0]
	flags := flag.NewFlagSet("context "+command, flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	task := flags.String("task", "", "agent task")
	domainName := flags.String("domain", "", "domain")
	competency := flags.String("competency", "", "competency id")
	format := flags.String("format", "yaml", "output format: yaml or json")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	resolution, info, ok := a.resolveAndInspect(*explicit)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	bundle, err := (contextbundle.Builder{Root: info.Path, Validator: validator}).Build(contextbundle.Request{
		Task: *task, Domain: *domainName, Competency: *competency, Inspect: command == "inspect",
	})
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}

	switch command {
	case "build":
		var output []byte
		if *format == "json" {
			output, err = json.MarshalIndent(bundle, "", "  ")
		} else if *format == "yaml" {
			output, err = yaml.Marshal(bundle)
		} else {
			fmt.Fprintf(a.ErrOut, "unsupported context format %q\n", *format)
			return 2
		}
		if err != nil {
			fmt.Fprintf(a.ErrOut, "encode context: %v\n", err)
			return 1
		}
		fmt.Fprintln(a.Out, string(output))
		return 0
	case "inspect":
		fmt.Fprintf(a.Out, "workspace: %s\nsource: %s\n", info.Path, resolution.Source)
		if bundle.EstimatedTokens != nil {
			fmt.Fprintf(a.Out, "estimated tokens: %d\n", *bundle.EstimatedTokens)
		}
		fmt.Fprintln(a.Out, "included:")
		for _, source := range bundle.IncludedSources {
			fmt.Fprintf(a.Out, "  - %s: %s\n", source.Ref, source.Reason)
		}
		fmt.Fprintln(a.Out, "omitted:")
		for _, source := range bundle.OmittedSources {
			fmt.Fprintf(a.Out, "  - %s: %s\n", source.Ref, source.Reason)
		}
		return 0
	default:
		fmt.Fprintf(a.ErrOut, "unknown context command %q\n", command)
		return 2
	}
}

func (a App) runDomain(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.ErrOut, "usage: alp domain <list|info>")
		return 2
	}

	registry := domain.NewRegistry()
	switch args[0] {
	case "list":
		for _, name := range registry.List() {
			fmt.Fprintln(a.Out, name)
		}
		return 0
	case "info":
		if len(args) != 2 {
			fmt.Fprintln(a.ErrOut, "usage: alp domain info <name>")
			return 2
		}
		pack, err := registry.Load(args[1])
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "domain: %s\nversion: %s\ncompetencies: %d\n", pack.Domain, pack.Version, len(pack.Competencies))
		return 0
	default:
		fmt.Fprintf(a.ErrOut, "unknown domain command %q\n", args[0])
		return 2
	}
}

func (a App) runValidate(args []string) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	resolution, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}

	fmt.Fprintf(a.Out, "valid workspace: %s\nsource: %s\nrevision: %s\n", info.Path, resolution.Source, info.Revision)
	return 0
}

func (a App) runWorkspaceMigrate(args []string) int {
	flags := flag.NewFlagSet("workspace migrate", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	dryRun := flags.Bool("dry-run", false, "show migration plan without writing")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	resolution, info, ok := a.resolveAndInspect(*explicit)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	migrator := migrate.NewWorkspaceMigrator(validator)
	migrator.Rebuild = func(root string) error {
		_, err := (state.Store{Root: root, Validator: validator}).RebuildProjection()
		return err
	}

	if *dryRun {
		plan, err := migrator.Plan(info.Path)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "workspace: %s\nsource: %s\ncurrent schema: %d\ntarget schema: %d\n", info.Path, resolution.Source, plan.Current, plan.Target)
		for _, step := range plan.Steps {
			fmt.Fprintf(a.Out, "%d -> %d: %s\n", step.From, step.To, step.Description)
		}
		if plan.Empty() {
			fmt.Fprintln(a.Out, "migration: none")
		}
		return 0
	}

	plan, err := migrator.Apply(info.Path, info.Revision)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	if plan.Empty() {
		fmt.Fprintln(a.Out, "workspace already uses the current schema")
		return 0
	}
	fmt.Fprintf(a.Out, "migrated workspace from schema %d to %d\n", plan.Current, plan.Target)
	return 0
}

func (a App) runWorkspaceCheck(args []string) int {
	flags := flag.NewFlagSet("workspace check", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	resolution, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}

	fmt.Fprintf(a.Out, "workspace: %s\nsource: %s\nrevision: %s\nschema: compatible\n", info.Path, resolution.Source, info.Revision)
	return 0
}

func (a App) checkWorkspace(explicit string) (workspace.Resolution, workspace.Info, bool) {
	resolution, info, ok := a.resolveAndInspect(explicit)
	if !ok {
		return workspace.Resolution{}, workspace.Info{}, false
	}

	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return workspace.Resolution{}, workspace.Info{}, false
	}
	issues := validator.ValidateWorkspace(info.Path)
	if len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintln(a.ErrOut, issue.Error())
		}
		return workspace.Resolution{}, workspace.Info{}, false
	}
	return resolution, info, true
}

func (a App) resolveAndInspect(explicit string) (workspace.Resolution, workspace.Info, bool) {
	start := ""
	if a.Getwd != nil {
		if value, err := a.Getwd(); err == nil {
			start = value
		}
	}

	resolution, err := workspace.NewResolver().Resolve(explicit, start)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return workspace.Resolution{}, workspace.Info{}, false
	}
	info, err := workspace.Inspect(resolution.Path)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return workspace.Resolution{}, workspace.Info{}, false
	}
	return resolution, info, true
}

func (a App) usage() {
	fmt.Fprintln(a.ErrOut, "usage: alp <command>")
	fmt.Fprintln(a.ErrOut, "commands: validate, workspace init, workspace clone, workspace list, workspace use, workspace status, workspace sync, workspace export, workspace verify, workspace restore, workspace move, workspace check, workspace migrate, domain list, domain info, context build, context inspect, persona show, status, competency show, evidence show, evidence add, assessment add, state rebuild, plan build, diagnostic")
}
