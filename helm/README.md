# Geeko-Ops Helm charts

| Chart | Path | Installs |
|-------|------|----------|
| **geeko-ops** | `helm/geeko-ops/` | Umbrella — backend + operator + config + CRD |
| **devportal** | `helm/devportal/` | Backend API, UIPlugin, platform-config |
| **operator** | `helm/operator/` | platform-operator Deployment + RBAC |

## Quick start

```bash
cd helm/geeko-ops
helm dependency update
helm upgrade --install geeko-ops . -n devportal-system --create-namespace \
  --set devportal.rancher.token="token-xxx:yyyy"
```

## Release artifacts

Charts and container images are published on git tag `v*` via `.github/workflows/release.yml`.

See [geeko-ops/README.md](geeko-ops/README.md) for full values reference.
