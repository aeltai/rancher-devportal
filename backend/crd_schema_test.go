package main

import "testing"

func TestOpenAPISchemaToFormFields(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"replicas": map[string]any{
				"type":        "integer",
				"description": "Replica count",
				"default":     float64(1),
			},
			"mode": map[string]any{
				"type": "string",
				"enum": []any{"fast", "slow"},
			},
			"enabled": map[string]any{
				"type": "boolean",
			},
			"template": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cpu": map[string]any{
						"type": "integer",
					},
				},
			},
			"status": map[string]any{
				"type": "object",
			},
		},
		"required": []any{"replicas"},
	}

	fields, truncated := openAPISchemaToFormFields(schema, 4, 20)
	if truncated {
		t.Fatalf("unexpected truncation")
	}
	if len(fields) < 4 {
		t.Fatalf("expected at least 4 fields, got %d", len(fields))
	}
	foundReplicas := false
	for _, f := range fields {
		if f.Key == "replicas" && f.Type == "number" && f.Required {
			foundReplicas = true
		}
		if f.Key == "template_cpu" && f.SpecPath == "template.cpu" {
			// nested
		}
	}
	if !foundReplicas {
		t.Fatalf("missing replicas field: %+v", fields)
	}
}

func TestSanitizeFieldKey(t *testing.T) {
	if got := sanitizeFieldKey("template.spec.domain.cpu"); got != "template_spec_domain_cpu" {
		t.Fatalf("got %q", got)
	}
}
