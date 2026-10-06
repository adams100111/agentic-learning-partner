package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

type WorkspaceConfig struct {
	Name       string `yaml:"-" json:"name"`
	Path       string `yaml:"path" json:"path"`
	Provider   string `yaml:"provider" json:"provider"`
	SyncMode   string `yaml:"syncMode,omitempty" json:"syncMode,omitempty"`
	Remote     string `yaml:"remote,omitempty" json:"remote,omitempty"`
	Branch     string `yaml:"branch,omitempty" json:"branch,omitempty"`
	PrivacyAck bool   `yaml:"privacyAcknowledged,omitempty" json:"privacyAcknowledged,omitempty"`
}

type Config struct {
	DefaultWorkspace string                     `yaml:"defaultWorkspace,omitempty" json:"defaultWorkspace,omitempty"`
	Workspace        string                     `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	Workspaces       map[string]WorkspaceConfig `yaml:"workspaces,omitempty" json:"workspaces,omitempty"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if config.Workspaces == nil {
		config.Workspaces = map[string]WorkspaceConfig{}
	}
	return config, nil
}

func (c Config) Resolve(name string) (WorkspaceConfig, error) {
	if name == "" {
		name = c.DefaultWorkspace
	}
	if name == "" && c.Workspace != "" {
		return WorkspaceConfig{Name: "default", Path: c.Workspace}, nil
	}
	if name == "" {
		return WorkspaceConfig{}, errors.New("no default workspace configured")
	}
	entry, ok := c.Workspaces[name]
	if !ok {
		return WorkspaceConfig{}, fmt.Errorf("unknown workspace %q", name)
	}
	entry.Name = name
	if entry.Path == "" {
		return WorkspaceConfig{}, fmt.Errorf("workspace %q path is required", name)
	}
	if entry.Provider == "" {
		entry.Provider = "git"
	}
	return entry, nil
}

func LoadConfigOrEmpty(path string) (Config, error) {
	config, err := LoadConfig(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Workspaces: map[string]WorkspaceConfig{}}, nil
	}
	return config, err
}

func WriteConfig(path string, config Config) error {
	if config.Workspaces == nil {
		config.Workspaces = map[string]WorkspaceConfig{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".alp-config-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (c Config) Names() []string {
	names := make([]string, 0, len(c.Workspaces))
	for name := range c.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
