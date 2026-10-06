package cli

import (
	"flag"
	"fmt"

	"github.com/adams100111/agentic-learning-partner/internal/lifecycle"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func (a App) runWorkspaceLifecycle(command string, args []string) int {
	configPath, err := lifecycle.DefaultConfigPath()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}

	switch command {
	case "init":
		flags := flag.NewFlagSet("workspace init", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		provider := flags.String("provider", "local", "store provider: local or git")
		path := flags.String("path", "", "local workspace path")
		learner := flags.String("learner", "", "learner id")
		branch := flags.String("branch", "main", "Git branch")
		remote := flags.String("remote", "", "optional Git remote URL")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace init <name> --path PATH --learner ID [--provider local|git]")
			return 2
		}
		status, err := lifecycle.New(*config, validator).Init(flags.Arg(0), *provider, *path, *learner, *branch, *remote)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		a.printWorkspaceStatus(status)
		return 0

	case "clone":
		flags := flag.NewFlagSet("workspace clone", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		path := flags.String("path", "", "local workspace path")
		branch := flags.String("branch", "main", "Git branch")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 2 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace clone <name> <remote> --path PATH")
			return 2
		}
		status, err := lifecycle.New(*config, validator).Clone(flags.Arg(0), flags.Arg(1), *path, *branch)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		a.printWorkspaceStatus(status)
		return 0

	case "list":
		flags := flag.NewFlagSet("workspace list", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		items, err := lifecycle.New(*config, validator).List()
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		for _, item := range items {
			marker := " "
			if item.Default {
				marker = "*"
			}
			fmt.Fprintf(a.Out, "%s %s\t%s\t%s\n", marker, item.Name, item.Provider, item.Path)
		}
		return 0

	case "use":
		flags := flag.NewFlagSet("workspace use", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace use <name>")
			return 2
		}
		if err := lifecycle.New(*config, validator).Use(flags.Arg(0)); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "default workspace: %s\n", flags.Arg(0))
		return 0

	case "acknowledge-privacy":
		flags := flag.NewFlagSet("workspace acknowledge-privacy", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace acknowledge-privacy <name>")
			return 2
		}
		if err := lifecycle.New(*config, validator).AcknowledgeRemotePrivacy(flags.Arg(0)); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "remote privacy acknowledged for workspace: %s\n", flags.Arg(0))
		return 0

	case "status":
		flags := flag.NewFlagSet("workspace status", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		name := ""
		if flags.NArg() > 0 {
			name = flags.Arg(0)
		}
		status, err := lifecycle.New(*config, validator).Status(name)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		a.printWorkspaceStatus(status)
		return 0
	default:
		fmt.Fprintf(a.ErrOut, "unknown workspace command %q\n", command)
		return 2
	}
}

func (a App) printWorkspaceStatus(status lifecycle.WorkspaceStatus) {
	fmt.Fprintf(a.Out, "name: %s\n", status.Name)
	fmt.Fprintf(a.Out, "provider: %s\n", status.Provider)
	fmt.Fprintf(a.Out, "path: %s\n", status.Path)
	fmt.Fprintf(a.Out, "workspace: %s\n", status.WorkspaceID)
	fmt.Fprintf(a.Out, "learner: %s\n", status.LearnerID)
	fmt.Fprintf(a.Out, "revision: %s\n", status.Revision)
	if status.SyncMode != "" {
		fmt.Fprintf(a.Out, "sync: %s\n", status.SyncMode)
	}
	if status.Remote != "" {
		fmt.Fprintf(a.Out, "remote: %s\n", status.Remote)
	}
	if status.Branch != "" {
		fmt.Fprintf(a.Out, "branch: %s\n", status.Branch)
	}
	fmt.Fprint(a.Out, "capabilities:")
	for _, capability := range status.Capabilities {
		fmt.Fprintf(a.Out, " %s", capability)
	}
	fmt.Fprintln(a.Out)
}
