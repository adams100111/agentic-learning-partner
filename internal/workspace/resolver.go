package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const EnvWorkspace = "ALP_WORKSPACE"

type Resolution struct {
	Path     string
	Source   string
	Name     string
	Provider ProviderConfig
}

type Resolver struct {
	Getenv  func(string) string
	HomeDir func() (string, error)
}

func NewResolver() Resolver {
	return Resolver{Getenv: os.Getenv, HomeDir: os.UserHomeDir}
}

func (r Resolver) Resolve(explicit, startDir string) (Resolution, error) {
	home, configPath, config, err := r.userConfig()
	if err != nil {
		return Resolution{}, err
	}

	if explicit != "" {
		return resolveReference(explicit, config, filepath.Dir(configPath), "explicit --workspace")
	}

	if startDir == "" {
		startDir, err = os.Getwd()
		if err != nil {
			return Resolution{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}
	if reference, ok, err := findProjectConfig(startDir); err != nil {
		return Resolution{}, err
	} else if ok {
		return resolveReference(reference, config, filepath.Dir(configPath), "project .alp.yaml")
	}

	getenv := r.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if value := getenv(EnvWorkspace); value != "" {
		return resolveReference(value, config, filepath.Dir(configPath), EnvWorkspace)
	}

	name, provider, ok := config.Default()
	if ok {
		return resolveProvider(name, provider, filepath.Dir(configPath), "user config")
	}

	return Resolution{}, fmt.Errorf("no ALP learner workspace configured under %s; pass --workspace, add .alp.yaml, set ALP_WORKSPACE, or configure a default workspace", home)
}

func (r Resolver) userConfig() (string, string, UserConfig, error) {
	homeDir := r.HomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return "", "", UserConfig{}, fmt.Errorf("resolve home directory: %w", err)
	}
	path := filepath.Join(home, ".config", "alp", "config.yaml")
	config, err := ReadUserConfig(path)
	if err != nil {
		return "", "", UserConfig{}, err
	}
	return home, path, config, nil
}

func findProjectConfig(start string) (string, bool, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false, fmt.Errorf("resolve project directory: %w", err)
	}
	for {
		path := filepath.Join(current, ".alp.yaml")
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			parent := filepath.Dir(current)
			if parent == current {
				return "", false, nil
			}
			current = parent
			continue
		}
		if err != nil {
			return "", false, fmt.Errorf("read %s: %w", path, err)
		}
		var config struct {
			Workspace string `yaml:"workspace"`
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", false, fmt.Errorf("parse %s: %w", path, err)
		}
		if config.Workspace == "" {
			return "", false, fmt.Errorf("%s: workspace is required", path)
		}
		if !filepath.IsAbs(config.Workspace) && looksLikePath(config.Workspace) {
			config.Workspace = filepath.Join(filepath.Dir(path), config.Workspace)
		}
		return config.Workspace, true, nil
	}
}

func resolveReference(reference string, config UserConfig, configDir, source string) (Resolution, error) {
	if provider, ok := config.Named(reference); ok {
		return resolveProvider(reference, provider, configDir, source)
	}
	return resolvePath(reference, source)
}

func resolveProvider(name string, provider ProviderConfig, configDir, source string) (Resolution, error) {
	if provider.Type == "" {
		provider.Type = "local"
	}
	if provider.Path == "" {
		return Resolution{}, fmt.Errorf("workspace %q: path is required", name)
	}
	path := provider.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	resolution, err := resolvePath(path, source)
	if err != nil {
		return Resolution{}, err
	}
	resolution.Name = name
	resolution.Provider = provider
	resolution.Provider.Path = resolution.Path
	return resolution, nil
}

func resolvePath(path, source string) (Resolution, error) {
	expanded, err := expandHome(path)
	if err != nil {
		return Resolution{}, err
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve workspace path %q: %w", path, err)
	}
	return Resolution{Path: filepath.Clean(absolute), Source: source}, nil
}

func expandHome(path string) (string, error) {
	if path == "~" || (len(path) > 2 && path[:2] == "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand workspace home: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func looksLikePath(value string) bool {
	return value == "." || value == ".." || filepath.IsAbs(value) || value[0] == '.' ||
		filepath.Dir(value) != "."
}
