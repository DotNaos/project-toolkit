package projectconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ProjectToolkitDirName     = ".project-toolkit"
	ConfigRelativePath        = ".project-toolkit/config.yaml"
	BaseWorkspaceRelativePath = ".project-toolkit/base.code-workspace"
	DefaultLogsRelativeDir    = "logs/project-toolkit"
)

type Config struct {
	Dev       *DevConfig         `yaml:"dev"`
	Logs      *LogsConfig        `yaml:"logs"`
	Project   *ProjectConfig     `yaml:"project"`
	Workspace *WorkspaceConfig   `yaml:"workspace"`
	Shared    []SharedLinkConfig `yaml:"shared"`
}

type DevConfig struct {
	Command string           `yaml:"command"`
	Args    []string         `yaml:"args"`
	Router  *DevRouterConfig `yaml:"router"`
}

type DevRouterConfig struct {
	Mode string `yaml:"mode"`
	Name string `yaml:"name"`
}

type LogsConfig struct {
	Dir string `yaml:"dir"`
}

type ProjectConfig struct {
	Name string `yaml:"name"`
}

type WorkspaceConfig struct {
	BaseFile string `yaml:"baseFile"`
}

type SharedLinkConfig struct {
	Path    string   `yaml:"path"`
	Source  string   `yaml:"source"`
	Target  string   `yaml:"target"`
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

func Load(cwd string) (Config, error) {
	configPath := filepath.Join(cwd, ConfigRelativePath)

	source, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}

		return Config{}, fmt.Errorf("failed to read %s: %w", ConfigRelativePath, err)
	}

	var config Config
	if err := yaml.Unmarshal(source, &config); err != nil {
		return Config{}, fmt.Errorf("invalid %s: %w", ConfigRelativePath, err)
	}

	if err := normalize(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func normalize(config *Config) error {
	if err := normalizeDev(config); err != nil {
		return err
	}
	normalizeLogs(config)
	normalizeProject(config)
	normalizeWorkspace(config)
	return normalizeShared(config)
}

func normalizeDev(config *Config) error {
	if config.Dev == nil {
		return nil
	}

	config.Dev.Command = strings.TrimSpace(config.Dev.Command)
	args, err := normalizeStringList(config.Dev.Args, ConfigRelativePath+".dev.args")
	if err != nil {
		return err
	}
	config.Dev.Args = args

	if config.Dev.Router != nil {
		config.Dev.Router.Mode = strings.TrimSpace(config.Dev.Router.Mode)
		config.Dev.Router.Name = strings.TrimSpace(config.Dev.Router.Name)
		if config.Dev.Router.Mode == "" && config.Dev.Router.Name == "" {
			config.Dev.Router = nil
		}
	}

	if config.Dev.Router != nil {
		switch config.Dev.Router.Mode {
		case "portless", "dockportless":
		default:
			return fmt.Errorf("%s.dev.router.mode must be 'portless' or 'dockportless'", ConfigRelativePath)
		}
	}

	if config.Dev.Command == "" && len(config.Dev.Args) == 0 {
		if config.Dev.Router != nil {
			return fmt.Errorf("%s.dev.router requires %s.dev.command or %s.dev.args", ConfigRelativePath, ConfigRelativePath, ConfigRelativePath)
		}

		config.Dev = nil
	}

	return nil
}

func normalizeLogs(config *Config) {
	if config.Logs == nil {
		return
	}

	config.Logs.Dir = strings.TrimSpace(config.Logs.Dir)
	if config.Logs.Dir == "" {
		config.Logs = nil
	}
}

func normalizeProject(config *Config) {
	if config.Project == nil {
		return
	}

	config.Project.Name = strings.TrimSpace(config.Project.Name)
	if config.Project.Name == "" {
		config.Project = nil
	}
}

func normalizeWorkspace(config *Config) {
	if config.Workspace == nil {
		return
	}

	config.Workspace.BaseFile = strings.TrimSpace(config.Workspace.BaseFile)
	if config.Workspace.BaseFile == "" {
		config.Workspace = nil
	}
}

func normalizeShared(config *Config) error {
	for index := range config.Shared {
		if err := normalizeSharedEntry(&config.Shared[index], index); err != nil {
			return err
		}
	}

	return nil
}

func normalizeSharedEntry(entry *SharedLinkConfig, index int) error {
	entry.Path = strings.TrimSpace(entry.Path)
	entry.Source = strings.TrimSpace(entry.Source)
	entry.Target = strings.TrimSpace(entry.Target)

	if entry.Path == "" {
		return fmt.Errorf("%s.shared[%d].path must be a non-empty string", ConfigRelativePath, index)
	}

	include, err := normalizeStringList(entry.Include, fmt.Sprintf("%s.shared[%d].include", ConfigRelativePath, index))
	if err != nil {
		return err
	}

	exclude, err := normalizeStringList(entry.Exclude, fmt.Sprintf("%s.shared[%d].exclude", ConfigRelativePath, index))
	if err != nil {
		return err
	}

	entry.Include = include
	entry.Exclude = exclude
	return nil
}

func normalizeStringList(values []string, label string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(values))
	for index, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			return nil, fmt.Errorf("%s[%d] must be a non-empty string", label, index)
		}

		result = append(result, normalized)
	}

	return result, nil
}
