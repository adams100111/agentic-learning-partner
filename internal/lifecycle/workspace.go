package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/store/gitstore"
	localstore "github.com/adams100111/agentic-learning-partner/internal/store/local"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type WorkspaceStatus struct {
	Name         string
	Path         string
	Provider     string
	WorkspaceID  string
	LearnerID    string
	Revision     storepkg.Revision
	Capabilities []storepkg.Capability
	SyncMode     string
	Remote       string
	Branch       string
	Default      bool
}

type Manager struct {
	ConfigPath string
	Validator  *workspace.Validator
}

func New(configPath string, validator *workspace.Validator) Manager {
	return Manager{ConfigPath: configPath, Validator: validator}
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "alp", "config.yaml"), nil
}

func (m Manager) Init(name, provider, path, learnerID, branch, remote string) (WorkspaceStatus, error) {
	if err := validateName(name); err != nil {
		return WorkspaceStatus{}, err
	}
	if provider == "" {
		provider = "local"
	}
	if path == "" {
		return WorkspaceStatus{}, errors.New("workspace path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	var configured workspace.ProviderConfig
	switch provider {
	case "local":
		if _, err := localstore.Initialize(absolute, learnerID, m.validate); err != nil {
			return WorkspaceStatus{}, err
		}
		configured = workspace.ProviderConfig{Type: "local", Path: absolute}
	case "git":
		if branch == "" {
			branch = "main"
		}
		store, err := gitstore.Initialize(absolute, learnerID, branch, m.validate)
		if err != nil {
			return WorkspaceStatus{}, err
		}
		configured = workspace.ProviderConfig{
			Type: "git", Path: absolute, SyncMode: "session", Remote: "origin", Branch: branch,
		}
		if remote != "" {
			if err := store.ConfigureRemote("origin", remote); err != nil {
				return WorkspaceStatus{}, err
			}
		}
	default:
		return WorkspaceStatus{}, fmt.Errorf("unsupported store provider %q", provider)
	}
	if err := m.saveWorkspace(name, configured, false); err != nil {
		return WorkspaceStatus{}, err
	}
	return m.Status(name)
}

func (m Manager) Clone(name, remote, path, branch string) (WorkspaceStatus, error) {
	if err := validateName(name); err != nil {
		return WorkspaceStatus{}, err
	}
	if strings.TrimSpace(remote) == "" {
		return WorkspaceStatus{}, errors.New("git remote is required")
	}
	if path == "" {
		return WorkspaceStatus{}, errors.New("workspace path is required")
	}
	if branch == "" {
		branch = "main"
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if _, err := gitstore.Clone(remote, absolute, branch, m.validate); err != nil {
		return WorkspaceStatus{}, err
	}
	config := workspace.ProviderConfig{
		Type: "git", Path: absolute, SyncMode: "session", Remote: "origin", Branch: branch,
	}
	if err := m.saveWorkspace(name, config, false); err != nil {
		return WorkspaceStatus{}, err
	}
	return m.Status(name)
}

func (m Manager) List() ([]WorkspaceStatus, error) {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return nil, err
	}
	var result []WorkspaceStatus
	for _, name := range config.Names() {
		status, err := m.Status(name)
		if err != nil {
			return nil, fmt.Errorf("workspace %q: %w", name, err)
		}
		result = append(result, status)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m Manager) AcknowledgeRemotePrivacy(name string) error {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return err
	}
	provider, ok := config.Named(name)
	if !ok {
		return fmt.Errorf("workspace %q is not configured", name)
	}
	if provider.Type != "git" {
		return fmt.Errorf("workspace %q does not use Git Store", name)
	}
	provider.PrivacyAck = true
	config.Workspaces[name] = provider
	return workspace.WriteUserConfig(m.ConfigPath, config)
}

func (m Manager) Use(name string) error {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return err
	}
	if _, ok := config.Named(name); !ok {
		return fmt.Errorf("workspace %q is not configured", name)
	}
	config.DefaultWorkspace = name
	config.Workspace = ""
	return workspace.WriteUserConfig(m.ConfigPath, config)
}

func (m Manager) Status(name string) (WorkspaceStatus, error) {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if name == "" {
		var ok bool
		name, _, ok = config.Default()
		if !ok {
			return WorkspaceStatus{}, errors.New("no default workspace configured")
		}
	}
	provider, ok := config.Named(name)
	if !ok {
		if legacyName, legacy, legacyOK := config.Default(); legacyOK && legacyName == name {
			provider = legacy
			ok = true
		}
	}
	if !ok {
		return WorkspaceStatus{}, fmt.Errorf("workspace %q is not configured", name)
	}
	active, err := m.open(provider)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	ctx := context.Background()
	identity, err := active.Workspace(ctx)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	revision, err := active.Revision(ctx)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	var capabilities []storepkg.Capability
	for capability, enabled := range active.Capabilities() {
		if enabled {
			capabilities = append(capabilities, capability)
		}
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i] < capabilities[j] })
	return WorkspaceStatus{
		Name: name, Path: identity.Path, Provider: active.Provider(),
		WorkspaceID: identity.ID, LearnerID: identity.LearnerID, Revision: revision,
		Capabilities: capabilities, SyncMode: provider.SyncMode, Remote: provider.Remote, Branch: provider.Branch,
		Default: config.DefaultWorkspace == name || (config.DefaultWorkspace == "" && config.Workspace != "" && name == "default"),
	}, nil
}

func (m Manager) Open(name string) (storepkg.Store, workspace.ProviderConfig, error) {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return nil, workspace.ProviderConfig{}, err
	}
	if name == "" {
		defaultName, provider, ok := config.Default()
		if !ok {
			return nil, workspace.ProviderConfig{}, errors.New("no default workspace configured")
		}
		name = defaultName
		active, err := m.open(provider)
		return active, provider, err
	}
	provider, ok := config.Named(name)
	if !ok {
		return nil, workspace.ProviderConfig{}, fmt.Errorf("workspace %q is not configured", name)
	}
	active, err := m.open(provider)
	return active, provider, err
}

func (m Manager) Register(name string, provider workspace.ProviderConfig) error {
	if err := validateName(name); err != nil {
		return err
	}
	return m.saveWorkspace(name, provider, false)
}

func (m Manager) Replace(name string, provider workspace.ProviderConfig) error {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return err
	}
	if _, ok := config.Named(name); !ok {
		return fmt.Errorf("workspace %q is not configured", name)
	}
	config.Workspaces[name] = provider
	config.Workspace = ""
	return workspace.WriteUserConfig(m.ConfigPath, config)
}

func (m Manager) saveWorkspace(name string, provider workspace.ProviderConfig, makeDefault bool) error {
	config, err := workspace.ReadUserConfig(m.ConfigPath)
	if err != nil {
		return err
	}
	if config.Workspaces == nil {
		config.Workspaces = map[string]workspace.ProviderConfig{}
	}
	if _, exists := config.Workspaces[name]; exists {
		return fmt.Errorf("workspace %q is already configured", name)
	}
	config.Workspaces[name] = provider
	config.Workspace = ""
	if makeDefault || config.DefaultWorkspace == "" {
		config.DefaultWorkspace = name
	}
	return workspace.WriteUserConfig(m.ConfigPath, config)
}

func (m Manager) open(provider workspace.ProviderConfig) (storepkg.Store, error) {
	switch provider.Type {
	case "", "local":
		return localstore.Open(provider.Path, m.validate)
	case "git":
		active, err := gitstore.Open(provider.Path, provider.Branch, m.validate)
		if err != nil {
			return nil, err
		}
		return active.WithRemote(provider.Remote), nil
	default:
		return nil, fmt.Errorf("unsupported store provider %q", provider.Type)
	}
}

func (m Manager) validate(root string) error {
	if m.Validator == nil {
		return nil
	}
	issues := m.Validator.ValidateWorkspace(root)
	if len(issues) != 0 {
		return issues[0]
	}
	return nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("workspace name is required")
	}
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	return nil
}
