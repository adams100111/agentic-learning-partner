package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

type ProviderConfig struct {
	Type       string `yaml:"provider"`
	Path       string `yaml:"path"`
	SyncMode   string `yaml:"syncMode,omitempty"`
	Remote     string `yaml:"remote,omitempty"`
	Branch     string `yaml:"branch,omitempty"`
	PrivacyAck bool   `yaml:"privacyAcknowledged,omitempty"`
}

type UserConfig struct {
	DefaultWorkspace string                    `yaml:"defaultWorkspace,omitempty"`
	Workspaces       map[string]ProviderConfig `yaml:"workspaces,omitempty"`

	// Workspace keeps backward compatibility with the original single-path config.
	Workspace string `yaml:"workspace,omitempty"`
}

func ReadUserConfig(path string) (UserConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return UserConfig{}, nil
	}
	if err != nil {
		return UserConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return UserConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return config, nil
}

func WriteUserConfig(path string, config UserConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal user config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return fmt.Errorf("write user config: %w", err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("replace user config: %w", err)
	}
	return nil
}

func (c UserConfig) Names() []string {
	names := make([]string, 0, len(c.Workspaces))
	for name := range c.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c UserConfig) Named(name string) (ProviderConfig, bool) {
	value, ok := c.Workspaces[name]
	return value, ok
}

func (c UserConfig) Default() (string, ProviderConfig, bool) {
	if c.DefaultWorkspace != "" {
		value, ok := c.Workspaces[c.DefaultWorkspace]
		return c.DefaultWorkspace, value, ok
	}
	if c.Workspace != "" {
		return "default", ProviderConfig{Type: "git", Path: c.Workspace, SyncMode: "session", Remote: "origin", Branch: "main"}, true
	}
	return "", ProviderConfig{}, false
}
