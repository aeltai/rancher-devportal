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
	ID               string `json:"id"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Detail           string `json:"detail,omitempty"`
	Icon             string `json:"icon,omitempty"`
	GitOps           bool   `json:"gitOps,omitempty"`
	RequiresApproval bool   `json:"requiresApproval,omitempty"`
}

type CustomResourceEntry struct {
	ID           string `json:"id,omitempty"`
	APIVersion   string `json:"apiVersion"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace,omitempty"`
	SpecYAML     string `json:"specYaml,omitempty"`
	ManifestYAML string `json:"manifestYaml,omitempty"`
}

type PlatformRequestSpec struct {
	Name           string                `json:"name"`
	DisplayName    string                `json:"displayName"`
	Description    string                `json:"description"`
	Template       string                `json:"template"`
	OfferingID     string                `json:"offeringId,omitempty"`
	CollectionID   string                `json:"collectionId,omitempty"`
	CloneFromRef   *CloneFromRef         `json:"cloneFromRef,omitempty"`
	Charts         []string              `json:"charts"`
	CustomResources []CustomResourceEntry `json:"customResources,omitempty"`
	Requester      string                `json:"requester"`
	GitRepo        string                `json:"gitRepo,omitempty"`
	GitBranch      string                `json:"gitBranch,omitempty"`
	GitPath        string                `json:"gitPath,omitempty"`
	GitSecretName  string                `json:"gitSecretName,omitempty"`
	TargetClusters []string              `json:"targetClusters,omitempty"`
	FormValues     map[string]string     `json:"formValues,omitempty"`
}

type PlatformRequestStatus struct {
	Phase            string `json:"phase"`
	Message          string `json:"message"`
	GitCommit        string `json:"gitCommit,omitempty"`
	FleetGitRepoName string `json:"fleetGitRepoName,omitempty"`
	NamespaceName    string `json:"namespaceName,omitempty"`
	ApprovedBy       string `json:"approvedBy,omitempty"`
}

type PlatformRequest struct {
	CRName          string          `json:"crName"`
	Name            string          `json:"name"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description"`
	Namespace       string          `json:"namespace"`
	Template         string                `json:"template"`
	OfferingID       string                `json:"offeringId,omitempty"`
	CollectionID     string                `json:"collectionId,omitempty"`
	CloneFromRef     *CloneFromRef         `json:"cloneFromRef,omitempty"`
	Charts           []string              `json:"charts"`
	CustomResources  []CustomResourceEntry `json:"customResources,omitempty"`
	Requester        string                `json:"requester"`
	Phase           string          `json:"phase"`
	Message         string          `json:"message"`
	CreatedAt       string          `json:"createdAt"`
	ManifestYAML    string          `json:"manifestYaml,omitempty"`
	FleetResources  []FleetResource `json:"fleetResources,omitempty"`
	GitRepoURL       string          `json:"gitRepoUrl,omitempty"`
	GitBranch        string          `json:"gitBranch,omitempty"`
	GitPath          string          `json:"gitPath,omitempty"`
	GitSecretName    string          `json:"gitSecretName,omitempty"`
	TargetClusters   []string        `json:"targetClusters,omitempty"`
	GitCommit        string          `json:"gitCommit,omitempty"`
	FleetGitRepoName string          `json:"fleetGitRepoName,omitempty"`
	GitPreview       *GitPreview     `json:"gitPreview,omitempty"`
	PullRequestHint  string          `json:"pullRequestHint,omitempty"`
	ApprovedBy       string          `json:"approvedBy,omitempty"`
}

var defaultCatalog = []ChartEntry{}
var defaultTemplates = []TemplateEntry{}

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
	_, err := runKubectlWithConfig(kubeCfg, "get", "crd", "platformrequests.platform.devportal.io")
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	// Non-admins cannot read or install CRDs; chart/operator must pre-install the CRD.
	if strings.Contains(errMsg, "Forbidden") {
		return nil
	}
	if !strings.Contains(errMsg, "NotFound") && !strings.Contains(errMsg, "not found") {
		return err
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
	charts := catalogFromConfig()
	templates := templatesFromConfig()
	if len(charts) == 0 {
		charts = defaultCatalog
	}
	if len(templates) == 0 {
		templates = defaultTemplates
	}
	cfg := getPlatformConfig()
	c.JSON(http.StatusOK, gin.H{
		"charts":       charts,
		"templates":    templates,
		"collections":  collectionsFromConfig(),
		"offerings":    offeringsFromConfig(),
		"git":          cfg.Git,
		"defaults":     cfg.Defaults,
		"presets":      cfg.CustomResourcePresets,
		"approval":     cfg.Approval,
		"crdDiscovery": cfg.CrdDiscovery,
	})
}

func handlePortalStack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"recommended": "Rancher + Fleet + platform-operator",
		"summary":     "Self-service environments as PlatformRequest CRs, reconciled by platform-operator into namespaces, Git manifests, and Fleet GitRepos.",
		"components": []gin.H{
			{"name": "PlatformRequest CRD", "role": "Request lifecycle"},
			{"name": "platform-operator", "role": "Git push + Fleet GitRepo reconcile"},
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
		"gitRepo":  defaultGitRepoFromConfig(),
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
	Name            string                `json:"name"`
	DisplayName     string                `json:"displayName"`
	Description     string                `json:"description"`
	Template        string                `json:"template"`
	OfferingID      string                `json:"offeringId"`
	CollectionID    string                `json:"collectionId"`
	CloneFromRef    *CloneFromRef         `json:"cloneFromRef"`
	FormValues      map[string]string     `json:"formValues"`
	Charts          []string              `json:"charts"`
	CustomResources []CustomResourceEntry `json:"customResources"`
	GitRepo         string                `json:"gitRepo"`
	GitRepoID       string                `json:"gitRepoId"`
	GitBranch       string                `json:"gitBranch"`
	GitPath         string                `json:"gitPath"`
	GitSecretName   string                `json:"gitSecretName"`
	TargetClusters  []string              `json:"targetClusters"`
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

	var offering OfferingEntry
	var resolved ResolvedRequest
	if body.OfferingID != "" {
		var ok bool
		offering, ok = offeringByID(body.OfferingID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown offering %q", body.OfferingID)})
			return
		}
		var err error
		resolved, err = resolveOfferingRequest(offering, body.Name, "", body.FormValues, body.Charts, body.CloneFromRef, ru.Kubeconfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		body.Template = resolved.Template
		body.Charts = resolved.Charts
		body.CustomResources = resolved.CustomResources
		body.CollectionID = resolved.CollectionID
	} else {
		if body.Template == "" {
			if ts := templatesFromConfig(); len(ts) > 0 {
				body.Template = ts[0].ID
			} else {
				body.Template = "sandbox"
			}
		}
		if _, ok := templateFromConfig(body.Template); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown template %q", body.Template)})
			return
		}
	}

	if body.Template == "" {
		body.Template = "sandbox"
	}

	def := getPlatformConfig().Defaults
	if body.GitRepoID != "" {
		if repo, ok := gitRepoByID(body.GitRepoID); ok {
			body.GitRepo = repo.URL
			if body.GitBranch == "" {
				body.GitBranch = repo.Branch
			}
			if body.GitSecretName == "" {
				body.GitSecretName = repo.SecretName
			}
		}
	}
	if strings.TrimSpace(body.GitRepo) == "" {
		body.GitRepo = defaultGitRepoFromConfig()
	}

	needsGitOps := false
	requiresApproval := false
	if body.OfferingID != "" {
		needsGitOps = resolved.GitOps
		requiresApproval = resolved.RequiresApproval
	} else {
		needsGitOps = requestNeedsGitOpsFromConfig(body.Template, body.Charts, body.CustomResources)
		requiresApproval = requestNeedsApprovalFromConfig(body.Template, body.Charts, body.CustomResources)
	}
	if needsGitOps && strings.TrimSpace(body.GitRepo) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gitRepo is required for this template or when charts/custom resources are selected"})
		return
	}
	if body.GitBranch == "" {
		body.GitBranch = def.GitBranch
		if body.GitBranch == "" {
			body.GitBranch = platformGitBranch()
		}
	}
	if body.GitPath == "" {
		prefix := def.GitPathPrefix
		if prefix == "" {
			prefix = "environments"
		}
		body.GitPath = fmt.Sprintf("%s/%s", prefix, body.Name)
	}
	if body.GitSecretName == "" {
		body.GitSecretName = def.GitSecretName
		if body.GitSecretName == "" {
			body.GitSecretName = platformGitSecretName()
		}
	}

	for i := range body.CustomResources {
		if body.CustomResources[i].Namespace == "" {
			body.CustomResources[i].Namespace = "env-" + body.Name
		}
		if body.CustomResources[i].Name == "" {
			body.CustomResources[i].Name = strings.ToLower(body.CustomResources[i].Kind) + "-" + body.Name
		}
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
			Name:            body.Name,
			DisplayName:     body.DisplayName,
			Description:     body.Description,
			Template:        body.Template,
			OfferingID:      body.OfferingID,
			CollectionID:    body.CollectionID,
			CloneFromRef:    body.CloneFromRef,
			Charts:          body.Charts,
			CustomResources: body.CustomResources,
			Requester:       requester,
			GitRepo:         strings.TrimSpace(body.GitRepo),
			GitBranch:       body.GitBranch,
			GitPath:         body.GitPath,
			GitSecretName:   body.GitSecretName,
			TargetClusters:  body.TargetClusters,
			FormValues:      body.FormValues,
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

	if requiresApproval {
		patchStatus(ru.Kubeconfig, ns, crName, "PendingApproval", "Waiting for platform admin approval before Git push and Fleet provisioning")
		msg := "Environment request submitted — waiting for admin approval"
		c.JSON(http.StatusCreated, gin.H{
			"name":            body.Name,
			"crName":          crName,
			"phase":           "PendingApproval",
			"message":         msg,
			"gitRepoUrl":      strings.TrimSpace(body.GitRepo),
			"gitBranch":       body.GitBranch,
			"gitPath":         body.GitPath,
			"gitSecretName":   body.GitSecretName,
			"targetClusters":  body.TargetClusters,
			"gitPreview":      buildGitPreview(PlatformRequestSpec{
				Name: body.Name, DisplayName: body.DisplayName, Description: body.Description,
				Template: body.Template, Charts: body.Charts, CustomResources: body.CustomResources, Requester: requester,
				GitRepo: strings.TrimSpace(body.GitRepo), GitBranch: body.GitBranch, GitPath: body.GitPath,
				TargetClusters: body.TargetClusters,
			}, body.Name),
			"pullRequestHint": pullRequestHint(body.Name, body.GitRepo, body.GitBranch, body.GitPath),
		})
		return
	}

	patchStatus(ru.Kubeconfig, ns, crName, "Approved", "Auto-approved sandbox request — queued for operator")
	c.JSON(http.StatusCreated, gin.H{
		"name":            body.Name,
		"crName":          crName,
		"phase":           "Approved",
		"message":         "Environment request accepted — platform operator will reconcile",
		"gitRepoUrl":      strings.TrimSpace(body.GitRepo),
		"gitBranch":       body.GitBranch,
		"gitPath":         body.GitPath,
		"gitSecretName":   body.GitSecretName,
		"targetClusters":  body.TargetClusters,
		"pullRequestHint": pullRequestHint(body.Name, body.GitRepo, body.GitBranch, body.GitPath),
	})
}

func handlePortalApproveRequest(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin approval required"})
		return
	}
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
	if !canAdminActOnRequest(req) {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("request phase is %q and cannot be approved", req.Phase)})
		return
	}

	approver := ru.User.Username
	if approver == "" {
		approver = ru.User.ID
	}
	patchStatusFull(ru.Kubeconfig, ns, crName, PlatformRequestStatus{
		Phase:      "Approved",
		Message:    fmt.Sprintf("Approved by %s — queued for operator reconciliation", approver),
		ApprovedBy: approver,
	})

	req, _ = getPlatformRequest(ru.Kubeconfig, ns, crName)
	c.JSON(http.StatusOK, gin.H{"message": "Request approved", "request": req})
}

func handlePortalRejectRequest(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin approval required"})
		return
	}
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
	if !canAdminActOnRequest(req) {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("request phase is %q and cannot be rejected", req.Phase)})
		return
	}

	reason := strings.TrimSpace(c.Query("reason"))
	msg := "Rejected by platform admin"
	if reason != "" {
		msg += ": " + reason
	}
	rejector := ru.User.Username
	if rejector == "" {
		rejector = ru.User.ID
	}
	msg = fmt.Sprintf("%s (%s)", msg, rejector)
	patchStatus(ru.Kubeconfig, ns, crName, "Rejected", msg)

	req, _ = getPlatformRequest(ru.Kubeconfig, ns, crName)
	c.JSON(http.StatusOK, gin.H{"message": "Request rejected", "request": req})
}

func patchStatus(kubeCfg, ns, name, phase, message string) {
	patchStatusFull(kubeCfg, ns, name, PlatformRequestStatus{Phase: phase, Message: message})
}

func patchStatusFull(kubeCfg, ns, name string, status PlatformRequestStatus) {
	patch := map[string]any{"status": status}
	b, _ := json.Marshal(patch)
	_, _ = runKubectlWithConfig(kubeCfg, "patch", crPlural+"."+crGroup+"/"+name, "-n", ns,
		"--subresource", "status", "--type", "merge", "-p", string(b))
}
