package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
	Name        string                  `json:"name"`
	DisplayName string                  `json:"displayName"`
	Namespace   string                  `json:"namespace"`
	Template    string                  `json:"template"`
	Charts      []string                `json:"charts"`
	Phase       string                  `json:"phase"`
	Message     string                  `json:"message"`
	CreatedAt   string                  `json:"createdAt"`
	Spec        PlatformRequestSpec     `json:"-"`
	Status      PlatformRequestStatus   `json:"-"`
}

func portalNamespace() string {
	if ns := os.Getenv("PLATFORM_NAMESPACE"); ns != "" {
		return ns
	}
	return "devportal-system"
}

func kubeconfigPath() string {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p
	}
	return ""
}

func runKubectl(args ...string) (string, error) {
	kubeCfg := os.Getenv("KUBECONFIG")
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

var defaultCatalog = []ChartEntry{
	{ID: "rancher-monitoring", Name: "Monitoring", Category: "observability", Description: "Prometheus + Grafana stack"},
	{ID: "rancher-logging", Name: "Logging", Category: "observability", Description: "Banzai Cloud logging operator"},
	{ID: "rancher-backup", Name: "Rancher Backup", Category: "backup", Description: "Backup/restore for Rancher"},
	{ID: "fleet", Name: "Fleet", Category: "gitops", Description: "GitOps continuous delivery"},
	{ID: "cert-manager", Name: "cert-manager", Category: "security", Description: "TLS certificate automation"},
	{ID: "ingress-nginx", Name: "Ingress NGINX", Category: "networking", Description: "Ingress controller"},
}

var defaultTemplates = []TemplateEntry{
	{ID: "sandbox", Label: "Sandbox", Description: "Single namespace, dev quotas, no production SLAs"},
	{ID: "team", Label: "Team", Description: "Namespace + Fleet GitRepo for team GitOps"},
	{ID: "vcluster", Label: "Virtual cluster", Description: "vCluster-style isolated control plane (requires operator)"},
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

func handlePortalListRequests(c *gin.Context) {
	ru, _ := requestUserFromContext(c)
	ns := portalNamespace()
	out, err := runKubectl("get", crPlural+"."+crGroup, "-n", ns, "-o", "json")
	if err != nil {
		// CRD may not exist yet — return empty list
		c.JSON(http.StatusOK, gin.H{"requests": []PlatformRequest{}})
		return
	}
	var list struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec   PlatformRequestSpec   `json:"spec"`
			Status PlatformRequestStatus `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var requests []PlatformRequest
	for _, item := range list.Items {
		if ru != nil && item.Spec.Requester != "" && item.Spec.Requester != ru.User.Username && item.Spec.Requester != ru.User.ID {
			continue
		}
		phase := item.Status.Phase
		if phase == "" {
			phase = "Pending"
		}
		requests = append(requests, PlatformRequest{
			Name:        item.Spec.Name,
			DisplayName: item.Spec.DisplayName,
			Namespace:   "env-" + item.Spec.Name,
			Template:    item.Spec.Template,
			Charts:      item.Spec.Charts,
			Phase:       phase,
			Message:     item.Status.Message,
			CreatedAt:   item.Metadata.CreationTimestamp,
		})
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
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
	ensureNamespace(ns)

	crName := fmt.Sprintf("pr-%s-%d", body.Name, time.Now().Unix()%100000)
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
			Requester:   ru.User.Username,
		},
		"status": PlatformRequestStatus{Phase: "Pending", Message: "Queued for provisioning"},
	}
	yamlBytes, _ := json.Marshal(cr)
	applyCmd := exec.Command("kubectl", "apply", "-f", "-")
	if kubeCfg := os.Getenv("KUBECONFIG"); kubeCfg != "" {
		applyCmd.Args = append([]string{"kubectl", "--kubeconfig", kubeCfg, "apply", "-f", "-"})
	}
	applyCmd.Stdin = strings.NewReader(string(yamlBytes))
	out, applyErr := applyCmd.CombinedOutput()
	if applyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("apply PlatformRequest: %s (%s)", applyErr, strings.TrimSpace(string(out)))})
		return
	}

	go provisionEnvironment(ns, crName, body.Name, body.Template, body.Charts)

	c.JSON(http.StatusCreated, gin.H{
		"name":    body.Name,
		"crName":  crName,
		"message": "Environment request accepted",
	})
}

func ensureNamespace(name string) {
	_, err := runKubectl("get", "namespace", name)
	if err == nil {
		return
	}
	_, _ = runKubectl("create", "namespace", name)
}

func provisionEnvironment(ns, crName, envName, template string, charts []string) {
	envNs := "env-" + envName
	ensureNamespace(envNs)
	patchStatus(ns, crName, "Provisioning", fmt.Sprintf("Created namespace %s", envNs))

	if template == "team" || template == "vcluster" {
		_, _ = runKubectl("label", "namespace", envNs, "devportal.io/template="+template, "--overwrite")
	}
	for _, chart := range charts {
		_, _ = runKubectl("annotate", "namespace", envNs, "devportal.io/chart-"+chart+"=requested", "--overwrite")
	}
	patchStatus(ns, crName, "Ready", fmt.Sprintf("Environment %s is ready (charts: %v)", envNs, charts))
}

func patchStatus(ns, name, phase, message string) {
	patch := map[string]any{
		"status": PlatformRequestStatus{Phase: phase, Message: message},
	}
	b, _ := json.Marshal(patch)
	args := []string{"patch", crPlural + "." + crGroup + "/" + name, "-n", ns, "--type", "merge", "-p", string(b)}
	if kubeCfg := os.Getenv("KUBECONFIG"); kubeCfg != "" {
		args = append([]string{"--kubeconfig", kubeCfg}, args...)
	}
	_ = exec.Command("kubectl", args...).Run()
}
