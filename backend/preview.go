package main

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type GitPreviewFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type GitPreview struct {
	Tree      string           `json:"tree"`
	Files     []GitPreviewFile `json:"files"`
	GitRepo   string           `json:"gitRepo,omitempty"`
	GitBranch string           `json:"gitBranch,omitempty"`
	GitPath   string           `json:"gitPath,omitempty"`
}

type chartDef struct {
	Repo    string
	Chart   string
	Version string
}

var catalogCharts = map[string]chartDef{
	"rancher-monitoring": {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-monitoring"},
	"rancher-logging":    {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-logging"},
	"rancher-backup":     {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-backup-crd"},
	"fleet":              {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "fleet"},
	"cert-manager":       {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "rancher-cert-manager"},
	"ingress-nginx":      {Repo: "https://charts.rancher.io/server-charts/latest", Chart: "ingress-nginx"},
}

func needsApproval(template string, charts []string) bool {
	if template != "sandbox" {
		return true
	}
	for _, c := range charts {
		if strings.TrimSpace(c) != "" {
			return true
		}
	}
	return false
}

func canAdminActOnRequest(req *PlatformRequest) bool {
	if req == nil {
		return false
	}
	if req.Phase == "PendingApproval" {
		return true
	}
	if req.Phase == "Pending" && requestNeedsApprovalFromConfig(req.Template, req.Charts, req.CustomResources) {
		return true
	}
	return false
}

func buildGitPreview(spec PlatformRequestSpec, envName string) *GitPreview {
	gitPath := strings.Trim(spec.GitPath, "/")
	if gitPath == "" {
		gitPath = fmt.Sprintf("environments/%s", envName)
	}
	gitRepo := spec.GitRepo
	if gitRepo == "" {
		gitRepo = platformGitRepo()
	}
	gitBranch := spec.GitBranch
	if gitBranch == "" {
		gitBranch = platformGitBranch()
	}

	files := renderGitManifests(spec, envName, gitPath)
	tree := gitTreeFromFiles(gitRepo, files)

	var previewFiles []GitPreviewFile
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		previewFiles = append(previewFiles, GitPreviewFile{Path: p, Content: files[p]})
	}

	return &GitPreview{
		Tree:      tree,
		Files:     previewFiles,
		GitRepo:   gitRepo,
		GitBranch: gitBranch,
		GitPath:   gitPath,
	}
}

func renderGitManifests(spec PlatformRequestSpec, envName, gitPath string) map[string]string {
	envNs := "env-" + envName
	out := map[string]string{}

	ns := map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]any{
			"name": envNs,
			"labels": map[string]any{
				"devportal.io/env":       envName,
				"devportal.io/template":  spec.Template,
				"devportal.io/requester": sanitizePreviewLabel(spec.Requester),
			},
			"annotations": map[string]any{
				"devportal.io/display-name": spec.DisplayName,
			},
		},
	}
	out[gitPath+"/namespace.yaml"] = mustPreviewYAML(ns)

	fleetDoc := map[string]any{"defaultNamespace": envNs}
	if releases := helmPreviewReleases(spec.Charts, envNs); len(releases) > 0 {
		fleetDoc["helm"] = map[string]any{"releases": releases}
	}
	out[gitPath+"/fleet.yaml"] = mustPreviewYAML(fleetDoc)

	readme := fmt.Sprintf(`# Environment %s

- **Template:** %s
- **Requester:** %s
- **Namespace:** %s

Managed by PlatformRequest operator (Developer Portal).
`, envName, spec.Template, spec.Requester, envNs)
	out[gitPath+"/README.md"] = readme

	for _, cr := range spec.CustomResources {
		if strings.TrimSpace(cr.ManifestYAML) != "" {
			fname := strings.ToLower(cr.Kind) + "-" + cr.Name + ".yaml"
			if cr.Kind == "" {
				fname = "manifest-" + cr.ID + ".yaml"
			}
			out[gitPath+"/resources/"+fname] = cr.ManifestYAML
			continue
		}
		yamlContent := renderCustomResourceYAML(cr, envNs)
		if yamlContent == "" {
			continue
		}
		fname := strings.ToLower(cr.Kind) + "-" + cr.Name + ".yaml"
		out[gitPath+"/resources/"+fname] = yamlContent
	}

	return out
}

func renderCustomResourceYAML(cr CustomResourceEntry, envNs string) string {
	if strings.TrimSpace(cr.ManifestYAML) != "" {
		return cr.ManifestYAML
	}
	if cr.APIVersion == "" || cr.Kind == "" {
		return ""
	}
	ns := cr.Namespace
	if ns == "" {
		ns = envNs
	}
	obj := map[string]any{
		"apiVersion": cr.APIVersion,
		"kind":       cr.Kind,
		"metadata": map[string]any{
			"name":      cr.Name,
			"namespace": ns,
			"labels": map[string]any{
				"devportal.io/managed": "true",
			},
		},
	}
	if strings.TrimSpace(cr.SpecYAML) != "" {
		var spec map[string]any
		if err := yaml.Unmarshal([]byte(cr.SpecYAML), &spec); err == nil {
			obj["spec"] = spec
		}
	}
	return mustPreviewYAML(obj)
}

func helmPreviewReleases(charts []string, envNs string) []map[string]any {
	var releases []map[string]any
	for _, id := range charts {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		def := chartDef{Repo: "https://charts.rancher.io/server-charts/latest", Chart: id}
		if h, ok := chartHelmFromConfig(id); ok {
			def.Repo = h.Repo
			def.Chart = h.Chart
			def.Version = h.Version
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

func gitTreeFromFiles(repo string, files map[string]string) string {
	if repo == "" {
		repo = "git-repo"
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return repo + "/\n  (no files)"
	}
	lines := []string{repo + "/"}
	for i, p := range paths {
		prefix := "├── "
		if i == len(paths)-1 {
			prefix = "└── "
		}
		lines = append(lines, prefix+p)
	}
	return strings.Join(lines, "\n")
}

func mustPreviewYAML(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func sanitizePreviewLabel(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "@", "-at-")
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}
