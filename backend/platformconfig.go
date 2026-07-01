package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed config/platform.yaml
var embeddedPlatformConfig []byte

type PlatformConfig struct {
	Defaults              PlatformDefaults       `yaml:"defaults" json:"defaults"`
	Git                   GitConfig              `yaml:"git" json:"git"`
	Collections           []CollectionEntry      `yaml:"collections" json:"collections"`
	Offerings             []OfferingEntry        `yaml:"offerings" json:"offerings"`
	Templates             []ConfigTemplate       `yaml:"templates" json:"templates"`
	Charts                []ConfigChart          `yaml:"charts" json:"charts"`
	CustomResourcePresets []CustomResourcePreset `yaml:"customResourcePresets" json:"customResourcePresets"`
	CrdDiscovery          CrdDiscoveryConfig     `yaml:"crdDiscovery" json:"crdDiscovery"`
	Approval              ApprovalConfig         `yaml:"approval" json:"approval"`
}

type PlatformDefaults struct {
	Namespace      string `yaml:"namespace" json:"namespace"`
	FleetNamespace string `yaml:"fleetNamespace" json:"fleetNamespace"`
	GitSecretName  string `yaml:"gitSecretName" json:"gitSecretName"`
	GitBranch      string `yaml:"gitBranch" json:"gitBranch"`
	GitPathPrefix  string `yaml:"gitPathPrefix" json:"gitPathPrefix"`
}

type GitConfig struct {
	Mode        string    `yaml:"mode" json:"mode"`
	DefaultRepo string    `yaml:"defaultRepo" json:"defaultRepo"`
	Repos       []GitRepo `yaml:"repos" json:"repos"`
}

type GitRepo struct {
	ID         string `yaml:"id" json:"id"`
	Name       string `yaml:"name" json:"name"`
	URL        string `yaml:"url" json:"url"`
	Branch     string `yaml:"branch" json:"branch"`
	SecretName string `yaml:"secretName" json:"secretName"`
	AuthType   string `yaml:"authType,omitempty" json:"authType,omitempty"`
}

type ConfigTemplate struct {
	ID               string `yaml:"id" json:"id"`
	Label            string `yaml:"label" json:"label"`
	Description      string `yaml:"description" json:"description"`
	Detail           string `yaml:"detail,omitempty" json:"detail,omitempty"`
	Icon             string `yaml:"icon,omitempty" json:"icon,omitempty"`
	GitOps           bool   `yaml:"gitOps" json:"gitOps"`
	RequiresApproval bool   `yaml:"requiresApproval" json:"requiresApproval"`
}

type ConfigChart struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Category    string         `yaml:"category" json:"category"`
	Description string         `yaml:"description" json:"description"`
	Type        string         `yaml:"type" json:"type"`
	Helm        *ConfigHelmRef `yaml:"helm,omitempty" json:"helm,omitempty"`
}

type ConfigHelmRef struct {
	Repo    string `yaml:"repo" json:"repo"`
	Chart   string `yaml:"chart" json:"chart"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type CustomResourcePreset struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Category    string `yaml:"category" json:"category"`
	Description string `yaml:"description" json:"description"`
	APIVersion  string `yaml:"apiVersion" json:"apiVersion"`
	Kind        string `yaml:"kind" json:"kind"`
	Scope       string `yaml:"scope" json:"scope"`
	DefaultSpec string `yaml:"defaultSpec,omitempty" json:"defaultSpec,omitempty"`
}

type CrdDiscoveryConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	Clusters      string   `yaml:"clusters" json:"clusters"`
	ExcludeGroups []string `yaml:"excludeGroups" json:"excludeGroups"`
}

type ApprovalConfig struct {
	ChartsRequireApproval           bool `yaml:"chartsRequireApproval" json:"chartsRequireApproval"`
	CustomResourcesRequireApproval  bool `yaml:"customResourcesRequireApproval" json:"customResourcesRequireApproval"`
}

var (
	platformConfigMu sync.RWMutex
	platformConfig   PlatformConfig
)

func platformConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("PLATFORM_CONFIG_PATH")); p != "" {
		return p
	}
	return "/etc/platform/platform.yaml"
}

func loadPlatformConfig() error {
	platformConfigMu.Lock()
	defer platformConfigMu.Unlock()

	data := embeddedPlatformConfig
	if path := platformConfigPath(); path != "" {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			data = b
		}
	}

	var cfg PlatformConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	applyConfigDefaults(&cfg)
	platformConfig = cfg
	return nil
}

func applyConfigDefaults(cfg *PlatformConfig) {
	if cfg.Defaults.Namespace == "" {
		cfg.Defaults.Namespace = "devportal-system"
	}
	if cfg.Defaults.FleetNamespace == "" {
		cfg.Defaults.FleetNamespace = "fleet-default"
	}
	if cfg.Defaults.GitSecretName == "" {
		cfg.Defaults.GitSecretName = "platform-git-credentials"
	}
	if cfg.Defaults.GitBranch == "" {
		cfg.Defaults.GitBranch = "main"
	}
	if cfg.Defaults.GitPathPrefix == "" {
		cfg.Defaults.GitPathPrefix = "environments"
	}
	if cfg.Git.Mode == "" {
		cfg.Git.Mode = "single"
	}
	if len(cfg.Templates) == 0 {
		cfg.Templates = fallbackTemplates()
	}
	if len(cfg.Charts) == 0 {
		cfg.Charts = fallbackCharts()
	}
}

func getPlatformConfig() PlatformConfig {
	platformConfigMu.RLock()
	defer platformConfigMu.RUnlock()
	return platformConfig
}

func setPlatformConfig(cfg PlatformConfig) {
	applyConfigDefaults(&cfg)
	platformConfigMu.Lock()
	platformConfig = cfg
	platformConfigMu.Unlock()
}

func platformConfigYAML() (string, error) {
	cfg := getPlatformConfig()
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parsePlatformConfigYAML(raw string) (PlatformConfig, error) {
	var cfg PlatformConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return PlatformConfig{}, err
	}
	applyConfigDefaults(&cfg)
	return cfg, nil
}

func catalogFromConfig() []ChartEntry {
	cfg := getPlatformConfig()
	out := make([]ChartEntry, 0, len(cfg.Charts))
	for _, c := range cfg.Charts {
		out = append(out, ChartEntry{
			ID:          c.ID,
			Name:        c.Name,
			Category:    c.Category,
			Description: c.Description,
		})
	}
	return out
}

func templatesFromConfig() []TemplateEntry {
	cfg := getPlatformConfig()
	out := make([]TemplateEntry, 0, len(cfg.Templates))
	for _, t := range cfg.Templates {
		out = append(out, TemplateEntry{
			ID:               t.ID,
			Label:            t.Label,
			Description:      t.Description,
			Detail:           t.Detail,
			Icon:             t.Icon,
			GitOps:           t.GitOps,
			RequiresApproval: t.RequiresApproval,
		})
	}
	return out
}

func templateFromConfig(id string) (ConfigTemplate, bool) {
	cfg := getPlatformConfig()
	for _, t := range cfg.Templates {
		if t.ID == id {
			return t, true
		}
	}
	return ConfigTemplate{}, false
}

func chartHelmFromConfig(id string) (ConfigHelmRef, bool) {
	cfg := getPlatformConfig()
	for _, c := range cfg.Charts {
		if c.ID == id && c.Helm != nil {
			return *c.Helm, true
		}
	}
	return ConfigHelmRef{}, false
}

func defaultGitRepoFromConfig() string {
	cfg := getPlatformConfig()
	if cfg.Git.DefaultRepo != "" {
		return cfg.Git.DefaultRepo
	}
	if len(cfg.Git.Repos) > 0 {
		return cfg.Git.Repos[0].URL
	}
	return platformGitRepo()
}

func gitRepoByID(id string) (GitRepo, bool) {
	cfg := getPlatformConfig()
	for _, r := range cfg.Git.Repos {
		if r.ID == id || r.URL == id {
			return r, true
		}
	}
	return GitRepo{}, false
}

func requestNeedsApprovalFromConfig(template string, charts []string, customResources []CustomResourceEntry) bool {
	cfg := getPlatformConfig()
	if t, ok := templateFromConfig(template); ok {
		if t.RequiresApproval {
			return true
		}
		if !t.GitOps && len(charts) == 0 && len(customResources) == 0 {
			return false
		}
	}
	if cfg.Approval.ChartsRequireApproval && len(charts) > 0 {
		return true
	}
	if cfg.Approval.CustomResourcesRequireApproval && len(customResources) > 0 {
		return true
	}
	if t, ok := templateFromConfig(template); ok && t.GitOps {
		return true
	}
	return needsApproval(template, charts)
}

func requestNeedsGitOpsFromConfig(template string, charts []string, customResources []CustomResourceEntry) bool {
	if len(charts) > 0 || len(customResources) > 0 {
		return true
	}
	if t, ok := templateFromConfig(template); ok {
		return t.GitOps
	}
	return template != "sandbox"
}

func isGroupExcluded(group string, excluded []string) bool {
	for _, g := range excluded {
		if g == group {
			return true
		}
	}
	return false
}

func fallbackTemplates() []ConfigTemplate {
	return []ConfigTemplate{
		{ID: "sandbox", Label: "Sandbox", Description: "Dev namespace", GitOps: false, RequiresApproval: false, Icon: "namespace"},
		{ID: "team", Label: "Team", Description: "Team + GitOps", GitOps: true, RequiresApproval: true, Icon: "fleet"},
		{ID: "vcluster", Label: "Virtual cluster", Description: "vCluster", GitOps: true, RequiresApproval: true, Icon: "cluster"},
	}
}

func fallbackCharts() []ConfigChart {
	return []ConfigChart{
		{ID: "rancher-monitoring", Name: "Monitoring", Category: "observability", Helm: &ConfigHelmRef{Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-monitoring"}},
	}
}

func configMapName() string {
	if n := os.Getenv("PLATFORM_CONFIGMAP"); n != "" {
		return n
	}
	return "platform-config"
}

func validatePlatformConfig(cfg PlatformConfig) error {
	migrateLegacyCatalog(&cfg)
	if len(cfg.Collections) == 0 {
		return fmt.Errorf("at least one collection is required")
	}
	if len(cfg.Offerings) == 0 {
		return fmt.Errorf("at least one offering is required")
	}
	seenCol := map[string]bool{}
	for _, c := range cfg.Collections {
		if c.ID == "" {
			return fmt.Errorf("collection id is required")
		}
		if seenCol[c.ID] {
			return fmt.Errorf("duplicate collection id %q", c.ID)
		}
		seenCol[c.ID] = true
	}
	seenOff := map[string]bool{}
	for _, o := range cfg.Offerings {
		if o.ID == "" {
			return fmt.Errorf("offering id is required")
		}
		if seenOff[o.ID] {
			return fmt.Errorf("duplicate offering id %q", o.ID)
		}
		seenOff[o.ID] = true
		if o.CollectionID == "" {
			return fmt.Errorf("offering %q requires collectionId", o.ID)
		}
		if !seenCol[o.CollectionID] {
			return fmt.Errorf("offering %q references unknown collection %q", o.ID, o.CollectionID)
		}
		if o.Kind == "" {
			return fmt.Errorf("offering %q requires kind", o.ID)
		}
	}
	// Legacy template validation for backward compatibility
	if len(cfg.Templates) > 0 {
		seen := map[string]bool{}
		for _, t := range cfg.Templates {
			if t.ID == "" {
				return fmt.Errorf("template id is required")
			}
			if seen[t.ID] {
				return fmt.Errorf("duplicate template id %q", t.ID)
			}
			seen[t.ID] = true
		}
	}
	return nil
}
