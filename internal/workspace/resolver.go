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
	home, userConfig, err := r.userConfig()
	if err != nil {
		return Resolution{}, err
	}

	if explicit != "" {
		return resolveReference(explicit, "explicit --workspace", "", userConfig)
	}

	if startDir == "" {
		startDir, err = os.Getwd()
		if err != nil {
			return Resolution{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}
	if reference, configDir, ok, err := findProjectPointer(startDir); err != nil {
		return Resolution{}, err
	} else if ok {
		return resolveReference(reference, "project .alp.yaml", configDir, userConfig)
	}

	getenv := r.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if value := getenv(EnvWorkspace); value != "" {
		return resolveReference(value, EnvWorkspace, "", userConfig)
	}

	if userConfig != nil {
		entry, err := userConfig.Resolve("")
		if err == nil {
			return resolveReference(entry.Path, "user config", filepath.Dir(filepath.Join(home, ".config", "alp", "config.yaml")), userConfig)
		}
		if !errors.Is(err, os.ErrNotExist) && err.Error() != "no default workspace configured" {
			return Resolution{}, err
		}
	}

	return Resolution{}, errors.New("no ALP learner workspace configured; pass --workspace, add .alp.yaml, set ALP_WORKSPACE, or configure ~/.config/alp/config.yaml")
}

func (r Resolver) userConfig() (string, *Config, error) {
	homeDir := r.HomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return "", nil, fmt.Errorf("resolve home directory: %w", err)
	}
	path := filepath.Join(home, ".config", "alp", "config.yaml")
	config, err := LoadConfig(path)
	if errors.Is(err, os.ErrNotExist) {
		return home, nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	return home, &config, nil
}

func findProjectPointer(start string) (reference, configDir string, ok bool, err error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve project directory: %w", err)
	}
	for {
		path := filepath.Join(current, ".alp.yaml")
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			var config struct {
				Workspace string `yaml:"workspace"`
			}
			if err := yaml.Unmarshal(data, &config); err != nil {
				return "", "", false, fmt.Errorf("parse %s: %w", path, err)
			}
			if config.Workspace == "" {
				return "", "", false, fmt.Errorf("%s: workspace is required", path)
			}
			return config.Workspace, current, true, nil
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return "", "", false, fmt.Errorf("read %s: %w", path, readErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", "", false, nil
		}
		current = parent
	}
}

func resolveReference(reference, source, baseDir string, config *Config) (Resolution, error) {
	if config != nil {
		if entry, ok := config.Workspaces[reference]; ok {
			return resolvePath(entry.Path, source+" named workspace "+reference, "")
		}
	}
	return resolvePath(reference, source, baseDir)
}

func resolvePath(path, source, baseDir string) (Resolution, error) {
	expanded, err := expandHome(path)
	if err != nil {
		return Resolution{}, err
	}
	if !filepath.IsAbs(expanded) && baseDir != "" {
		expanded = filepath.Join(baseDir, expanded)
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
