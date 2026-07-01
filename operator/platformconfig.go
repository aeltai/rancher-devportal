package main

import (
	_ "embed"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed platform.yaml
var embeddedPlatformYAML []byte

type opPlatformConfig struct {
	Defaults PlatformDefaults `yaml:"defaults"`
	Git      PlatformGit      `yaml:"git"`
	Charts   []ConfigChart    `yaml:"charts"`
}

type PlatformGit struct {
	DefaultRepo string        `yaml:"defaultRepo"`
	Repos       []PlatformGitRepo `yaml:"repos"`
}

type PlatformGitRepo struct {
	URL string `yaml:"url"`
}

type PlatformDefaults struct {
	FleetNamespace string `yaml:"fleetNamespace"`
	GitSecretName  string `yaml:"gitSecretName"`
	GitBranch      string `yaml:"gitBranch"`
	GitPathPrefix  string `yaml:"gitPathPrefix"`
}

type ConfigChart struct {
	ID   string         `yaml:"id"`
	Helm *ConfigHelmRef `yaml:"helm,omitempty"`
}

type ConfigHelmRef struct {
	Repo    string `yaml:"repo"`
	Chart   string `yaml:"chart"`
	Version string `yaml:"version,omitempty"`
}

func (r *reconciler) loadOpConfig() opPlatformConfig {
	data := embeddedPlatformYAML
	if path := strings.TrimSpace(os.Getenv("PLATFORM_CONFIG_PATH")); path != "" {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			data = b
		}
	}
	var cfg opPlatformConfig
	_ = yaml.Unmarshal(data, &cfg)
	if cfg.Defaults.FleetNamespace == "" {
		cfg.Defaults.FleetNamespace = r.cfg.FleetNamespace
	}
	if cfg.Defaults.GitBranch == "" {
		cfg.Defaults.GitBranch = r.cfg.DefaultGitBranch
	}
	if cfg.Defaults.GitSecretName == "" {
		cfg.Defaults.GitSecretName = r.cfg.DefaultGitSecret
	}
	return cfg
}

func (cfg opPlatformConfig) defaultGitRepo() string {
	if cfg.Git.DefaultRepo != "" {
		return cfg.Git.DefaultRepo
	}
	if len(cfg.Git.Repos) > 0 && cfg.Git.Repos[0].URL != "" {
		return cfg.Git.Repos[0].URL
	}
	return ""
}

func (cfg opPlatformConfig) chartHelm(id string) (ConfigHelmRef, bool) {
	for _, c := range cfg.Charts {
		if c.ID == id && c.Helm != nil {
			return *c.Helm, true
		}
	}
	return ConfigHelmRef{}, false
}
