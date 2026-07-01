package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var fleetGitRepoGVR = schema.GroupVersionResource{
	Group:    "fleet.cattle.io",
	Version:  "v1alpha1",
	Resource: "gitrepos",
}

func (r *reconciler) ensureFleetGitRepo(ctx context.Context, fleetNs, name string, obj map[string]any) error {
	u := &unstructured.Unstructured{Object: obj}
	u.SetNamespace(fleetNs)
	u.SetName(name)

	existing, err := r.dynamic.Resource(fleetGitRepoGVR).Namespace(fleetNs).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		u.SetResourceVersion(existing.GetResourceVersion())
		_, err = r.dynamic.Resource(fleetGitRepoGVR).Namespace(fleetNs).Update(ctx, u, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update GitRepo: %w", err)
		}
		return nil
	}

	_, err = r.dynamic.Resource(fleetGitRepoGVR).Namespace(fleetNs).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create GitRepo: %w", err)
	}
	return nil
}

func (r *reconciler) fleetGitRepoPhase(ctx context.Context, fleetNs, name string) string {
	obj, err := r.dynamic.Resource(fleetGitRepoGVR).Namespace(fleetNs).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "displayState")
	return phase
}
