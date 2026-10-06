package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/adams100111/agentic-learning-partner/internal/lifecycle"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"github.com/adams100111/agentic-learning-partner/internal/workspacearchive"
)

func (a App) runWorkspaceLifecycle(command string, args []string) int {
	configPath, err := lifecycle.DefaultConfigPath()
	if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
	runtimeDir, err := lifecycle.DefaultRuntimeDir()
	if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
	validator, err := workspace.NewValidator()
	if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
	manager := lifecycle.New(configPath, runtimeDir, validator)
	manager.Rebuild = func(root string) error {
		_, err := (state.Store{Root: root, Validator: validator}).RebuildProjection()
		return err
	}
	ctx := context.Background()

	switch command {
	case "init":
		flags := flag.NewFlagSet("workspace init", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		runtime := flags.String("runtime", runtimeDir, "ALP machine runtime directory")
		provider := flags.String("provider", "local", "store provider: local or git")
		path := flags.String("path", "", "local workspace path")
		learner := flags.String("learner", "", "learner id")
		branchName := flags.String("branch", "main", "Git branch")
		remote := flags.String("remote", "", "optional Git remote URL")
		privacy := flags.Bool("acknowledge-unverified-privacy", false, "acknowledge that ALP cannot verify remote privacy")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 1 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace init <name> --path PATH --learner ID [--provider local|git]")
			return 2
		}
		manager.ConfigPath, manager.RuntimeDir = *config, *runtime
		status, err := manager.Init(ctx, flags.Arg(0), *provider, *path, *learner, *branchName, *remote, *privacy)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		a.printWorkspaceStatus(status)
		return 0

	case "clone":
		flags := flag.NewFlagSet("workspace clone", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		runtime := flags.String("runtime", runtimeDir, "ALP machine runtime directory")
		path := flags.String("path", "", "local workspace path")
		branchName := flags.String("branch", "main", "Git branch")
		privacy := flags.Bool("acknowledge-unverified-privacy", false, "acknowledge that ALP cannot verify remote privacy")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 2 {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace clone <name> <remote> --path PATH --acknowledge-unverified-privacy")
			return 2
		}
		manager.ConfigPath, manager.RuntimeDir = *config, *runtime
		status, err := manager.Clone(ctx, flags.Arg(0), flags.Arg(1), *path, *branchName, *privacy)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		a.printWorkspaceStatus(status)
		return 0

	case "list":
		flags := flag.NewFlagSet("workspace list", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil { return 2 }
		manager.ConfigPath = *config
		items, err := manager.List(ctx)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		for _, item := range items {
			marker := " "
			if item.Default { marker = "*" }
			fmt.Fprintf(a.Out, "%s %s\t%s\t%s\n", marker, item.Name, item.Provider, item.Path)
		}
		return 0

	case "use":
		flags := flag.NewFlagSet("workspace use", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 1 { fmt.Fprintln(a.ErrOut, "usage: alp workspace use <name>"); return 2 }
		manager.ConfigPath = *config
		if err := manager.Use(flags.Arg(0)); err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "default workspace: %s\n", flags.Arg(0))
		return 0

	case "status":
		flags := flag.NewFlagSet("workspace status", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil { return 2 }
		name := ""
		if flags.NArg() > 0 { name = flags.Arg(0) }
		manager.ConfigPath = *config
		status, err := manager.Status(ctx, name)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		a.printWorkspaceStatus(status)
		return 0

	case "acknowledge-privacy":
		flags := flag.NewFlagSet("workspace acknowledge-privacy", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 1 { fmt.Fprintln(a.ErrOut, "usage: alp workspace acknowledge-privacy <name>"); return 2 }
		manager.ConfigPath = *config
		if err := manager.AcknowledgeRemotePrivacy(flags.Arg(0)); err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "remote privacy acknowledged for workspace: %s\n", flags.Arg(0))
		return 0

	case "sync":
		flags := flag.NewFlagSet("workspace sync", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		direction := flags.String("direction", "sync", "sync direction: sync, pull, or push")
		if err := flags.Parse(args); err != nil { return 2 }
		name := ""
		if flags.NArg() > 0 { name = flags.Arg(0) }
		manager.ConfigPath = *config
		result, err := manager.Sync(ctx, name, *direction)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "revision: %s\nchanged: %t\npending: %t\nattempts: %d\n", result.Revision, result.Changed, result.Pending, result.Attempts)
		if result.Message != "" { fmt.Fprintf(a.Out, "message: %s\n", result.Message) }
		return 0

	case "export":
		flags := flag.NewFlagSet("workspace export", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		output := flags.String("out", "", "destination .alp archive")
		if err := flags.Parse(args); err != nil { return 2 }
		if *output == "" { fmt.Fprintln(a.ErrOut, "--out is required"); return 2 }
		name := ""
		if flags.NArg() > 0 { name = flags.Arg(0) }
		manager.ConfigPath = *config
		manifest, err := manager.Export(ctx, name, *output)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "archive: %s\nworkspace: %s\nentries: %d\n", *output, manifest.WorkspaceID, len(manifest.Entries))
		return 0

	case "verify":
		flags := flag.NewFlagSet("workspace verify", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 1 { fmt.Fprintln(a.ErrOut, "usage: alp workspace verify <archive.alp>"); return 2 }
		verified, err := manager.VerifyArchive(flags.Arg(0))
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "valid archive: %s\nworkspace: %s\nlearner: %s\nentries: %d\n", flags.Arg(0), verified.Manifest.WorkspaceID, verified.Manifest.LearnerID, len(verified.Manifest.Entries))
		return 0

	case "restore":
		flags := flag.NewFlagSet("workspace restore", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		runtime := flags.String("runtime", runtimeDir, "ALP machine runtime directory")
		mode := flags.String("mode", "", "restore mode: recover, clone, or merge")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 2 || *mode == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace restore <name> <archive.alp> --mode recover|clone|merge")
			return 2
		}
		manager.ConfigPath, manager.RuntimeDir = *config, *runtime
		revision, err := manager.Restore(ctx, flags.Arg(0), flags.Arg(1), workspacearchive.RestoreMode(*mode))
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		fmt.Fprintf(a.Out, "restored revision: %s\n", revision)
		return 0

	case "move":
		flags := flag.NewFlagSet("workspace move", flag.ContinueOnError)
		flags.SetOutput(a.ErrOut)
		config := flags.String("config", configPath, "ALP machine config path")
		runtime := flags.String("runtime", runtimeDir, "ALP machine runtime directory")
		provider := flags.String("provider", "", "destination provider: local or git")
		path := flags.String("path", "", "destination workspace path")
		remote := flags.String("remote", "", "optional Git remote URL")
		branchName := flags.String("branch", "main", "Git branch")
		privacy := flags.Bool("acknowledge-unverified-privacy", false, "acknowledge that ALP cannot verify remote privacy")
		if err := flags.Parse(args); err != nil { return 2 }
		if flags.NArg() != 1 || *provider == "" || *path == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp workspace move <name> --provider local|git --path PATH")
			return 2
		}
		manager.ConfigPath, manager.RuntimeDir = *config, *runtime
		status, err := manager.Move(ctx, flags.Arg(0), *provider, *path, *remote, *branchName, *privacy)
		if err != nil { fmt.Fprintln(a.ErrOut, err); return 1 }
		a.printWorkspaceStatus(status)
		return 0

	default:
		fmt.Fprintf(a.ErrOut, "unknown workspace command %q\n", command)
		return 2
	}
}

func (a App) printWorkspaceStatus(status lifecycle.WorkspaceStatus) {
	fmt.Fprintf(a.Out, "name: %s\nprovider: %s\npath: %s\nworkspace: %s\nlearner: %s\nrevision: %s\n", status.Name, status.Provider, status.Path, status.WorkspaceID, status.LearnerID, status.Revision)
	if status.SyncMode != "" { fmt.Fprintf(a.Out, "sync: %s\n", status.SyncMode) }
	if status.Remote != "" { fmt.Fprintf(a.Out, "remote: %s\n", status.Remote) }
	if status.Branch != "" { fmt.Fprintf(a.Out, "branch: %s\n", status.Branch) }
	fmt.Fprint(a.Out, "capabilities:")
	for _, capability := range status.Capabilities { fmt.Fprintf(a.Out, " %s", capability) }
	fmt.Fprintln(a.Out)
}
