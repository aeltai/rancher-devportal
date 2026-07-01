# Geeko-Ops Helm Chart (umbrella)

Installs the full Geeko-Ops stack:

| Subchart | Component |
|----------|-----------|
| **devportal** | Backend API, UIPlugin, platform-config ConfigMap, CRD |
| **operator** | platform-operator (Fleet Git reconciliation) |

## Prerequisites

- Rancher 2.10+ with UI extensions
- Helm 3.10+
- Rancher API token with cluster-admin (for backend)

## Install from repo

```bash
cd helm/geeko-ops
helm dependency update
helm upgrade --install geeko-ops . \
  --namespace devportal-system \
  --create-namespace \
  --set devportal.rancher.token="token-xxxxx:yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy" \
  --set devportal.rancher.publicHost="rancher.example.com" \
  --set devportal.uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin"
```

## Install from GitHub Release

Download `geeko-ops-0.1.0.tgz` from [Releases](https://github.com/aeltai/rancher-devportal/releases), then:

```bash
helm upgrade --install geeko-ops geeko-ops-0.1.0.tgz \
  --namespace devportal-system --create-namespace \
  --set devportal.rancher.token="..." \
  --set devportal.image.tag=v0.1.0 \
  --set operator.image.tag=v0.1.0
```

## Git credentials (Fleet push)

Create the secret before or during install:

```bash
kubectl create secret generic platform-git-credentials \
  --namespace devportal-system \
  --from-literal=username=git-user \
  --from-literal=password=token-or-password
```

Or enable chart-managed secret:

```yaml
devportal:
  git:
    credentials:
      create: true
      token: "your-gitea-token"
```

## Images

Published to GHCR on tag `v*`:

- `ghcr.io/aeltai/rancher-devportal-backend:v0.1.0`
- `ghcr.io/aeltai/rancher-devportal-operator:v0.1.0`

## Subcharts only

```bash
# Backend + UI only
helm upgrade --install geeko-ops-backend ../devportal -n devportal-system --create-namespace

# Operator only (requires platform-config ConfigMap)
helm upgrade --install geeko-ops-operator ../operator -n devportal-system --create-namespace
```
