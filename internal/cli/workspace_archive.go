package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/lifecycle"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	workspacearchive "github.com/adams100111/agentic-learning-partner/internal/workspacearchive"
)

func (a App) runWorkspaceArchive(command string, args []string) int {
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
	validate := func(root string) error {
		issues := validator.ValidateWorkspace(root)
		if len(issues) != 0 {
			return issues[0]
		}
		return nil
	}
	ctx := context.Background()

	switch command {
	case "export":
		flags := flag.NewFlagSet("workspace export", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		out := flags.String("out", "", "output .alp archive")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		name := ""
		if flags.NArg() > 0 {
			name = flags.Arg(0)
		}
		if *out == "" {
			fmt.Fprintln(a.ErrOut, "--out is required")
			return 2
		}
		source, _, err := lifecycle.New(*config, validator).Open(name)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		manifest, err := workspacearchive.Export(ctx, source, *out, time.Now().UTC())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "archive: %s\nworkspace: %s\nentries: %d\n", *out, manifest.WorkspaceID, len(manifest.Entries))
		return 0

	case "verify":
		flags := flag.NewFlagSet("workspace verify", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace verify <archive.alp>")
			return 2
		}
		verified, err := workspacearchive.Verify(flags.Arg(0), validate)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "valid archive\nworkspace: %s\nlearner: %s\nentries: %d\n",
			verified.Manifest.WorkspaceID, verified.Manifest.LearnerID, len(verified.Manifest.Entries))
		return 0

	case "restore":
		flags := flag.NewFlagSet("workspace restore", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		mode := flags.String("mode", "recover", "restore mode: recover, clone, merge")
		name := flags.String("name", "", "target workspace name")
		provider := flags.String("provider", "local", "provider for clone mode")
		path := flags.String("path", "", "workspace path for clone mode")
		branch := flags.String("branch", "main", "Git branch for clone mode")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 || *name == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace restore <archive.alp> --name NAME --mode recover|clone|merge")
			return 2
		}
		verified, err := workspacearchive.Verify(flags.Arg(0), validate)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		manager := lifecycle.New(*config, validator)
		if workspacearchive.RestoreMode(*mode) == workspacearchive.RestoreClone {
			if *path == "" {
				fmt.Fprintln(a.ErrOut, "--path is required for clone restore")
				return 2
			}
			if _, err := manager.Init(*name, *provider, *path, verified.Manifest.LearnerID, *branch, ""); err != nil {
				fmt.Fprintln(a.ErrOut, err)
				return 1
			}
		}
		target, _, err := manager.Open(*name)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		checkpoint, err := workspacearchive.Restore(ctx, verified, target, workspacearchive.RestoreMode(*mode))
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "restored workspace: %s\nrevision: %s\n", *name, checkpoint.Revision)
		return 0

	case "move":
		flags := flag.NewFlagSet("workspace move", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		provider := flags.String("provider", "", "destination provider")
		path := flags.String("path", "", "destination workspace path")
		remote := flags.String("remote", "", "optional Git remote URL")
		branch := flags.String("branch", "main", "Git branch")
		if err := flags.Parse(args); err != nil {
			return 2
		}
		if flags.NArg() != 1 || *provider != "git" || *path == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace move <name> --provider git --path PATH [--remote URL]")
			return 2
		}
		name := flags.Arg(0)
		manager := lifecycle.New(*config, validator)
		source, _, err := manager.Open(name)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		destination, err := filepath.Abs(*path)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		if _, err := workspacearchive.ConvertToGit(ctx, source, destination, *branch, *remote, validate); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		providerConfig := workspace.ProviderConfig{
			Type: "git", Path: destination, SyncMode: "session", Remote: "origin", Branch: *branch,
		}
		if err := manager.Replace(name, providerConfig); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "workspace %s moved to Git Store at %s\n", name, destination)
		return 0
	default:
		fmt.Fprintf(a.ErrOut, "unknown workspace archive command %q\n", command)
		return 2
	}
}
