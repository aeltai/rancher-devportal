package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type chartDef struct {
	Repo    string
	Chart   string
	Version string
}

// Catalog chart IDs → Helm chart coordinates (Rancher chart repo).
var catalogCharts = map[string]chartDef{
	"rancher-monitoring": {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-monitoring"},
	"rancher-logging":  {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-logging"},
	"rancher-backup":   {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-backup-crd"},
	"fleet":            {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "fleet"},
	"cert-manager":     {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-cert-manager"},
	"ingress-nginx":    {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "ingress-nginx"},
}

type renderInput struct {
	EnvName         string
	DisplayName     string
	Description     string
	Template        string
	Charts          []string
	Requester       string
	GitPath         string
	TargetClusters  []string
}

func defaultGitPath(envName string) string {
	return fmt.Sprintf("environments/%s", envName)
}

func renderManifests(in renderInput) map[string]string {
	envNs := "env-" + in.EnvName
	gitPath := strings.Trim(in.GitPath, "/")
	if gitPath == "" {
		gitPath = defaultGitPath(in.EnvName)
	}

	out := map[string]string{}

	ns := map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]any{
			"name": envNs,
			"labels": map[string]any{
				"devportal.io/env":       in.EnvName,
				"devportal.io/template":  in.Template,
				"devportal.io/requester": sanitizeLabel(in.Requester),
			},
			"annotations": map[string]any{
				"devportal.io/display-name": in.DisplayName,
			},
		},
	}
	out[gitPath+"/namespace.yaml"] = mustYAML(ns)

	fleetDoc := map[string]any{
		"defaultNamespace": envNs,
	}
	releases := helmReleases(in.Charts, envNs)
	if len(releases) > 0 {
		fleetDoc["helm"] = map[string]any{"releases": releases}
	}
	out[gitPath+"/fleet.yaml"] = mustYAML(fleetDoc)

	readme := fmt.Sprintf(`# Environment %s

- **Template:** %s
- **Requester:** %s
- **Namespace:** %s

Managed by PlatformRequest operator (Developer Portal).
`, in.EnvName, in.Template, in.Requester, envNs)
	out[gitPath+"/README.md"] = readme

	return out
}

func helmReleases(charts []string, envNs string) []map[string]any {
	var releases []map[string]any
	for _, id := range charts {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		def, ok := catalogCharts[id]
		if !ok {
			def = chartDef{Repo: "https://charts.rancher.io/server-charts/latest", Chart: id}
		}
		release := map[string]any{
			"name":      id,
			"chart":     def.Chart,
			"repo":      def.Repo,
			"namespace": envNs,
		}
		if def.Version != "" {
			release["version"] = def.Version
		}
		releases = append(releases, release)
	}
	return releases
}

func renderFleetGitRepo(name, repo, branch, gitPath string, targetClusters []string) map[string]any {
	spec := map[string]any{
		"repo":   repo,
		"branch": branch,
		"paths":  []string{gitPath},
	}
	spec["targets"] = fleetTargets(targetClusters)
	return map[string]any{
		"apiVersion": "fleet.cattle.io/v1alpha1",
		"kind":       "GitRepo",
		"metadata": map[string]any{
			"name": name,
			"labels": map[string]any{
				"devportal.io/managed": "true",
			},
		},
		"spec": spec,
	}
}

func fleetTargets(clusterNames []string) []map[string]any {
	if len(clusterNames) == 0 {
		return []map[string]any{
			{"name": "all-clusters", "clusterSelector": map[string]any{}},
		}
	}
	return []map[string]any{
		{"name": "selected-clusters", "clusterNames": clusterNames},
	}
}

func mustYAML(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func sanitizeLabel(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "@", "-at-")
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}
