package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type DiscoveredCRD struct {
	ID          string   `json:"id"`
	Group       string   `json:"group"`
	Version     string   `json:"version"`
	Kind        string   `json:"kind"`
	Plural      string   `json:"plural"`
	Scope       string   `json:"scope"`
	APIVersion  string   `json:"apiVersion"`
	Description string   `json:"description,omitempty"`
	Versions    []string `json:"versions,omitempty"`
}

func discoverCRDs(kubeCfg string) ([]DiscoveredCRD, error) {
	cfg := getPlatformConfig()
	if !cfg.CrdDiscovery.Enabled {
		return []DiscoveredCRD{}, nil
	}
	clusterID := cfg.CrdDiscovery.Clusters
	if clusterID == "" {
		clusterID = "local"
	}
	return discoverCRDsForCluster(kubeCfg, clusterID)
}

func handlePortalDiscoverCRDs(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	clusterID := strings.TrimSpace(c.Query("cluster"))
	if clusterID == "" {
		clusterID = getPlatformConfig().CrdDiscovery.Clusters
	}
	if clusterID == "" {
		clusterID = "local"
	}
	crds, err := discoverCRDsForCluster(ru.Kubeconfig, clusterID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	presets := getPlatformConfig().CustomResourcePresets
	c.JSON(http.StatusOK, gin.H{"crds": crds, "presets": presets, "clusterId": clusterID})
}

func handlePortalGetPlatformConfig(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	admin := evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin
	yamlText, err := platformConfigYAML()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg := getPlatformConfig()
	resp := gin.H{
		"git":          cfg.Git,
		"templates":    cfg.Templates,
		"charts":       cfg.Charts,
		"collections":  collectionsFromConfig(),
		"offerings":    offeringsFromConfig(),
		"presets":      cfg.CustomResourcePresets,
		"defaults":     cfg.Defaults,
		"approval":     cfg.Approval,
		"crdDiscovery": cfg.CrdDiscovery,
	}
	if admin {
		resp["yaml"] = yamlText
		resp["configMap"] = configMapName()
		resp["namespace"] = portalNamespace()
	}
	c.JSON(http.StatusOK, resp)
}

func handlePortalGetBundledPlatformConfig(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
		return
	}
	cfg, err := parsePlatformConfigYAML(string(embeddedPlatformConfig))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	applyConfigDefaults(&cfg)
	c.JSON(http.StatusOK, gin.H{
		"yaml":         string(embeddedPlatformConfig),
		"collections":  cfg.Collections,
		"offerings":    cfg.Offerings,
		"git":          cfg.Git,
		"defaults":     cfg.Defaults,
		"approval":     cfg.Approval,
		"crdDiscovery": cfg.CrdDiscovery,
		"charts":       cfg.Charts,
		"templates":    cfg.Templates,
		"config": gin.H{
			"defaults":     cfg.Defaults,
			"git":          cfg.Git,
			"collections":  cfg.Collections,
			"offerings":    cfg.Offerings,
			"templates":    cfg.Templates,
			"charts":       cfg.Charts,
			"crdDiscovery": cfg.CrdDiscovery,
			"approval":     cfg.Approval,
		},
	})
}

type saveConfigBody struct {
	YAML   string          `json:"yaml"`
	Config *PlatformConfig `json:"config,omitempty"`
}

func handlePortalSerializePlatformConfig(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
		return
	}
	var cfg PlatformConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applyConfigDefaults(&cfg)
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"yaml": string(b), "config": cfg})
}

func handlePortalSavePlatformConfig(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
		return
	}
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var body saveConfigBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	yamlRaw := strings.TrimSpace(body.YAML)
	if body.Config != nil {
		b, err := yaml.Marshal(body.Config)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("marshal config: %v", err)})
			return
		}
		yamlRaw = string(b)
	}
	if yamlRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "yaml or config body required"})
		return
	}
	cfg, err := parsePlatformConfigYAML(yamlRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid yaml: %v", err)})
		return
	}
	if err := validatePlatformConfig(cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ns := portalNamespace()
	cmName := configMapName()
	patch := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
data:
  platform.yaml: |
%s`, cmName, ns, indentYAML(yamlRaw, "    "))

	args := []string{"apply", "-f", "-"}
	if ru.Kubeconfig != "" {
		args = append([]string{"--kubeconfig", ru.Kubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = strings.NewReader(patch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("apply ConfigMap: %s (%s)", err, strings.TrimSpace(string(out)))})
		return
	}

	setPlatformConfig(cfg)
	c.JSON(http.StatusOK, gin.H{"message": "Platform config saved", "configMap": cmName, "namespace": ns})
}

func indentYAML(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
