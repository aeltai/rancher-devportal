# platform-operator Helm Chart

Deploys the Geeko-Ops **platform-operator** — watches `PlatformRequest` CRs and pushes Fleet Git manifests.

## Requirements

- `PlatformRequest` CRD installed (via `devportal` chart or `deploy/crd/platformrequest.yaml`)
- ConfigMap `platform-config` in the same namespace (catalog YAML)

## Install

```bash
helm upgrade --install geeko-ops-operator . \
  --namespace devportal-system \
  --create-namespace \
  --set image.tag=v0.1.0
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `image.repository` | `ghcr.io/aeltai/rancher-devportal-operator` | Operator image |
| `image.tag` | chart appVersion | Image tag |
| `platformConfig.configMapName` | `platform-config` | Catalog ConfigMap |
| `env.reconcileInterval` | `15s` | Reconcile loop interval |
