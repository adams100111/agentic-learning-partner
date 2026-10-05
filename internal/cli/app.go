package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	contextbundle "github.com/adams100111/agentic-learning-partner/internal/context"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
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
		if len(args) > 1 && args[1] == "check" {
			return a.runWorkspaceCheck(args[2:])
		}
		fmt.Fprintln(a.ErrOut, "usage: alp workspace check [--workspace PATH]")
		return 2
	case "domain":
		return a.runDomain(args[1:])
	case "context":
		return a.runContext(args[1:])
	default:
		fmt.Fprintf(a.ErrOut, "unknown command %q\n", args[0])
		a.usage()
		return 2
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
	fmt.Fprintln(a.ErrOut, "commands: validate, workspace check, domain list, domain info, context build, context inspect")
}
