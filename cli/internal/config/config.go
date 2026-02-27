package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type RepositoryConfig struct {
	Version   int    `yaml:"version"`
	Manifest  string `yaml:"manifest"`
	SourceURL string `yaml:"source_url"`
}

type Manifest struct {
	Name  string      `yaml:"name"`
	Files []FileEntry `yaml:"files"`
}

type FileEntry struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

func Write(target string, cfg RepositoryConfig) error {
	dir := filepath.Join(target, ".repo-kit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), b, 0o644)
}

func Read(target string) (RepositoryConfig, error) {
	path := filepath.Join(target, ".repo-kit", "config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return RepositoryConfig{}, err
	}
	var cfg RepositoryConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return RepositoryConfig{}, err
	}
	if cfg.Manifest == "" {
		return RepositoryConfig{}, fmt.Errorf("manifest is required")
	}
	return cfg, nil
}

func ReadManifest(kitRoot, manifest string) (Manifest, error) {
	name := manifest
	if filepath.Ext(name) == "" {
		name += ".yaml"
	}
	path := filepath.Join(kitRoot, "manifests", name)
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return Manifest{}, err
	}
	if m.Name == "" {
		m.Name = manifest
	}
	return m, nil
}
