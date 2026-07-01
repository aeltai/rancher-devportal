# Geeko-Ops Helm Chart

Installs **devportal-backend** and registers the **Geeko-Ops** UI extension in Rancher.

See the main [README](../README.md) for full installation steps, local dev, and configuration.

## Install

```bash
helm upgrade --install geeko-ops ./helm/devportal \
  --namespace devportal-system \
  --create-namespace \
  --set rancher.url=https://rancher.cattle-system.svc \
  --set rancher.token="token-xxx:yyyy" \
  --set uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin"
```

Also apply:

```bash
kubectl apply -f deploy/crd/platformrequest.yaml
kubectl create configmap platform-config \
  --from-file=platform.yaml=../config/platform.yaml \
  -n devportal-system --dry-run=client -o yaml | kubectl apply -f -
```

## Values

| Key | Description |
|-----|-------------|
| `uiPlugin.enabled` | Register UIPlugin CR |
| `uiPlugin.endpoint` | GitHub Pages or self-hosted extension URL |
| `uiPlugin.metadata.catalog.cattle.io/display-name` | Sidebar label (**Geeko-Ops**) |
| `rancher.url` / `rancher.token` | Backend → Rancher API |
