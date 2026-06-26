# Developer Portal UI

## Overview

The Developer Portal is a Rancher UI extension under **Platform → Developer Portal**.  
It renders at full width (layout `plain`) — no sidebar padding — to maximise usable space.

## Wizard — requesting an environment

Click **Request environment** to open the 4-step wizard.

### Step 1 — Name

- **Environment name**: lowercase, numbers, hyphens (slugified automatically). Becomes the namespace `env-<name>` and the CR name `pr-<name>`.
- **Description**: optional free-text stored in `spec.description`.

### Step 2 — Template

Choose one of three environment profiles:

| Template | When to use |
|----------|-------------|
| **Sandbox** | Dev experiments, personal namespaces, no Fleet GitRepo |
| **Team environment** | Shared team namespace + Fleet GitRepo for GitOps chart delivery |
| **Virtual cluster** | Full isolated control plane (requires vCluster operator) |

### Step 3 — Charts

Optional Helm charts delivered via Fleet bundles. Available catalog:

| Chart | Category |
|-------|----------|
| rancher-monitoring | observability |
| rancher-logging | observability |
| rancher-backup | backup |
| fleet | gitops |
| cert-manager | security |
| ingress-nginx | networking |

### Step 4 — Review

Summary of name, template, and selected charts before submit.

## "What gets generated" preview

Below the request table, a live preview updates as you fill in the wizard:

- **PlatformRequest CR** — the YAML submitted to Kubernetes
- **Namespace** — `env-<name>` with template/env labels
- **Fleet GitRepo** — (team/vcluster only) Fleet CR pointing at the platform Git repo
- **Git repository layout** — folder tree under `environments/<name>/`

The preview uses the currently entered name, selected template and charts, so it reflects exactly what will be created.

## Request table

The request list shows all your environments (admins see all users' requests).

**Columns:** Name · Requester (admin) · Phase · Namespace · Template · Charts · Created · View manifest

Click a row or **View manifest** to expand:

- **PlatformRequest YAML** — the full CR as stored in Kubernetes
- **Fleet & cluster resources** — table of planned Namespace / GitRepo / Bundle resources with Git path and live Fleet phase
- **GitOps repo hint** — where future PR automation will commit manifests

## Admin view

Admins (username `admin` or `globalRoleBinding` admin) additionally see:

- All users' requests
- **Requester** column
- CR name in Name cell
- Git repo/branch banner above the table

## Phases

| Phase | Meaning |
|-------|---------|
| `Pending` | CR created, provisioning not yet started |
| `Provisioning` | Namespace being created |
| `Ready` | Namespace exists, annotations applied |
| `Failed` | Error during provisioning — see expanded Status message |
