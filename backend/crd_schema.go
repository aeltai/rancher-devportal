package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type CRDFormSchemaResponse struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Group      string      `json:"group"`
	Version    string      `json:"version"`
	Plural     string      `json:"plural"`
	Scope      string      `json:"scope"`
	Fields     []FormField `json:"fields"`
	FieldCount int         `json:"fieldCount"`
	Truncated  bool        `json:"truncated"`
}

func handlePortalCRDFormSchema(c *gin.Context) {
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

	clusterID := strings.TrimSpace(c.Query("cluster"))
	if clusterID == "" {
		clusterID = getPlatformConfig().CrdDiscovery.Clusters
	}
	if clusterID == "" {
		clusterID = "local"
	}

	group := strings.TrimSpace(c.Query("group"))
	version := strings.TrimSpace(c.Query("version"))
	kind := strings.TrimSpace(c.Query("kind"))
	if id := strings.TrimSpace(c.Query("id")); id != "" && (group == "" || version == "" || kind == "") {
		parts := strings.Split(id, "/")
		if len(parts) == 3 {
			group, version, kind = parts[0], parts[1], parts[2]
		}
	}
	if group == "" || version == "" || kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group, version, and kind query params required (or id=group/version/kind)"})
		return
	}

	fields, plural, scope, truncated, err := crdFormFieldsFromCluster(ru.Kubeconfig, clusterID, group, version, kind)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, CRDFormSchemaResponse{
		APIVersion: group + "/" + version,
		Kind:       kind,
		Group:      group,
		Version:    version,
		Plural:     plural,
		Scope:      scope,
		Fields:     fields,
		FieldCount: len(fields),
		Truncated:  truncated,
	})
}

func crdFormFieldsFromCluster(kubeCfg, clusterID, group, version, kind string) ([]FormField, string, string, bool, error) {
	crdJSON, err := fetchCRDJSON(kubeCfg, clusterID, group, kind)
	if err != nil {
		return nil, "", "", false, err
	}
	schema, plural, scope, err := openAPISchemaForCRD(crdJSON, version)
	if err != nil {
		return nil, "", "", false, err
	}
	fields, truncated := openAPISchemaToFormFields(schema, 4, 30)
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].Required != fields[j].Required {
			return fields[i].Required && !fields[j].Required
		}
		return fields[i].Label < fields[j].Label
	})
	return fields, plural, scope, truncated, nil
}

func fetchCRDJSON(kubeCfg, clusterID, group, kind string) (map[string]any, error) {
	crds, err := discoverCRDsForCluster(kubeCfg, clusterID)
	if err != nil {
		return nil, err
	}
	var plural string
	for _, c := range crds {
		if c.Group == group && c.Kind == kind {
			plural = c.Plural
			break
		}
	}
	if plural == "" {
		plural = guessCRDPlural(kind)
	}
	ctxArgs, err := kubectlContextArgs(kubeCfg, clusterID)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s.%s", plural, group)
	args := append(ctxArgs, "get", "crd", name, "-o", "json")
	out, err := runKubectlWithConfig(kubeCfg, args...)
	if err != nil {
		return nil, fmt.Errorf("get CRD %s: %w", name, err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func guessCRDPlural(kind string) string {
	if kind == "" {
		return ""
	}
	lower := strings.ToLower(kind)
	if strings.HasSuffix(lower, "s") {
		return lower + "es"
	}
	if strings.HasSuffix(lower, "y") {
		return lower[:len(lower)-1] + "ies"
	}
	return lower + "s"
}

func openAPISchemaForCRD(crd map[string]any, version string) (map[string]any, string, string, error) {
	spec, _ := crd["spec"].(map[string]any)
	if spec == nil {
		return nil, "", "", fmt.Errorf("CRD has no spec")
	}
	names, _ := spec["names"].(map[string]any)
	plural, _ := names["plural"].(string)
	scope, _ := spec["scope"].(string)

	versions, _ := spec["versions"].([]any)
	var schema map[string]any
	for _, v := range versions {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		vName, _ := vm["name"].(string)
		if version != "" && vName != version {
			continue
		}
		if sch, ok := vm["schema"].(map[string]any); ok {
			if oas, ok := sch["openAPIV3Schema"].(map[string]any); ok {
				schema = oas
				break
			}
		}
	}
	if schema == nil {
		// CRD v1beta1 legacy
		if v1beta1, ok := spec["validation"].(map[string]any); ok {
			if oas, ok := v1beta1["openAPIV3Schema"].(map[string]any); ok {
				schema = oas
			}
		}
	}
	if schema == nil {
		return nil, plural, scope, fmt.Errorf("no openAPIV3Schema found for version %q", version)
	}

	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return schema, plural, scope, nil
	}
	specProp, _ := props["spec"].(map[string]any)
	if specProp == nil {
		return schema, plural, scope, nil
	}
	return specProp, plural, scope, nil
}

var skipSchemaPaths = map[string]bool{
	"status": true, "metadata": true, "conditions": true,
	"managedFields": true, "finalizers": true, "ownerReferences": true,
}

func openAPISchemaToFormFields(root map[string]any, maxDepth, maxFields int) ([]FormField, bool) {
	var fields []FormField
	truncated := false
	var walk func(node map[string]any, path string, depth int, requiredSet map[string]bool)
	walk = func(node map[string]any, path string, depth int, requiredSet map[string]bool) {
		if len(fields) >= maxFields {
			truncated = true
			return
		}
		if depth > maxDepth {
			return
		}
		props, _ := node["properties"].(map[string]any)
		if props == nil {
			return
		}
		required := requiredKeys(node)
		reqSet := map[string]bool{}
		for _, k := range required {
			reqSet[k] = true
		}
		keys := sortedKeys(props)
		for _, key := range keys {
			if skipSchemaPaths[key] {
				continue
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			prop, _ := props[key].(map[string]any)
			if prop == nil {
				continue
			}
			if field, ok := schemaPropertyToField(key, childPath, prop, reqSet[key]); ok {
				fields = append(fields, field)
				if len(fields) >= maxFields {
					truncated = true
					return
				}
				continue
			}
			if isObjectType(prop) {
				walk(prop, childPath, depth+1, reqSet)
			}
		}
	}
	walk(root, "", 0, map[string]bool{})
	return fields, truncated
}

func schemaPropertyToField(key, path string, prop map[string]any, required bool) (FormField, bool) {
	if isObjectType(prop) {
		return FormField{}, false
	}
	if typ, _ := prop["type"].(string); typ == "array" {
		items, _ := prop["items"].(map[string]any)
		if items != nil && !isObjectType(items) {
			if field, ok := scalarFieldFromSchema(key, path, items, required, prop); ok {
				return field, true
			}
		}
		return FormField{}, false
	}
	return scalarFieldFromSchema(key, path, prop, required, prop)
}

func scalarFieldFromSchema(key, path string, prop map[string]any, required bool, descSource map[string]any) (FormField, bool) {
	typ, _ := prop["type"].(string)
	if typ == "" && prop["x-kubernetes-int-or-string"] != nil {
		typ = "string"
	}
	fieldType := openAPITypeToFieldType(typ)
	if fieldType == "" {
		return FormField{}, false
	}
	label := humanizeKey(key)
	if desc, _ := descSource["description"].(string); desc != "" {
		if len(desc) <= 80 {
			label = desc
		}
	}
	field := FormField{
		Key:      sanitizeFieldKey(path),
		Label:    label,
		Type:     fieldType,
		SpecPath: path,
		Required: required,
	}
	if enum, ok := prop["enum"].([]any); ok && len(enum) > 0 {
		field.Type = "select"
		for _, e := range enum {
			field.Options = append(field.Options, fmt.Sprintf("%v", e))
		}
	}
	if def, ok := prop["default"]; ok {
		field.Default = fmt.Sprintf("%v", def)
	} else if ex, ok := prop["example"]; ok {
		field.Default = fmt.Sprintf("%v", ex)
	}
	return field, true
}

func sanitizeFieldKey(path string) string {
	key := strings.ReplaceAll(path, ".", "_")
	key = strings.ReplaceAll(key, "-", "_")
	return key
}

func openAPITypeToFieldType(typ string) string {
	switch typ {
	case "string":
		return "text"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return ""
	}
}

func isObjectType(prop map[string]any) bool {
	if prop == nil {
		return false
	}
	if t, _ := prop["type"].(string); t == "object" {
		return true
	}
	if props, _ := prop["properties"].(map[string]any); len(props) > 0 {
		return true
	}
	return false
}

func requiredKeys(node map[string]any) []string {
	raw, ok := node["required"].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, r := range raw {
		if s, ok := r.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
