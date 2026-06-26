# Local development

## Prerequisites

- Docker + Docker Compose
- Node 20 + Yarn 1.x
- A running `krew-workstation` stack (provides Rancher + network)
- `RANCHER_TOKEN` in `krew-workstation/.env`

## Quick start

```bash
# 1. Start Rancher (inside krew-workstation)
cd krew-workstation && docker compose up -d

# 2. Start devportal backend on the same Docker network
export RANCHER_TOKEN=$(grep RANCHER_TOKEN krew-workstation/.env | cut -d= -f2-)
cd rancher-devportal
docker compose -f docker-compose.local.yml up -d --build

# 3. Link devportal into the Rancher UI dev project
cd krew-workstation && ./scripts/link-devportal.sh

# 4. Start the UI (Krew + Developer Portal both appear)
API=http://localhost:8089 yarn dev
```

Open **https://localhost:8005** → Platform → Developer Portal.

## Backend environment variables

Set in `docker-compose.local.yml`:

| Variable | Default | Purpose |
|----------|---------|---------|
| `RANCHER_URL` | `https://rancher:443` | Rancher API endpoint (Docker service name) |
| `RANCHER_TOKEN` | — | Admin API token for kubeconfig generation |
| `PLATFORM_NAMESPACE` | `devportal-system` | Namespace for PlatformRequest CRs |
| `PLATFORM_GIT_REPO` | `https://github.com/aeltai/rancher-devportal` | Platform GitOps repo (future PR target) |
| `PLATFORM_GIT_BRANCH` | `main` | Branch for Fleet manifests |
| `PLATFORM_FLEET_NAMESPACE` | `fleet-default` | Namespace for Fleet GitRepo CRs |
| `ALLOW_SERVICE_TOKEN` | `false` | Set `true` to allow backend's own token (dev only) |

## Rebuilding the backend

After any Go changes:

```bash
cd rancher-devportal
docker compose -f docker-compose.local.yml up -d --build
```

## Applying the CRD manually

The backend auto-applies the embedded CRD on startup. To apply manually:

```bash
kubectl apply -f deploy/crd/platformrequest.yaml
```

## UI hot-reload

The `krew-workstation` `vue.config.js` enables `poll: 500` watching on symlinked directories, so edits to `pkg/devportal/DevPortalPage.vue` (which is symlinked from `../rancher-devportal/pkg/devportal`) are picked up automatically without restarting the dev server.

## Architecture

See [architecture.md](architecture.md) for the full request flow and CRD design.
