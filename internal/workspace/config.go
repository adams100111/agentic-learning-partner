package workspace

import (
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

type WorkspaceConfig struct {
	Name     string `yaml:"-" json:"name"`
	Path     string `yaml:"path" json:"path"`
	Provider string `yaml:"provider" json:"provider"`
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
