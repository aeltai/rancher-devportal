# Developer Portal Helm Chart

Deploy **devportal-backend** and the **UIPlugin** CR to the Rancher management cluster.

## Install

```bash
kubectl apply -f ../../deploy/crd/platformrequest.yaml

helm install devportal . \
  --set rancher.url=https://rancher.cattle-system.svc \
  --set rancher.token="token-xxx" \
  --set uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin" \
  -n devportal-system --create-namespace
```

## Key values

| Parameter | Description |
|-----------|-------------|
| `uiPlugin.endpoint` | GitHub Pages URL to extension bundle |
| `uiPlugin.pluginName` | Must be `devportal` |
| `rancher.url` | In-cluster Rancher API URL |
| `rancher.token` | Service account token (optional if using existingSecret) |
| `persistence.enabled` | PVC for backend data (optional) |

## RBAC

The backend needs a ServiceAccount with permissions to create namespaces and manage `platformrequests.platform.devportal.io` in `devportal-system`. Extend `templates/serviceaccount.yaml` with a ClusterRoleBinding for production use.
