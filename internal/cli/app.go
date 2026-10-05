package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/migrate"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
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
		if len(args) > 1 && args[1] == "check" {
			return a.runWorkspaceCheck(args[2:])
		}
		if len(args) > 1 && args[1] == "migrate" {
			return a.runWorkspaceMigrate(args[2:])
		}
		fmt.Fprintln(a.ErrOut, "usage: alp workspace <check|migrate> [options]")
		return 2
	case "domain":
		return a.runDomain(args[1:])
	default:
		fmt.Fprintf(a.ErrOut, "unknown command %q\n", args[0])
		a.usage()
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
	fmt.Fprintln(a.ErrOut, "commands: validate, workspace check, workspace migrate, domain list, domain info")
}
