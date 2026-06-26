package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type FleetResource struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	RepoURL     string `json:"repoUrl,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Path        string `json:"path,omitempty"`
	Phase       string `json:"phase"`
	Description string `json:"description,omitempty"`
}

func platformGitRepo() string {
	if r := strings.TrimSpace(os.Getenv("PLATFORM_GIT_REPO")); r != "" {
		return r
	}
	return "https://github.com/aeltai/rancher-devportal"
}

func platformGitBranch() string {
	if b := strings.TrimSpace(os.Getenv("PLATFORM_GIT_BRANCH")); b != "" {
		return b
	}
	return "main"
}

func platformFleetNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("PLATFORM_FLEET_NAMESPACE")); ns != "" {
		return ns
	}
	return "fleet-default"
}

func buildFleetPlan(envName, template string, charts []string) []FleetResource {
	envNs := "env-" + envName
	repo := platformGitRepo()
	branch := platformGitBranch()
	gitPath := fmt.Sprintf("environments/%s", envName)
	fleetNs := platformFleetNamespace()

	resources := []FleetResource{
		{
			Kind:        "Namespace",
			Name:        envNs,
			Namespace:   "",
			Phase:       "planned",
			Description: "Isolated environment namespace",
		},
		{
			Kind:        "PlatformRequest",
			Name:        fmt.Sprintf("pr-%s", envName),
			Namespace:   portalNamespace(),
			Phase:       "planned",
			Description: "Self-service request CR (this manifest)",
		},
	}

	if template == "team" || template == "vcluster" || len(charts) > 0 {
		resources = append(resources, FleetResource{
			Kind:        "GitRepo",
			Name:        "fleet-" + envName,
			Namespace:   fleetNs,
			RepoURL:     repo,
			Branch:      branch,
			Path:        gitPath,
			Phase:       "planned",
			Description: "Fleet GitRepo — chart manifests synced from Git",
		})
	}

	for _, chart := range charts {
		if chart == "" {
			continue
		}
		resources = append(resources, FleetResource{
			Kind:        "Bundle",
			Name:        chart + "-" + envName,
			Namespace:   envNs,
			RepoURL:     repo,
			Branch:      branch,
			Path:        gitPath + "/charts/" + chart,
			Phase:       "planned",
			Description: "Fleet bundle for Helm chart " + chart,
		})
	}

	return resources
}

func pullRequestHint(envName string) string {
	repo := platformGitRepo()
	branch := platformGitBranch()
	path := fmt.Sprintf("environments/%s", envName)
	return fmt.Sprintf(
		"Future workflow: automation opens a pull request against %s (branch %s) under %s/ with Fleet bundles and namespace manifests.",
		repo, branch, path,
	)
}

func objectToYAML(obj any) string {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(b)
}

func enrichFromRawItem(item struct {
	Metadata struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		CreationTimestamp string `json:"creationTimestamp"`
	} `json:"metadata"`
	Spec   PlatformRequestSpec   `json:"spec"`
	Status PlatformRequestStatus `json:"status"`
}) PlatformRequest {
	envName := item.Spec.Name
	if envName == "" {
		envName = item.Metadata.Name
	}
	phase := item.Status.Phase
	if phase == "" {
		phase = "Pending"
	}

	obj := map[string]any{
		"apiVersion": crGroup + "/" + crVersion,
		"kind":       "PlatformRequest",
		"metadata": map[string]any{
			"name":              item.Metadata.Name,
			"namespace":         item.Metadata.Namespace,
			"creationTimestamp": item.Metadata.CreationTimestamp,
		},
		"spec":   item.Spec,
		"status": item.Status,
	}

	return PlatformRequest{
		CRName:          item.Metadata.Name,
		Name:            envName,
		DisplayName:     item.Spec.DisplayName,
		Description:     item.Spec.Description,
		Namespace:       "env-" + envName,
		Template:        item.Spec.Template,
		Charts:          item.Spec.Charts,
		Requester:       item.Spec.Requester,
		Phase:           phase,
		Message:         item.Status.Message,
		CreatedAt:       item.Metadata.CreationTimestamp,
		ManifestYAML:    objectToYAML(obj),
		FleetResources:  buildFleetPlan(envName, item.Spec.Template, item.Spec.Charts),
		GitRepoURL:      platformGitRepo(),
		GitBranch:       platformGitBranch(),
		GitPath:         fmt.Sprintf("environments/%s", envName),
		PullRequestHint: pullRequestHint(envName),
	}
}

func mergeLiveFleetStatus(kubeCfg string, req *PlatformRequest) {
	if kubeCfg == "" || req == nil {
		return
	}
	gitRepoName := "fleet-" + req.Name
	out, err := runKubectlWithConfig(kubeCfg, "get", "gitrepos.fleet.cattle.io", gitRepoName,
		"-n", platformFleetNamespace(), "-o", "json")
	if err == nil {
		var gr struct {
			Status struct {
				DisplayState string `json:"displayState"`
			} `json:"status"`
		}
		if json.Unmarshal([]byte(out), &gr) == nil && gr.Status.DisplayState != "" {
			for i := range req.FleetResources {
				if req.FleetResources[i].Kind == "GitRepo" && req.FleetResources[i].Name == gitRepoName {
					req.FleetResources[i].Phase = strings.ToLower(gr.Status.DisplayState)
				}
			}
		}
	}
}
