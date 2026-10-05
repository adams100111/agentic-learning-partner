package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

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
		fmt.Fprintln(a.ErrOut, "usage: alp workspace check [--workspace PATH]")
		return 2
	default:
		fmt.Fprintf(a.ErrOut, "unknown command %q\n", args[0])
		a.usage()
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

	resolution, info, ok := a.resolveAndInspect(*explicit)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	issues := validator.ValidateWorkspace(info.Path)
	if len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintln(a.ErrOut, issue.Error())
		}
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
	resolution, info, ok := a.resolveAndInspect(*explicit)
	if !ok {
		return 1
	}

	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintf(a.ErrOut, "initialize validation: %v\n", err)
		return 1
	}
	issues := validator.ValidateWorkspace(info.Path)
	if len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintln(a.ErrOut, issue.Error())
		}
		return 1
	}

	fmt.Fprintf(a.Out, "workspace: %s\nsource: %s\nrevision: %s\nschema: compatible\n", info.Path, resolution.Source, info.Revision)
	return 0
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
	fmt.Fprintln(a.ErrOut, "commands: validate, workspace check")
}
