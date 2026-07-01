package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	crGroup    = "platform.devportal.io"
	crVersion  = "v1alpha1"
	crPlural   = "platformrequests"
	crResource = "platformrequests"
)

var platformRequestGVR = schema.GroupVersionResource{
	Group:    crGroup,
	Version:  crVersion,
	Resource: crResource,
}

type reconciler struct {
	cfg     config
	dynamic dynamic.Interface
	core    kubernetes.Interface
}

func newReconciler(cfg config) (*reconciler, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig: %w", err)
		}
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	core, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return &reconciler{cfg: cfg, dynamic: dyn, core: core}, nil
}

func (r *reconciler) reconcileAll(ctx context.Context) error {
	list, err := r.dynamic.Resource(platformRequestGVR).Namespace(r.cfg.WatchNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list PlatformRequests: %w", err)
	}
	for _, item := range list.Items {
		if err := r.reconcileOne(ctx, &item); err != nil {
			logReconcileError(item.GetName(), err)
		}
	}
	return nil
}

func logReconcileError(name string, err error) {
	fmt.Printf("reconcile %s: %v\n", name, err)
}

func (r *reconciler) reconcileOne(ctx context.Context, pr *unstructured.Unstructured) error {
	spec := pr.Object["spec"]
	specMap, _ := spec.(map[string]any)
	if specMap == nil {
		return fmt.Errorf("missing spec")
	}

	envName := strField(specMap, "name")
	if envName == "" {
		return fmt.Errorf("spec.name is empty")
	}
	template := strField(specMap, "template")
	if template == "" {
		template = "sandbox"
	}
	charts := strSliceField(specMap, "charts")
	requester := strField(specMap, "requester")
	displayName := strField(specMap, "displayName")
	description := strField(specMap, "description")

	gitRepo := strField(specMap, "gitRepo")
	gitBranch := strField(specMap, "gitBranch")
	if gitBranch == "" {
		gitBranch = r.cfg.DefaultGitBranch
	}
	gitPath := strField(specMap, "gitPath")
	if gitPath == "" {
		gitPath = defaultGitPath(envName)
	}
	gitSecret := strField(specMap, "gitSecretName")
	if gitSecret == "" {
		gitSecret = r.cfg.DefaultGitSecret
	}
	targetClusters := strSliceField(specMap, "targetClusters")

	envNs := "env-" + envName
	fleetGitRepoName := "fleet-" + envName

	r.setPhase(ctx, pr, "Reconciling", "Operator processing request")

	// 1. Ensure namespace exists in cluster
	if err := r.ensureNamespace(ctx, envNs, envName, template, requester, displayName, charts); err != nil {
		r.setPhase(ctx, pr, "Failed", err.Error())
		return err
	}
	r.patchStatusFields(ctx, pr, map[string]any{"namespaceName": envNs})

	needsGitOps := template == "team" || template == "vcluster" || len(charts) > 0
	if !needsGitOps {
		r.setPhase(ctx, pr, "Ready", fmt.Sprintf("Namespace %s ready (sandbox)", envNs))
		return nil
	}

	if gitRepo == "" {
		msg := "gitRepo is required for team/vcluster templates or when charts are selected"
		r.setPhase(ctx, pr, "Failed", msg)
		return fmt.Errorf("%s", msg)
	}

	// 2. Render and push manifests to Git
	renderIn := renderInput{
		EnvName:        envName,
		DisplayName:    displayName,
		Description:    description,
		Template:       template,
		Charts:         charts,
		Requester:      requester,
		GitPath:        gitPath,
		TargetClusters: targetClusters,
	}
	files := renderManifests(renderIn)

	creds, err := readGitCreds(ctx, r, pr.GetNamespace(), gitSecret)
	if err != nil {
		r.setPhase(ctx, pr, "Failed", "Git credentials: "+err.Error())
		return err
	}

	r.setPhase(ctx, pr, "Pushing", fmt.Sprintf("Pushing manifests to %s (%s)", gitRepo, gitPath))
	commit, err := pushManifests(gitRepo, gitBranch, gitPath, files, creds)
	if err != nil {
		r.setPhase(ctx, pr, "Failed", "Git push: "+err.Error())
		return err
	}
	r.patchStatusFields(ctx, pr, map[string]any{"gitCommit": commit})

	// 3. Create/update Fleet GitRepo
	gitRepoObj := renderFleetGitRepo(fleetGitRepoName, gitRepo, gitBranch, gitPath, targetClusters)
	if err := r.ensureFleetGitRepo(ctx, r.cfg.FleetNamespace, fleetGitRepoName, gitRepoObj); err != nil {
		r.setPhase(ctx, pr, "Failed", "Fleet GitRepo: "+err.Error())
		return err
	}
	r.patchStatusFields(ctx, pr, map[string]any{"fleetGitRepoName": fleetGitRepoName})

	fleetPhase := r.fleetGitRepoPhase(ctx, r.cfg.FleetNamespace, fleetGitRepoName)
	msg := fmt.Sprintf("Namespace %s; Git commit %s; Fleet GitRepo %s", envNs, shortCommit(commit), fleetGitRepoName)
	if fleetPhase != "" {
		msg += "; Fleet: " + fleetPhase
	}
	r.setPhase(ctx, pr, "Ready", msg)
	return nil
}

func (r *reconciler) ensureNamespace(ctx context.Context, name, envName, template, requester, displayName string, charts []string) error {
	_, err := r.core.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"devportal.io/env":       envName,
				"devportal.io/template":  template,
				"devportal.io/requester": sanitizeLabel(requester),
			},
			Annotations: map[string]string{
				"devportal.io/display-name": displayName,
			},
		},
	}
	for _, chart := range charts {
		if chart != "" {
			ns.Annotations["devportal.io/chart-"+chart] = "requested"
		}
	}
	_, err = r.core.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create namespace: %w", err)
	}
	return nil
}

func (r *reconciler) setPhase(ctx context.Context, pr *unstructured.Unstructured, phase, message string) {
	_ = r.patchStatusFields(ctx, pr, map[string]any{
		"phase":   phase,
		"message": message,
	})
}

func (r *reconciler) patchStatusFields(ctx context.Context, pr *unstructured.Unstructured, fields map[string]any) error {
	status, _, _ := unstructured.NestedMap(pr.Object, "status")
	if status == nil {
		status = map[string]any{}
	}
	for k, v := range fields {
		status[k] = v
	}
	patch := map[string]any{"status": status}
	b, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = r.dynamic.Resource(platformRequestGVR).Namespace(pr.GetNamespace()).Patch(
		ctx,
		pr.GetName(),
		"merge",
		b,
		metav1.PatchOptions{},
		"status",
	)
	return err
}

func strField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func strSliceField(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}
