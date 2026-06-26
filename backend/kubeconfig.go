package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type kubeConfig struct {
	APIVersion     string                 `yaml:"apiVersion"`
	Kind           string                 `yaml:"kind"`
	Clusters       []namedCluster         `yaml:"clusters"`
	Contexts       []namedContext         `yaml:"contexts"`
	CurrentContext string                 `yaml:"current-context"`
	Users          []namedUser            `yaml:"users"`
	Preferences    map[string]interface{} `yaml:"preferences,omitempty"`
}

type namedCluster struct {
	Name    string                 `yaml:"name"`
	Cluster map[string]interface{} `yaml:"cluster"`
}

type namedContext struct {
	Name    string                 `yaml:"name"`
	Context map[string]interface{} `yaml:"context"`
}

type namedUser struct {
	Name string                 `yaml:"name"`
	User map[string]interface{} `yaml:"user"`
}

type Cluster struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func userKubeconfigDir(userID string) string {
	safe := strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(userID)
	if safe == "" {
		safe = "anonymous"
	}
	return filepath.Join("/root/.kube/users", safe)
}

func kubeconfigPathForUser(userID string) string {
	return filepath.Join(userKubeconfigDir(userID), "config")
}

func rewriteKubeconfigServerURLs(cfg *kubeConfig) {
	ru := rancherURL()
	rancherU, err := url.Parse(ru)
	if err != nil {
		return
	}
	rancherHost := rancherU.Host
	if rancherU.Port() == "" && rancherU.Scheme == "https" {
		rancherHost = rancherU.Hostname() + ":443"
	} else if rancherU.Port() == "" && rancherU.Scheme == "http" {
		rancherHost = rancherU.Hostname() + ":80"
	}
	for i := range cfg.Clusters {
		cluster := cfg.Clusters[i].Cluster
		server, _ := cluster["server"].(string)
		if server == "" {
			continue
		}
		su, err := url.Parse(server)
		if err != nil {
			continue
		}
		host := su.Hostname()
		if host == "127.0.0.1" || host == "localhost" || host == "::1" || host == "0.0.0.0" {
			su.Scheme = rancherU.Scheme
			su.Host = rancherHost
			cluster["server"] = su.String()
		}
		cluster["insecure-skip-tls-verify"] = true
	}
}

func mergeKubeconfigs(configs []string) ([]byte, error) {
	var merged kubeConfig
	merged.APIVersion = "v1"
	merged.Kind = "Config"
	seenClusters := make(map[string]bool)
	seenContexts := make(map[string]bool)
	seenUsers := make(map[string]bool)

	for i, cfgYaml := range configs {
		if strings.TrimSpace(cfgYaml) == "" {
			continue
		}
		var cfg kubeConfig
		if err := yaml.Unmarshal([]byte(cfgYaml), &cfg); err != nil {
			return nil, fmt.Errorf("parse config %d: %w", i, err)
		}
		for _, c := range cfg.Clusters {
			if !seenClusters[c.Name] {
				merged.Clusters = append(merged.Clusters, c)
				seenClusters[c.Name] = true
			}
		}
		for _, c := range cfg.Contexts {
			if !seenContexts[c.Name] {
				merged.Contexts = append(merged.Contexts, c)
				seenContexts[c.Name] = true
			}
		}
		for _, u := range cfg.Users {
			if !seenUsers[u.Name] {
				merged.Users = append(merged.Users, u)
				seenUsers[u.Name] = true
			}
		}
		if merged.CurrentContext == "" && cfg.CurrentContext != "" {
			merged.CurrentContext = cfg.CurrentContext
		}
	}

	rewriteKubeconfigServerURLs(&merged)
	return yaml.Marshal(merged)
}

func fetchClustersWithToken(token string) ([]Cluster, error) {
	body, err := rancherRequestWithToken("GET", "/v3/clusters", token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	clusters := make([]Cluster, len(result.Data))
	for i, c := range result.Data {
		clusters[i] = Cluster{ID: c.ID, Name: c.Name, State: c.State}
	}
	return clusters, nil
}

func fetchKubeconfigWithToken(clusterID, token string) (string, error) {
	apiURL := rancherURL() + "/v3/clusters/" + clusterID + "?action=generateKubeconfig"
	req, err := http.NewRequest(http.MethodPost, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("kubeconfig request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("kubeconfig API returned %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Config string `json:"config"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Config, nil
}

func syncKubeconfigForUser(token string, kubeCfgPath string) (int, error) {
	clusters, err := fetchClustersWithToken(token)
	if err != nil {
		return 0, err
	}
	if len(clusters) == 0 {
		return 0, fmt.Errorf("no Rancher clusters available for this user")
	}
	var configs []string
	for _, cl := range clusters {
		cfg, err := fetchKubeconfigWithToken(cl.ID, token)
		if err != nil {
			return 0, fmt.Errorf("cluster %s: %w", cl.Name, err)
		}
		configs = append(configs, cfg)
	}
	merged, err := mergeKubeconfigs(configs)
	if err != nil {
		return 0, err
	}
	dir := filepath.Dir(kubeCfgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return 0, err
	}
	if err := os.WriteFile(kubeCfgPath, merged, 0600); err != nil {
		return 0, err
	}
	return len(clusters), nil
}

func ensureKubeconfig(ru *requestUser) error {
	if ru == nil {
		return fmt.Errorf("not authenticated")
	}
	if _, err := os.Stat(ru.Kubeconfig); err == nil {
		return nil
	}
	_, err := syncKubeconfigForUser(ru.Token, ru.Kubeconfig)
	return err
}

func runKubectlWithConfig(kubeCfg string, args ...string) (string, error) {
	cmdArgs := args
	if kubeCfg != "" {
		cmdArgs = append([]string{"--kubeconfig", kubeCfg}, args...)
	}
	cmd := exec.Command("kubectl", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
