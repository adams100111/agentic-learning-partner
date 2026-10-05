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
	Path   string
	Source string
}

type Resolver struct {
	Getenv  func(string) string
	HomeDir func() (string, error)
}

func NewResolver() Resolver {
	return Resolver{
		Getenv:  os.Getenv,
		HomeDir: os.UserHomeDir,
	}
}

func (r Resolver) Resolve(explicit, startDir string) (Resolution, error) {
	if explicit != "" {
		return resolvePath(explicit, "explicit --workspace")
	}

	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return Resolution{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	if path, ok, err := findProjectConfig(startDir); err != nil {
		return Resolution{}, err
	} else if ok {
		return resolvePath(path, "project .alp.yaml")
	}

	getenv := r.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if value := getenv(EnvWorkspace); value != "" {
		return resolvePath(value, EnvWorkspace)
	}

	homeDir := r.HomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve home directory: %w", err)
	}
	if path, ok, err := readWorkspaceConfig(filepath.Join(home, ".config", "alp", "config.yaml")); err != nil {
		return Resolution{}, err
	} else if ok {
		return resolvePath(path, "user config")
	}

	return Resolution{}, errors.New("no ALP learner workspace configured; pass --workspace, add .alp.yaml, set ALP_WORKSPACE, or configure ~/.config/alp/config.yaml")
}

func findProjectConfig(start string) (string, bool, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false, fmt.Errorf("resolve project directory: %w", err)
	}

	for {
		path, ok, err := readWorkspaceConfig(filepath.Join(current, ".alp.yaml"))
		if err != nil || ok {
			return path, ok, err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}

func readWorkspaceConfig(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
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

	if !filepath.IsAbs(config.Workspace) {
		config.Workspace = filepath.Join(filepath.Dir(path), config.Workspace)
	}
	return config.Workspace, true, nil
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
