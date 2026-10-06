package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/device"
	"github.com/adams100111/agentic-learning-partner/internal/session"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/lifecycle"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

func (a App) runSession(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.ErrOut, "usage: alp session <begin|status|put|delete|flush|close|abort|recover-sync> [options]")
		return 2
	}
	command := args[0]
	flags := flag.NewFlagSet("session "+command, flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	explicit := flags.String("workspace", "", "learner workspace path or configured name")
	mode := flags.String("mode", "", "sync mode override: session, manual, or eager")
	harness := flags.String("harness", "", "harness identifier")
	deviceID := flags.String("device-id", "", "override machine-local device id")
	noRecord := flags.Bool("no-retain-record", false, "do not retain compact canonical session record")
	summary := flags.String("summary", "", "compact session summary")
	file := flags.String("file", "", "source file for session put")
	path := flags.String("path", "", "ALP-owned workspace path")
	remote := flags.String("remote", "", "Git remote name")
	branchName := flags.String("branch", "", "Git branch")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	_, info, ok := a.checkWorkspace(*explicit)
	if !ok {
		return 1
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	active, err := openStateStore(info.Path, validator)
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	configuredMode, configuredRemote, configuredBranch := sessionConfiguredDefaults(*explicit, info.Path)
	if *mode == "" {
		*mode = configuredMode
	}
	if *remote == "" {
		*remote = configuredRemote
	}
	if *branchName == "" {
		*branchName = configuredBranch
	}
	runtimeDir, err := lifecycle.DefaultRuntimeDir()
	if err != nil {
		fmt.Fprintln(a.ErrOut, err)
		return 1
	}
	manager := &session.Manager{
		Store: active,
		Validator: validator,
		RuntimeDir: runtimeDir,
		Rebuild: func(root string) error {
			_, err := (state.Store{Root: root, Validator: validator}).RebuildProjection()
			return err
		},
	}

	switch command {
	case "begin":
		id := *deviceID
		if id == "" {
			devicePath, err := device.DefaultPath()
			if err != nil {
				fmt.Fprintln(a.ErrOut, err)
				return 1
			}
			info, err := device.LoadOrCreate(devicePath)
			if err != nil {
				fmt.Fprintln(a.ErrOut, err)
				return 1
			}
			id = info.ID
		}
		retain := !*noRecord
		started, err := manager.Begin(context.Background(), session.BeginOptions{
			Harness: *harness,
			DeviceID: id,
			Mode: session.SyncMode(*mode),
			RetainRecord: &retain,
			Sync: storepkg.SyncOptions{Remote: *remote, Branch: *branchName},
		})
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "session: %s\nbase revision: %s\noffline: %t\n", started.ID(), started.BaseRevision(), started.Offline())
		return 0

	case "status":
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "session: %s\nbase revision: %s\noffline: %t\nstage: %s\n", resumed.ID(), resumed.BaseRevision(), resumed.Offline(), resumed.StageRoot())
		return 0

	case "put":
		if *path == "" || *file == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp session put --path ALP_PATH --file FILE [--workspace ...]")
			return 2
		}
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		if err := resumed.Put(*path, data); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "staged: %s\n", *path)
		return 0

	case "delete":
		if *path == "" {
			fmt.Fprintln(a.ErrOut, "usage: alp session delete --path ALP_PATH [--workspace ...]")
			return 2
		}
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		if err := resumed.Delete(*path); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "staged deletion: %s\n", *path)
		return 0

	case "flush":
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		result, err := resumed.Flush(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "revision: %s\nsync pending: %t\n", result.Revision, result.SyncPending)
		if result.SyncResult.Message != "" {
			fmt.Fprintf(a.Out, "sync message: %s\n", result.SyncResult.Message)
		}
		return 0

	case "close":
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		input, err := discoverSessionCloseInput(active.Root(), resumed.StageRoot(), *summary)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		result, err := resumed.Close(context.Background(), input)
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "revision: %s\nsync pending: %t\n", result.Revision, result.SyncPending)
		if result.SyncResult.Message != "" {
			fmt.Fprintf(a.Out, "sync message: %s\n", result.SyncResult.Message)
		}
		return 0

	case "abort":
		resumed, err := manager.Resume(context.Background())
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		if err := resumed.Abort(); err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintln(a.Out, "session aborted")
		return 0

	case "recover-sync":
		result, err := manager.RecoverPendingSync(context.Background(), storepkg.SyncOptions{Remote: *remote, Branch: *branchName})
		if err != nil {
			fmt.Fprintln(a.ErrOut, err)
			return 1
		}
		fmt.Fprintf(a.Out, "revision: %s\npending: %t\nattempts: %d\n", result.Revision, result.Pending, result.Attempts)
		return 0

	default:
		fmt.Fprintf(a.ErrOut, "unknown session command %q\n", command)
		return 2
	}
}

func sessionConfiguredDefaults(reference, root string) (mode, remote, branch string) {
	mode = "session"
	configPath, err := lifecycle.DefaultConfigPath()
	if err != nil {
		return mode, "", ""
	}
	config, err := workspace.LoadConfig(configPath)
	if err != nil {
		return mode, "", ""
	}
	choose := func(entry workspace.WorkspaceConfig) bool {
		entryPath, err := filepath.Abs(entry.Path)
		if err != nil || filepath.Clean(entryPath) != filepath.Clean(root) {
			return false
		}
		if entry.SyncMode != "" {
			mode = entry.SyncMode
		}
		remote = entry.Remote
		branch = entry.Branch
		return true
	}
	if reference != "" {
		if entry, ok := config.Workspaces[reference]; ok && choose(entry) {
			return mode, remote, branch
		}
	}
	if config.DefaultWorkspace != "" {
		if entry, ok := config.Workspaces[config.DefaultWorkspace]; ok && choose(entry) {
			return mode, remote, branch
		}
	}
	for _, entry := range config.Workspaces {
		if choose(entry) {
			return mode, remote, branch
		}
	}
	return mode, "", ""
}

func discoverSessionCloseInput(root, stage, summary string) (session.CloseInput, error) {
	input := session.CloseInput{Summary: summary}
	domains := map[string]struct{}{}
	for _, group := range []struct {
		dir string
		target *[]string
	}{
		{dir: "evidence", target: &input.Evidence},
		{dir: "assessments", target: &input.Assessments},
	} {
		pattern := filepath.Join(stage, group.dir, "*.yaml")
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return session.CloseInput{}, err
		}
		for _, stagedPath := range paths {
			name := filepath.Base(stagedPath)
			if _, err := os.Stat(filepath.Join(root, group.dir, name)); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return session.CloseInput{}, err
			}
			id := strings.TrimSuffix(name, filepath.Ext(name))
			*group.target = append(*group.target, id)
			data, err := os.ReadFile(stagedPath)
			if err != nil {
				return session.CloseInput{}, err
			}
			var record struct{ Domain string `yaml:"domain"` }
			if err := yaml.Unmarshal(data, &record); err != nil {
				return session.CloseInput{}, err
			}
			if record.Domain != "" {
				domains[record.Domain] = struct{}{}
			}
		}
		sort.Strings(*group.target)
	}
	for domainName := range domains {
		input.Domains = append(input.Domains, domainName)
	}
	sort.Strings(input.Domains)
	return input, nil
}
