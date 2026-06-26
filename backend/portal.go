package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed crd/platformrequest.yaml
var platformRequestCRD []byte

const crGroup = "platform.devportal.io"
const crVersion = "v1alpha1"
const crPlural = "platformrequests"

type ChartEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type TemplateEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Detail      string `json:"detail,omitempty"`
}

type PlatformRequestSpec struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Template    string   `json:"template"`
	Charts      []string `json:"charts"`
	Requester   string   `json:"requester"`
}

type PlatformRequestStatus struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

type PlatformRequest struct {
	CRName          string          `json:"crName"`
	Name            string          `json:"name"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description"`
	Namespace       string          `json:"namespace"`
	Template        string          `json:"template"`
	Charts          []string        `json:"charts"`
	Requester       string          `json:"requester"`
	Phase           string          `json:"phase"`
	Message         string          `json:"message"`
	CreatedAt       string          `json:"createdAt"`
	ManifestYAML    string          `json:"manifestYaml,omitempty"`
	FleetResources  []FleetResource `json:"fleetResources,omitempty"`
	GitRepoURL      string          `json:"gitRepoUrl,omitempty"`
	GitBranch       string          `json:"gitBranch,omitempty"`
	GitPath         string          `json:"gitPath,omitempty"`
	PullRequestHint string          `json:"pullRequestHint,omitempty"`
}

var defaultCatalog = []ChartEntry{
	{ID: "rancher-monitoring", Name: "Monitoring", Category: "observability", Description: "Prometheus + Grafana stack"},
	{ID: "rancher-logging", Name: "Logging", Category: "observability", Description: "Banzai Cloud logging operator"},
	{ID: "rancher-backup", Name: "Rancher Backup", Category: "backup", Description: "Backup/restore for Rancher"},
	{ID: "fleet", Name: "Fleet", Category: "gitops", Description: "GitOps continuous delivery"},
	{ID: "cert-manager", Name: "cert-manager", Category: "security", Description: "TLS certificate automation"},
	{ID: "ingress-nginx", Name: "Ingress NGINX", Category: "networking", Description: "Ingress controller"},
}

var defaultTemplates = []TemplateEntry{
	{
		ID:          "sandbox",
		Label:       "Sandbox",
		Description: "Lightweight dev namespace with resource quotas.",
		Detail:      "Best for experiments and personal sandboxes — no Fleet GitRepo.",
	},
	{
		ID:          "team",
		Label:       "Team environment",
		Description: "Shared namespace plus a Fleet GitRepo for GitOps.",
		Detail:      "Charts sync from environments/<name>/ in the platform Git repo.",
	},
	{
		ID:          "vcluster",
		Label:       "Virtual cluster",
		Description: "Isolated control plane (vCluster-style).",
		Detail:      "Requires the vCluster operator — full cluster isolation for a team.",
	},
}

func portalNamespace() string {
	if ns := os.Getenv("PLATFORM_NAMESPACE"); ns != "" {
		return ns
	}
	return "devportal-system"
}

func ensureClusterReady(ru *requestUser) error {
	if err := ensureKubeconfig(ru); err != nil {
		return fmt.Errorf("sync kubeconfig from Rancher: %w", err)
	}
	if err := ensurePlatformCRD(ru.Kubeconfig); err != nil {
		return fmt.Errorf("ensure PlatformRequest CRD: %w", err)
	}
	return nil
}

func ensurePlatformCRD(kubeCfg string) error {
	if _, err := runKubectlWithConfig(kubeCfg, "get", "crd", "platformrequests.platform.devportal.io"); err == nil {
		return nil
	}
	args := []string{"apply", "-f", "-"}
	if kubeCfg != "" {
		args = append([]string{"--kubeconfig", kubeCfg}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = strings.NewReader(string(platformRequestCRD))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func ensureNamespace(kubeCfg, name string) {
	_, err := runKubectlWithConfig(kubeCfg, "get", "namespace", name)
	if err == nil {
		return
	}
	_, _ = runKubectlWithConfig(kubeCfg, "create", "namespace", name)
}

func handlePortalCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"charts":    defaultCatalog,
		"templates": defaultTemplates,
	})
}

func handlePortalStack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"recommended": "Rancher + Fleet + Monitoring",
		"summary":     "Self-service environments provisioned as PlatformRequest CRs, reconciled into namespaces and Fleet bundles.",
		"components": []gin.H{
			{"name": "PlatformRequest CRD", "role": "Request lifecycle"},
			{"name": "Fleet GitRepo", "role": "GitOps delivery of selected charts"},
			{"name": "Namespace", "role": "env-{name} isolation"},
		},
	})
}

func parsePlatformRequestList(out string, listAll bool, ru *requestUser, kubeCfg string) ([]PlatformRequest, error) {
	var list struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				Namespace         string `json:"namespace"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec   PlatformRequestSpec   `json:"spec"`
			Status PlatformRequestStatus `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, err
	}
	var requests []PlatformRequest
	for _, item := range list.Items {
		if !listAll && ru != nil && item.Spec.Requester != "" &&
			item.Spec.Requester != ru.User.Username && item.Spec.Requester != ru.User.ID {
			continue
		}
		req := enrichFromRawItem(item)
		mergeLiveFleetStatus(kubeCfg, &req)
		requests = append(requests, req)
	}
	return requests, nil
}

func getPlatformRequest(kubeCfg, ns, crName string) (*PlatformRequest, error) {
	out, err := runKubectlWithConfig(kubeCfg, "get", crPlural+"."+crGroup+"/"+crName, "-n", ns, "-o", "json")
	if err != nil {
		return nil, err
	}
	var item struct {
		Metadata struct {
			Name              string `json:"name"`
			Namespace         string `json:"namespace"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec   PlatformRequestSpec   `json:"spec"`
		Status PlatformRequestStatus `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return nil, err
	}
	req := enrichFromRawItem(item)
	mergeLiveFleetStatus(kubeCfg, &req)
	return &req, nil
}

func canViewRequest(req *PlatformRequest, ru *requestUser, listAll bool) bool {
	if listAll {
		return true
	}
	if ru == nil || req == nil {
		return false
	}
	if req.Requester == "" {
		return true
	}
	return req.Requester == ru.User.Username || req.Requester == ru.User.ID
}

func handlePortalListRequests(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	listAll := evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).ListAllRequests

	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"requests": []PlatformRequest{},
			"listAll":  listAll,
			"warning":  err.Error(),
		})
		return
	}

	ns := portalNamespace()
	out, err := runKubectlWithConfig(ru.Kubeconfig, "get", crPlural+"."+crGroup, "-n", ns, "-o", "json")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"requests": []PlatformRequest{},
			"listAll":  listAll,
			"warning":  err.Error(),
		})
		return
	}
	requests, err := parsePlatformRequestList(out, listAll, ru, ru.Kubeconfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"listAll":  listAll,
		"gitRepo":  platformGitRepo(),
		"gitBranch": platformGitBranch(),
	})
}

func handlePortalGetRequest(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	listAll := evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).ListAllRequests
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	crName := c.Param("name")
	ns := portalNamespace()
	req, err := getPlatformRequest(ru.Kubeconfig, ns, crName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if !canViewRequest(req, ru, listAll) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed to view this request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": req})
}

type createRequestBody struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Template    string   `json:"template"`
	Charts      []string `json:"charts"`
}

func handlePortalCreateRequest(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var body createRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if body.Template == "" {
		body.Template = "sandbox"
	}

	ns := portalNamespace()
	ensureNamespace(ru.Kubeconfig, ns)

	crName := fmt.Sprintf("pr-%s-%d", body.Name, time.Now().Unix()%100000)
	requester := ru.User.Username
	if requester == "" {
		requester = ru.User.ID
	}

	cr := map[string]any{
		"apiVersion": crGroup + "/" + crVersion,
		"kind":       "PlatformRequest",
		"metadata": map[string]any{
			"name":      crName,
			"namespace": ns,
		},
		"spec": PlatformRequestSpec{
			Name:        body.Name,
			DisplayName: body.DisplayName,
			Description: body.Description,
			Template:    body.Template,
			Charts:      body.Charts,
			Requester:   requester,
		},
	}
	yamlBytes, _ := json.Marshal(cr)
	args := []string{"apply", "-f", "-"}
	if ru.Kubeconfig != "" {
		args = append([]string{"--kubeconfig", ru.Kubeconfig}, args...)
	}
	applyCmd := exec.Command("kubectl", args...)
	applyCmd.Stdin = strings.NewReader(string(yamlBytes))
	out, applyErr := applyCmd.CombinedOutput()
	if applyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("apply PlatformRequest: %s (%s)", applyErr, strings.TrimSpace(string(out))),
		})
		return
	}

	patchStatus(ru.Kubeconfig, ns, crName, "Pending", "Queued for provisioning")
	go provisionEnvironment(ru.Kubeconfig, ns, crName, body.Name, body.Template, body.Charts)

	c.JSON(http.StatusCreated, gin.H{
		"name":           body.Name,
		"crName":         crName,
		"message":        "Environment request accepted",
		"gitRepoUrl":     platformGitRepo(),
		"gitPath":        fmt.Sprintf("environments/%s", body.Name),
		"pullRequestHint": pullRequestHint(body.Name),
	})
}

func provisionEnvironment(kubeCfg, ns, crName, envName, template string, charts []string) {
	envNs := "env-" + envName
	ensureNamespace(kubeCfg, envNs)
	patchStatus(kubeCfg, ns, crName, "Provisioning", fmt.Sprintf("Created namespace %s", envNs))

	if template == "team" || template == "vcluster" {
		_, _ = runKubectlWithConfig(kubeCfg, "label", "namespace", envNs, "devportal.io/template="+template, "--overwrite")
	}
	for _, chart := range charts {
		_, _ = runKubectlWithConfig(kubeCfg, "annotate", "namespace", envNs, "devportal.io/chart-"+chart+"=requested", "--overwrite")
	}
	patchStatus(kubeCfg, ns, crName, "Ready", fmt.Sprintf("Environment %s is ready (charts: %v)", envNs, charts))
}

func patchStatus(kubeCfg, ns, name, phase, message string) {
	patch := map[string]any{
		"status": PlatformRequestStatus{Phase: phase, Message: message},
	}
	b, _ := json.Marshal(patch)
	_, _ = runKubectlWithConfig(kubeCfg, "patch", crPlural+"."+crGroup+"/"+name, "-n", ns, "--type", "merge", "-p", string(b))
}
