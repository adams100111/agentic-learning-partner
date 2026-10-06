package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/migrate"
	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"github.com/adams100111/agentic-learning-partner/internal/workspacearchive"
)

type WorkspaceStatus struct {
	Name         string
	Path         string
	Provider     string
	WorkspaceID  string
	LearnerID    string
	Revision     store.Revision
	Capabilities []store.Capability
	SyncMode     string
	Remote       string
	Branch       string
	Default      bool
	PrivacyAck   bool
}

type Manager struct {
	ConfigPath string
	RuntimeDir string
	Validator  *workspace.Validator
	Rebuild    func(root string) error
	Now        func() time.Time
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil { return "", err }
	return filepath.Join(home, ".config", "alp", "config.yaml"), nil
}

func DefaultRuntimeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil { return "", err }
	return filepath.Join(home, ".local", "share", "alp", "runtime"), nil
}

func New(configPath, runtimeDir string, validator *workspace.Validator) Manager {
	return Manager{ConfigPath: configPath, RuntimeDir: runtimeDir, Validator: validator}
}

func (m Manager) Init(ctx context.Context, name, provider, path, learnerID, branch, remote string, privacyAcknowledged bool) (WorkspaceStatus, error) {
	if err := validateName(name); err != nil { return WorkspaceStatus{}, err }
	if strings.TrimSpace(learnerID) == "" { return WorkspaceStatus{}, errors.New("learner id is required") }
	if strings.TrimSpace(path) == "" { return WorkspaceStatus{}, errors.New("workspace path is required") }
	if provider == "" { provider = "local" }
	if provider != "local" && provider != "git" {
		return WorkspaceStatus{}, fmt.Errorf("unsupported store provider %q", provider)
	}
	if provider == "git" && remote != "" && !privacyAcknowledged {
		return WorkspaceStatus{}, errors.New("Git remote privacy is not verified; acknowledge the existing/unverifiable remote explicitly")
	}
	absolute, err := filepath.Abs(path)
	if err != nil { return WorkspaceStatus{}, err }
	if err := ensureEmptyDirectory(absolute); err != nil { return WorkspaceStatus{}, err }

	id, err := workspace.NewWorkspaceID()
	if err != nil { return WorkspaceStatus{}, err }
	if err := workspace.WriteManifest(absolute, workspace.Manifest{
		SchemaVersion: workspace.CurrentSchemaVersion,
		WorkspaceID: id,
		LearnerID: learnerID,
	}); err != nil { return WorkspaceStatus{}, err }

	entry := workspace.WorkspaceConfig{Path: absolute, Provider: provider}
	switch provider {
	case "local":
		if _, err := store.OpenLocal(absolute, m.Validator); err != nil { return WorkspaceStatus{}, err }
		entry.SyncMode = "manual"
	case "git":
		if branch == "" { branch = "main" }
		if _, err := store.InitializeGit(ctx, absolute, branch, remote, m.Validator); err != nil { return WorkspaceStatus{}, err }
		entry.SyncMode = "session"
		entry.Remote = "origin"
		entry.Branch = branch
		entry.PrivacyAck = privacyAcknowledged || remote == ""
	default:
		return WorkspaceStatus{}, fmt.Errorf("unsupported store provider %q", provider)
	}
	if err := m.addWorkspace(name, entry); err != nil { return WorkspaceStatus{}, err }
	return m.Status(ctx, name)
}

func (m Manager) Connect(ctx context.Context, name, provider, path, branch, remote string, privacyAcknowledged bool) (WorkspaceStatus, error) {
	if err := validateName(name); err != nil { return WorkspaceStatus{}, err }
	if path == "" { return WorkspaceStatus{}, errors.New("workspace path is required") }
	if provider == "" { provider = "local" }
	absolute, err := filepath.Abs(path)
	if err != nil { return WorkspaceStatus{}, err }

	entry := workspace.WorkspaceConfig{Path: absolute, Provider: provider}
	switch provider {
	case "local":
		active, err := store.OpenLocal(absolute, m.Validator)
		if err != nil { return WorkspaceStatus{}, err }
		manifest, err := workspace.ReadManifest(active.Root())
		if err != nil { return WorkspaceStatus{}, err }
		if manifest.SchemaVersion != workspace.CurrentSchemaVersion || manifest.WorkspaceID == "" {
			return WorkspaceStatus{}, fmt.Errorf("existing Local Store uses workspace schema %d; migrate it to schema %d before connecting", manifest.SchemaVersion, workspace.CurrentSchemaVersion)
		}
		entry.SyncMode = "manual"
	case "git":
		if branch == "" { branch = "main" }
		if remote != "" && !privacyAcknowledged {
			return WorkspaceStatus{}, errors.New("Git remote privacy is not verified; explicit acknowledgement is required")
		}
		active, err := store.OpenGit(absolute, m.Validator)
		if err != nil { return WorkspaceStatus{}, err }
		if err := m.migrateClonedGitWorkspace(ctx, active); err != nil { return WorkspaceStatus{}, err }
		entry.SyncMode = "session"
		entry.Remote = "origin"
		entry.Branch = branch
		entry.PrivacyAck = privacyAcknowledged || remote == ""
	default:
		return WorkspaceStatus{}, fmt.Errorf("unsupported store provider %q", provider)
	}
	if err := m.addWorkspace(name, entry); err != nil { return WorkspaceStatus{}, err }
	return m.Status(ctx, name)
}

func (m Manager) Clone(ctx context.Context, name, remote, path, branch string, privacyAcknowledged bool) (WorkspaceStatus, error) {
	if err := validateName(name); err != nil { return WorkspaceStatus{}, err }
	if strings.TrimSpace(remote) == "" { return WorkspaceStatus{}, errors.New("git remote is required") }
	if !privacyAcknowledged {
		return WorkspaceStatus{}, errors.New("existing Git remote privacy is not verified; explicit acknowledgement is required")
	}
	if path == "" { return WorkspaceStatus{}, errors.New("workspace path is required") }
	if branch == "" { branch = "main" }
	absolute, err := filepath.Abs(path)
	if err != nil { return WorkspaceStatus{}, err }
	gitStore, err := store.CloneGit(ctx, remote, absolute, branch, m.Validator)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if err := m.migrateClonedGitWorkspace(ctx, gitStore); err != nil {
		return WorkspaceStatus{}, err
	}
	entry := workspace.WorkspaceConfig{
		Path: absolute, Provider: "git", SyncMode: "session",
		Remote: "origin", Branch: branch, PrivacyAck: true,
	}
	if err := m.addWorkspace(name, entry); err != nil { return WorkspaceStatus{}, err }
	return m.Status(ctx, name)
}

func (m Manager) List(ctx context.Context) ([]WorkspaceStatus, error) {
	config, err := workspace.LoadConfigOrEmpty(m.ConfigPath)
	if err != nil { return nil, err }
	names := config.Names()
	result := make([]WorkspaceStatus, 0, len(names))
	for _, name := range names {
		status, err := m.Status(ctx, name)
		if err != nil { return nil, fmt.Errorf("workspace %q: %w", name, err) }
		result = append(result, status)
	}
	return result, nil
}

func (m Manager) Use(name string) error {
	config, err := workspace.LoadConfigOrEmpty(m.ConfigPath)
	if err != nil { return err }
	if _, ok := config.Workspaces[name]; !ok { return fmt.Errorf("workspace %q is not configured", name) }
	config.DefaultWorkspace = name
	config.Workspace = ""
	return workspace.WriteConfig(m.ConfigPath, config)
}

func (m Manager) AcknowledgeRemotePrivacy(name string) error {
	config, err := workspace.LoadConfigOrEmpty(m.ConfigPath)
	if err != nil { return err }
	entry, ok := config.Workspaces[name]
	if !ok { return fmt.Errorf("workspace %q is not configured", name) }
	if entry.Provider != "git" { return fmt.Errorf("workspace %q does not use Git Store", name) }
	entry.PrivacyAck = true
	config.Workspaces[name] = entry
	return workspace.WriteConfig(m.ConfigPath, config)
}

func (m Manager) Status(ctx context.Context, name string) (WorkspaceStatus, error) {
	active, entry, resolvedName, config, err := m.open(name)
	if err != nil { return WorkspaceStatus{}, err }
	manifest, err := workspace.ReadManifest(active.Root())
	if err != nil { return WorkspaceStatus{}, err }
	revision, err := active.Revision(ctx)
	if err != nil { return WorkspaceStatus{}, err }
	var capabilities []store.Capability
	for capability, enabled := range active.Capabilities() {
		if enabled { capabilities = append(capabilities, capability) }
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i] < capabilities[j] })
	status := WorkspaceStatus{
		Name: resolvedName, Path: active.Root(), Provider: active.Provider(),
		WorkspaceID: manifest.WorkspaceID, LearnerID: manifest.LearnerID,
		Revision: revision, Capabilities: capabilities, SyncMode: entry.SyncMode,
		Remote: entry.Remote, Branch: entry.Branch, PrivacyAck: entry.PrivacyAck,
		Default: config.DefaultWorkspace == resolvedName || (config.DefaultWorkspace == "" && resolvedName == "default"),
	}
	if gitStore, ok := active.(*store.Git); ok {
		gitStatus, statusErr := gitStore.Status(ctx)
		if statusErr == nil {
			if status.Branch == "" { status.Branch = gitStatus.Branch }
			if status.Remote == "" && gitStatus.Remote != "" { status.Remote = "origin" }
		}
	}
	return status, nil
}

func (m Manager) Sync(ctx context.Context, name, direction string) (store.SyncResult, error) {
	active, entry, _, _, err := m.open(name)
	if err != nil { return store.SyncResult{}, err }
	if err := store.Require(active, store.CapabilitySync); err != nil { return store.SyncResult{}, err }
	syncer, ok := active.(store.Syncer)
	if !ok { return store.SyncResult{}, fmt.Errorf("store provider %q advertises sync without Syncer implementation", active.Provider()) }
	options := store.SyncOptions{Remote: entry.Remote, Branch: entry.Branch, Rebuild: m.Rebuild}
	switch direction {
	case "", "sync":
		return syncer.Sync(ctx, options)
	case "pull":
		return syncer.Pull(ctx, options)
	case "push":
		return syncer.Push(ctx, options)
	default:
		return store.SyncResult{}, fmt.Errorf("unsupported sync direction %q", direction)
	}
}

func (m Manager) Export(ctx context.Context, name, destination string) (workspacearchive.Manifest, error) {
	active, _, _, _, err := m.open(name)
	if err != nil { return workspacearchive.Manifest{}, err }
	now := time.Now().UTC()
	if m.Now != nil { now = m.Now().UTC() }
	return workspacearchive.Export(ctx, active, destination, now)
}

func (m Manager) VerifyArchive(path string) (workspacearchive.Verified, error) {
	return workspacearchive.Verify(path, m.Validator)
}

func (m Manager) Restore(ctx context.Context, name, archivePath string, mode workspacearchive.RestoreMode) (store.Revision, error) {
	active, _, _, _, err := m.open(name)
	if err != nil { return "", err }
	verified, err := workspacearchive.Verify(archivePath, m.Validator)
	if err != nil { return "", err }
	return workspacearchive.Restore(ctx, verified, active, m.Validator, workspacearchive.RestoreOptions{
		Mode: mode, RuntimeDir: m.RuntimeDir, Rebuild: m.Rebuild,
	})
}

func (m Manager) Move(ctx context.Context, name, provider, destination, remote, branch string, privacyAcknowledged bool) (WorkspaceStatus, error) {
	active, current, resolvedName, config, err := m.open(name)
	if err != nil { return WorkspaceStatus{}, err }
	if provider == "" { return WorkspaceStatus{}, errors.New("destination provider is required") }
	if provider == active.Provider() { return WorkspaceStatus{}, fmt.Errorf("workspace %q already uses provider %q", resolvedName, provider) }
	if destination == "" { return WorkspaceStatus{}, errors.New("destination path is required") }
	absolute, err := filepath.Abs(destination)
	if err != nil { return WorkspaceStatus{}, err }

	var next workspace.WorkspaceConfig
	switch provider {
	case "git":
		if remote != "" && !privacyAcknowledged {
			return WorkspaceStatus{}, errors.New("Git remote privacy is not verified; explicit acknowledgement is required")
		}
		target, err := workspacearchive.ConvertToGit(ctx, active, m.Validator, workspacearchive.ConvertToGitOptions{
			Destination: absolute, Branch: branch, Remote: remote, RuntimeDir: m.RuntimeDir, Rebuild: m.Rebuild,
		})
		if err != nil { return WorkspaceStatus{}, err }
		status, _ := target.Status(ctx)
		next = workspace.WorkspaceConfig{Path: absolute, Provider: "git", SyncMode: "session", Remote: "origin", Branch: status.Branch, PrivacyAck: privacyAcknowledged || remote == ""}
	case "local":
		if err := ensureEmptyDirectory(absolute); err != nil { return WorkspaceStatus{}, err }
		manifest, err := workspace.ReadManifest(active.Root())
		if err != nil { return WorkspaceStatus{}, err }
		if err := workspace.WriteManifest(absolute, manifest); err != nil { return WorkspaceStatus{}, err }
		target, err := store.OpenLocal(absolute, m.Validator)
		if err != nil { return WorkspaceStatus{}, err }
		tempDir, err := os.MkdirTemp("", "alp-move-*")
		if err != nil { return WorkspaceStatus{}, err }
		defer os.RemoveAll(tempDir)
		archivePath := filepath.Join(tempDir, "workspace.alp")
		if _, err := workspacearchive.Export(ctx, active, archivePath, time.Unix(0, 0).UTC()); err != nil { return WorkspaceStatus{}, err }
		verified, err := workspacearchive.Verify(archivePath, m.Validator)
		if err != nil { return WorkspaceStatus{}, err }
		if _, err := workspacearchive.Restore(ctx, verified, target, m.Validator, workspacearchive.RestoreOptions{
			Mode: workspacearchive.RestoreRecover, RuntimeDir: m.RuntimeDir, Rebuild: m.Rebuild,
		}); err != nil { return WorkspaceStatus{}, err }
		next = workspace.WorkspaceConfig{Path: absolute, Provider: "local", SyncMode: "manual"}
	default:
		return WorkspaceStatus{}, fmt.Errorf("unsupported destination provider %q", provider)
	}

	// Switch configuration only after the destination has been fully created,
	// restored, rebuilt, and validated.
	config.Workspaces[resolvedName] = next
	if config.DefaultWorkspace == "" && current.Name == "default" {
		config.DefaultWorkspace = resolvedName
	}
	if err := workspace.WriteConfig(m.ConfigPath, config); err != nil { return WorkspaceStatus{}, err }
	return m.Status(ctx, resolvedName)
}

func (m Manager) open(name string) (store.Store, workspace.WorkspaceConfig, string, workspace.Config, error) {
	config, err := workspace.LoadConfigOrEmpty(m.ConfigPath)
	if err != nil { return nil, workspace.WorkspaceConfig{}, "", workspace.Config{}, err }
	resolvedName := name
	if resolvedName == "" {
		if config.DefaultWorkspace != "" {
			resolvedName = config.DefaultWorkspace
		} else if config.Workspace != "" {
			resolvedName = "default"
		}
	}
	entry, err := config.Resolve(name)
	if err != nil { return nil, workspace.WorkspaceConfig{}, "", config, err }
	if entry.Provider == "" { entry.Provider = "git" }
	var active store.Store
	switch entry.Provider {
	case "local":
		active, err = store.OpenLocal(entry.Path, m.Validator)
	case "git":
		active, err = store.OpenGit(entry.Path, m.Validator)
	default:
		err = fmt.Errorf("unsupported store provider %q", entry.Provider)
	}
	return active, entry, resolvedName, config, err
}

func (m Manager) migrateClonedGitWorkspace(ctx context.Context, active *store.Git) error {
	manifest, err := workspace.ReadManifest(active.Root())
	if err != nil {
		return err
	}
	if manifest.SchemaVersion >= workspace.CurrentSchemaVersion {
		return nil
	}
	before, err := active.Revision(ctx)
	if err != nil {
		return err
	}
	migrator := migrate.NewWorkspaceMigrator(m.Validator)
	migrator.Rebuild = m.Rebuild
	if _, err := migrator.Apply(active.Root(), strings.TrimPrefix(string(before), "git:")); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(active.Root(), "workspace.yaml"))
	if err != nil {
		return err
	}
	_, err = active.Commit(ctx, before, store.ChangeSet{
		Message: "alp: migrate workspace schema",
		Mutations: []store.Mutation{{Path: "workspace.yaml", Data: data}},
	})
	return err
}

func (m Manager) addWorkspace(name string, entry workspace.WorkspaceConfig) error {
	config, err := workspace.LoadConfigOrEmpty(m.ConfigPath)
	if err != nil { return err }
	if config.Workspaces == nil { config.Workspaces = map[string]workspace.WorkspaceConfig{} }
	if _, exists := config.Workspaces[name]; exists { return fmt.Errorf("workspace %q is already configured", name) }
	config.Workspaces[name] = entry
	config.Workspace = ""
	if config.DefaultWorkspace == "" { config.DefaultWorkspace = name }
	return workspace.WriteConfig(m.ConfigPath, config)
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" { return errors.New("workspace name is required") }
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	return nil
}

func ensureEmptyDirectory(path string) error {
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() { return fmt.Errorf("workspace path is not a directory: %s", path) }
		entries, err := os.ReadDir(path)
		if err != nil { return err }
		if len(entries) != 0 { return fmt.Errorf("workspace directory is not empty: %s", path) }
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}
