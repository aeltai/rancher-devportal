package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type CollectionEntry struct {
	ID          string `json:"id" yaml:"id"`
	Label       string `json:"label" yaml:"label"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Icon        string `json:"icon,omitempty" yaml:"icon,omitempty"`
	Weight      int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

type OfferingEntry struct {
	ID               string     `json:"id" yaml:"id"`
	CollectionID     string     `json:"collectionId" yaml:"collectionId"`
	Label            string     `json:"label" yaml:"label"`
	Description      string     `json:"description,omitempty" yaml:"description,omitempty"`
	Detail           string     `json:"detail,omitempty" yaml:"detail,omitempty"`
	Icon             string     `json:"icon,omitempty" yaml:"icon,omitempty"`
	Kind             string     `json:"kind" yaml:"kind"`
	Weight           int        `json:"weight,omitempty" yaml:"weight,omitempty"`
	GitOps           bool       `json:"gitOps,omitempty" yaml:"gitOps,omitempty"`
	RequiresApproval bool       `json:"requiresApproval,omitempty" yaml:"requiresApproval,omitempty"`
	Template         string     `json:"template,omitempty" yaml:"template,omitempty"`
	Charts           []string   `json:"charts,omitempty" yaml:"charts,omitempty"`
	ClusterType      string     `json:"clusterType,omitempty" yaml:"clusterType,omitempty"`
	CloneFrom        bool       `json:"cloneFrom,omitempty" yaml:"cloneFrom,omitempty"`
	APIVersion       string     `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	KindName         string     `json:"kindName,omitempty" yaml:"kindName,omitempty"`
	TargetCluster    string     `json:"targetCluster,omitempty" yaml:"targetCluster,omitempty"`
	ManifestTemplate string     `json:"manifestTemplate,omitempty" yaml:"manifestTemplate,omitempty"`
	FormSchema       []FormField `json:"formSchema,omitempty" yaml:"formSchema,omitempty"`
}

type FormField struct {
	Key         string   `json:"key" yaml:"key"`
	Label       string   `json:"label" yaml:"label"`
	Type        string   `json:"type" yaml:"type"`
	SpecPath    string   `json:"specPath,omitempty" yaml:"specPath,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
	Options     []string `json:"options,omitempty" yaml:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
}

type CloneFromRef struct {
	ClusterID   string `json:"clusterId"`
	ClusterName string `json:"clusterName,omitempty"`
	Namespace   string `json:"namespace"`
	ProjectID   string `json:"projectId,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
}

type ResolvedRequest struct {
	Template        string
	CollectionID    string
	OfferingID      string
	Charts          []string
	CustomResources []CustomResourceEntry
	CloneFromRef    *CloneFromRef
	GitOps          bool
	RequiresApproval bool
}

func collectionsFromConfig() []CollectionEntry {
	cfg := getPlatformConfig()
	migrateLegacyCatalog(&cfg)
	out := make([]CollectionEntry, len(cfg.Collections))
	copy(out, cfg.Collections)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight == out[j].Weight {
			return out[i].Label < out[j].Label
		}
		return out[i].Weight < out[j].Weight
	})
	return out
}

func offeringsFromConfig() []OfferingEntry {
	cfg := getPlatformConfig()
	migrateLegacyCatalog(&cfg)
	out := make([]OfferingEntry, len(cfg.Offerings))
	copy(out, cfg.Offerings)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight == out[j].Weight {
			return out[i].Label < out[j].Label
		}
		return out[i].Weight < out[j].Weight
	})
	return out
}

func offeringsForCollection(collectionID string) []OfferingEntry {
	var out []OfferingEntry
	for _, o := range offeringsFromConfig() {
		if o.CollectionID == collectionID {
			out = append(out, o)
		}
	}
	return out
}

func offeringByID(id string) (OfferingEntry, bool) {
	cfg := getPlatformConfig()
	migrateLegacyCatalog(&cfg)
	for _, o := range cfg.Offerings {
		if o.ID == id {
			return o, true
		}
	}
	return OfferingEntry{}, false
}

func migrateLegacyCatalog(cfg *PlatformConfig) {
	if len(cfg.Collections) > 0 && len(cfg.Offerings) > 0 {
		return
	}
	if len(cfg.Collections) == 0 {
		cfg.Collections = []CollectionEntry{
			{ID: "namespaces", Label: "Namespaces & Projects", Description: "Isolated namespaces for development and teams", Icon: "namespace", Weight: 10},
			{ID: "clusters", Label: "Clusters", Description: "Virtual or dedicated clusters", Icon: "cluster", Weight: 20},
			{ID: "platform-services", Label: "Platform Services", Description: "Helm-managed platform add-ons", Icon: "apps", Weight: 30},
			{ID: "virtual-machines", Label: "Virtual Machines", Description: "Harvester VM workloads", Icon: "vm", Weight: 40},
			{ID: "custom", Label: "Custom Offerings", Description: "Platform team custom manifests", Icon: "file", Weight: 50},
		}
	}
	if len(cfg.Offerings) > 0 {
		return
	}
	for _, t := range cfg.Templates {
		collectionID := "namespaces"
		kind := "namespace"
		switch t.ID {
		case "vcluster":
			collectionID = "clusters"
			kind = "cluster"
		case "team":
			collectionID = "namespaces"
		}
		cfg.Offerings = append(cfg.Offerings, OfferingEntry{
			ID:               t.ID,
			CollectionID:     collectionID,
			Label:            t.Label,
			Description:      t.Description,
			Detail:           t.Detail,
			Icon:             t.Icon,
			Kind:             kind,
			Template:         t.ID,
			GitOps:           t.GitOps,
			RequiresApproval: t.RequiresApproval,
		})
	}
	for _, c := range cfg.Charts {
		cfg.Offerings = append(cfg.Offerings, OfferingEntry{
			ID:               c.ID,
			CollectionID:     "platform-services",
			Label:            c.Name,
			Description:      c.Description,
			Icon:             "helm",
			Kind:             "helm",
			Charts:           []string{c.ID},
			GitOps:           true,
			RequiresApproval: true,
		})
	}
	for _, p := range cfg.CustomResourcePresets {
		collectionID := "custom"
		if strings.Contains(strings.ToLower(p.Kind), "virtualmachine") {
			collectionID = "virtual-machines"
		}
		cfg.Offerings = append(cfg.Offerings, OfferingEntry{
			ID:            p.ID,
			CollectionID:  collectionID,
			Label:         p.Name,
			Description:   p.Description,
			Icon:          "crd",
			Kind:          "crd",
			APIVersion:    p.APIVersion,
			KindName:      p.Kind,
			TargetCluster: cfg.CrdDiscovery.Clusters,
			FormSchema:    presetToFormSchema(p),
			GitOps:        true,
			RequiresApproval: true,
		})
	}
}

func presetToFormSchema(p CustomResourcePreset) []FormField {
	if strings.TrimSpace(p.DefaultSpec) == "" {
		return nil
	}
	var spec map[string]any
	if err := yaml.Unmarshal([]byte(p.DefaultSpec), &spec); err != nil {
		return nil
	}
	var fields []FormField
	flattenSpec("", spec, &fields)
	return fields
}

func flattenSpec(prefix string, v any, fields *[]FormField) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	for k, val := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch t := val.(type) {
		case map[string]any:
			flattenSpec(path, t, fields)
		default:
			*fields = append(*fields, FormField{
				Key:      strings.ReplaceAll(path, ".", "_"),
				Label:    humanizeKey(k),
				Type:     inferFieldType(val),
				SpecPath: path,
				Default:  fmt.Sprintf("%v", val),
			})
		}
	}
}

func humanizeKey(k string) string {
	k = strings.ReplaceAll(k, "-", " ")
	if len(k) > 0 {
		return strings.ToUpper(k[:1]) + k[1:]
	}
	return k
}

func inferFieldType(v any) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	default:
		return "text"
	}
}

func requestNeedsGitOpsFromOffering(offering OfferingEntry, charts []string, customResources []CustomResourceEntry) bool {
	if len(charts) > 0 || len(customResources) > 0 {
		return true
	}
	return offering.GitOps
}

func requestNeedsApprovalFromOffering(offering OfferingEntry, charts []string, customResources []CustomResourceEntry) bool {
	cfg := getPlatformConfig()
	if offering.RequiresApproval {
		return true
	}
	if cfg.Approval.ChartsRequireApproval && len(charts) > 0 {
		return true
	}
	if cfg.Approval.CustomResourcesRequireApproval && len(customResources) > 0 {
		return true
	}
	if offering.GitOps {
		return true
	}
	return false
}

func resolveOfferingRequest(offering OfferingEntry, envName, requester string, formValues map[string]string, selectedCharts []string, cloneFrom *CloneFromRef, kubeCfg string) (ResolvedRequest, error) {
	res := ResolvedRequest{
		Template:     offering.Template,
		CollectionID: offering.CollectionID,
		OfferingID:   offering.ID,
		CloneFromRef: cloneFrom,
		GitOps:       offering.GitOps,
	}
	if res.Template == "" {
		switch offering.Kind {
		case "namespace":
			res.Template = "sandbox"
		case "cluster":
			res.Template = "vcluster"
		case "helm":
			res.Template = "team"
		default:
			res.Template = "team"
		}
	}

	envNs := "env-" + envName
	switch offering.Kind {
	case "namespace", "cluster":
		if offering.GitOps {
			res.Charts = offering.Charts
		}
		if cloneFrom != nil && kubeCfg != "" {
			crs, err := cloneNamespaceResources(kubeCfg, cloneFrom, envNs, envName)
			if err != nil {
				return res, err
			}
			res.CustomResources = append(res.CustomResources, crs...)
		}
	case "helm":
		if len(selectedCharts) > 0 {
			res.Charts = selectedCharts
		} else {
			res.Charts = offering.Charts
		}
	case "crd":
		cr, err := buildCRDResource(offering, envName, envNs, formValues)
		if err != nil {
			return res, err
		}
		res.CustomResources = append(res.CustomResources, cr)
	case "generic":
		crs, err := buildGenericResources(offering, envName, envNs, formValues)
		if err != nil {
			return res, err
		}
		res.CustomResources = append(res.CustomResources, crs...)
	default:
		return res, fmt.Errorf("unknown offering kind %q", offering.Kind)
	}

	res.RequiresApproval = requestNeedsApprovalFromOffering(offering, res.Charts, res.CustomResources)
	res.GitOps = requestNeedsGitOpsFromOffering(offering, res.Charts, res.CustomResources)
	return res, nil
}

func buildCRDResource(offering OfferingEntry, envName, envNs string, formValues map[string]string) (CustomResourceEntry, error) {
	anyVals := map[string]any{}
	for k, v := range formValues {
		anyVals[k] = v
	}
	return buildCRDResourceAny(offering, envName, envNs, anyVals)
}

func buildCRDResourceAny(offering OfferingEntry, envName, envNs string, formValues map[string]any) (CustomResourceEntry, error) {
	spec := map[string]any{}
	for _, field := range offering.FormSchema {
		val := formValueForFieldAny(field, formValues)
		if val == nil && field.Default != "" {
			val = coerceFieldValue(field, field.Default)
		}
		if val != nil && field.SpecPath != "" {
			setNestedValue(spec, field.SpecPath, val)
		}
	}
	specYAML, err := yaml.Marshal(spec)
	if err != nil {
		return CustomResourceEntry{}, err
	}
	name := strings.ToLower(offering.KindName) + "-" + strings.TrimPrefix(envNs, "env-")
	return CustomResourceEntry{
		ID:         offering.ID,
		APIVersion: offering.APIVersion,
		Kind:       offering.KindName,
		Name:       name,
		Namespace:  envNs,
		SpecYAML:   string(specYAML),
	}, nil
}

func buildGenericResources(offering OfferingEntry, envName, envNs string, formValues map[string]string) ([]CustomResourceEntry, error) {
	if strings.TrimSpace(offering.ManifestTemplate) == "" {
		return nil, fmt.Errorf("generic offering %q has no manifestTemplate", offering.ID)
	}
	data := templateData(envName, envNs, formValues)
	tmpl, err := template.New("manifest").Parse(offering.ManifestTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse manifest template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render manifest template: %w", err)
	}
	docs := splitYAMLDocuments(buf.String())
	var out []CustomResourceEntry
	for i, doc := range docs {
		out = append(out, CustomResourceEntry{
			ID:           fmt.Sprintf("%s-%d", offering.ID, i),
			ManifestYAML: doc,
		})
	}
	return out, nil
}

func templateData(envName, envNs string, formValues map[string]string) map[string]any {
	data := map[string]any{
		"envName":   envName,
		"envNs":     envNs,
		"namespace": envNs,
		"name":      envName,
	}
	for k, v := range formValues {
		data[k] = v
	}
	return data
}

func splitYAMLDocuments(raw string) []string {
	parts := strings.Split(raw, "\n---")
	var docs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			docs = append(docs, p)
		}
	}
	return docs
}

func formValueForField(field FormField, values map[string]string) any {
	if v, ok := values[field.Key]; ok && strings.TrimSpace(v) != "" {
		return coerceFieldValue(field, v)
	}
	return nil
}

func formValueForFieldAny(field FormField, values map[string]any) any {
	if v, ok := values[field.Key]; ok {
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				return coerceFieldValue(field, t)
			}
		default:
			return v
		}
	}
	return nil
}

func coerceFieldValue(field FormField, raw string) any {
	switch field.Type {
	case "number":
		if n, err := strconv.ParseFloat(raw, 64); err == nil {
			return n
		}
	case "boolean":
		switch strings.ToLower(raw) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return raw
}

func setNestedValue(root map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	cur := root
	for i, part := range parts {
		if i == len(parts)-1 {
			cur[part] = value
			return
		}
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
}

func cloneNamespaceResources(kubeCfg string, ref *CloneFromRef, envNs, envName string) ([]CustomResourceEntry, error) {
	if ref == nil || ref.Namespace == "" {
		return nil, nil
	}
	ctxArgs, err := kubectlContextArgs(kubeCfg, ref.ClusterID)
	if err != nil {
		return nil, err
	}
	var out []CustomResourceEntry
	for _, kind := range []string{"resourcequota", "limitrange"} {
		args := append(ctxArgs, "get", kind, "-n", ref.Namespace, "-o", "json")
		raw, err := runKubectlWithConfig(kubeCfg, args...)
		if err != nil {
			continue
		}
		var list struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			continue
		}
		for _, item := range list.Items {
			cr, ok := cloneResourceItem(item, kind, envNs, envName)
			if ok {
				out = append(out, cr)
			}
		}
	}
	return out, nil
}

func cloneResourceItem(item map[string]any, kind, envNs, envName string) (CustomResourceEntry, bool) {
	meta, _ := item["metadata"].(map[string]any)
	if meta == nil {
		return CustomResourceEntry{}, false
	}
	name, _ := meta["name"].(string)
	if name == "" {
		return CustomResourceEntry{}, false
	}
	spec, _ := item["spec"].(map[string]any)
	if spec == nil {
		return CustomResourceEntry{}, false
	}
	specYAML, err := yaml.Marshal(spec)
	if err != nil {
		return CustomResourceEntry{}, false
	}
	apiVersion := "v1"
	kindTitle := "ResourceQuota"
	if kind == "limitrange" {
		kindTitle = "LimitRange"
	}
	return CustomResourceEntry{
		ID:         "clone-" + kind + "-" + envName,
		APIVersion: apiVersion,
		Kind:       kindTitle,
		Name:       name + "-" + envName,
		Namespace:  envNs,
		SpecYAML:   string(specYAML),
	}, true
}

func kubectlContextArgs(kubeCfg, clusterID string) ([]string, error) {
	if clusterID == "" || clusterID == "local" || clusterID == "management" {
		return nil, nil
	}
	out, err := runKubectlWithConfig(kubeCfg, "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, err
	}
	for _, ctx := range strings.Split(strings.TrimSpace(out), "\n") {
		ctx = strings.TrimSpace(ctx)
		if ctx == "" {
			continue
		}
		if ctx == clusterID || strings.Contains(ctx, clusterID) {
			return []string{"--context", ctx}, nil
		}
	}
	return []string{"--context", clusterID}, nil
}

func discoverCRDsForCluster(kubeCfg, clusterID string) ([]DiscoveredCRD, error) {
	ctxArgs, err := kubectlContextArgs(kubeCfg, clusterID)
	if err != nil {
		return nil, err
	}
	args := append(ctxArgs, "get", "crd", "-o", "json")
	out, err := runKubectlWithConfig(kubeCfg, args...)
	if err != nil {
		return nil, fmt.Errorf("list CRDs: %w", err)
	}
	return parseCRDList(out)
}

func parseCRDList(out string) ([]DiscoveredCRD, error) {
	cfg := getPlatformConfig()
	var list struct {
		Items []struct {
			Spec struct {
				Group    string `json:"group"`
				Names    struct {
					Kind   string `json:"kind"`
					Plural string `json:"plural"`
				} `json:"names"`
				Scope    string `json:"scope"`
				Versions []struct {
					Name    string `json:"name"`
					Storage bool   `json:"storage"`
				} `json:"versions"`
			} `json:"spec"`
			Metadata struct {
				Name        string            `json:"name"`
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, err
	}
	var result []DiscoveredCRD
	for _, item := range list.Items {
		group := item.Spec.Group
		if isGroupExcluded(group, cfg.CrdDiscovery.ExcludeGroups) {
			continue
		}
		version := ""
		for _, v := range item.Spec.Versions {
			if v.Storage {
				version = v.Name
				break
			}
		}
		if version == "" && len(item.Spec.Versions) > 0 {
			version = item.Spec.Versions[0].Name
		}
		kind := item.Spec.Names.Kind
		if kind == "" {
			continue
		}
		id := fmt.Sprintf("%s/%s/%s", group, version, kind)
		desc := item.Metadata.Annotations["api-approved.kubernetes.io"]
		if desc == "" {
			desc = item.Metadata.Annotations["description"]
		}
		var versions []string
		for _, v := range item.Spec.Versions {
			versions = append(versions, v.Name)
		}
		result = append(result, DiscoveredCRD{
			ID:          id,
			Group:       group,
			Version:     version,
			Kind:        kind,
			Plural:      item.Spec.Names.Plural,
			Scope:       item.Spec.Scope,
			APIVersion:  group + "/" + version,
			Description: desc,
			Versions:    versions,
		})
	}
	return result, nil
}

func listExistingNamespaces(kubeCfg, clusterID string) ([]map[string]string, error) {
	ctxArgs, err := kubectlContextArgs(kubeCfg, clusterID)
	if err != nil {
		return nil, err
	}
	args := append(ctxArgs, "get", "namespaces", "-o", "json")
	out, err := runKubectlWithConfig(kubeCfg, args...)
	if err != nil {
		return nil, err
	}
	var list struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, err
	}
	var result []map[string]string
	for _, item := range list.Items {
		name := item.Metadata.Name
		if strings.HasPrefix(name, "kube-") || name == "default" {
			continue
		}
		result = append(result, map[string]string{
			"type":      "namespace",
			"name":      name,
			"clusterId": clusterID,
		})
	}
	return result, nil
}

func testGitConnection(repoURL, branch, secretName, kubeCfg string) error {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return fmt.Errorf("repo URL is required")
	}
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		return fmt.Errorf("only http:// and https:// git URLs are supported")
	}
	authURL := repoURL
	if secretName != "" {
		ns := portalNamespace()
		out, err := runKubectlWithConfig(kubeCfg, "get", "secret", secretName, "-n", ns, "-o", "json")
		if err == nil {
			var sec struct {
				Data map[string]string `json:"data"`
			}
			if json.Unmarshal([]byte(out), &sec) == nil {
				creds := parseGitSecretFromJSON(decodeSecretData(sec.Data))
				if u, err := embedGitCreds(repoURL, creds); err == nil {
					authURL = u
				}
			}
		}
	}
	cmd := exec.Command("git", "ls-remote", "--heads", authURL, branch)
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git ls-remote failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

type gitCredsJSON struct {
	Username string
	Password string
	Token    string
}

func parseGitSecretFromJSON(data map[string]string) gitCredsJSON {
	decode := func(k string) string {
		if v, ok := data[k]; ok {
			return v
		}
		return ""
	}
	c := gitCredsJSON{
		Username: decode("username"),
		Password: decode("password"),
		Token:    decode("token"),
	}
	if c.Token == "" {
		c.Token = c.Password
	}
	if c.Username == "" {
		c.Username = "git"
	}
	return c
}

func embedGitCreds(repoURL string, creds gitCredsJSON) (string, error) {
	scheme := ""
	switch {
	case strings.HasPrefix(repoURL, "https://"):
		scheme = "https"
	case strings.HasPrefix(repoURL, "http://"):
		scheme = "http"
	default:
		return "", fmt.Errorf("unsupported scheme")
	}
	token := creds.Token
	if token == "" {
		return repoURL, nil
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(repoURL, "https://"), "http://")
	user := creds.Username
	if user == "" {
		user = "git"
	}
	return fmt.Sprintf("%s://%s:%s@%s", scheme, user, token, rest), nil
}
