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

# 2. Start Gitea (local Fleet Git repo) + Geeko-Ops controller
export RANCHER_TOKEN=$(grep RANCHER_TOKEN krew-workstation/.env | cut -d= -f2-)
cd krew-workstation && docker compose up -d gitea
cd ../rancher-devportal
docker compose -f docker-compose.local.yml up -d --build

# 4. Link devportal into the Rancher UI dev project
cd krew-workstation && ./scripts/link-devportal.sh

# 5. Start the UI (Krew + Developer Portal both appear)
API=http://localhost:8089 yarn dev
```

Open **https://localhost:8005** → Platform → Geeko-Ops.

See [ui-tour.md](ui-tour.md) for a screenshot walkthrough.

## Controller environment variables

Set in `docker-compose.local.yml`:

| Variable | Default | Purpose |
|----------|---------|---------|
| `RANCHER_URL` | `https://rancher:443` | Rancher API endpoint (Docker service name) |
| `RANCHER_TOKEN` | — | Admin API token for kubeconfig generation |
| `PLATFORM_NAMESPACE` | `devportal-system` | Namespace for PlatformRequest CRs |
| `PLATFORM_GIT_REPO` | `http://gitea.devportal-system.svc:3000/platform/fleet.git` | Default Git repo in wizard (in-cluster Gitea Service) |
| `PLATFORM_GIT_BRANCH` | `main` | Default branch |
| `PLATFORM_GIT_SECRET` | `platform-git-credentials` | Default Secret name for Git PAT |
| `PLATFORM_FLEET_NAMESPACE` | `fleet-default` | Namespace for Fleet GitRepo CRs |
| `ALLOW_SERVICE_TOKEN` | `false` | Set `true` to allow the controller's service token fallback (dev only) |

## Deploying the platform operator

```bash
./scripts/deploy-operator-local.sh
docker exec krew-workstation-rancher-1 kubectl -n devportal-system get pods -l app=platform-operator
```

See [architecture.md](architecture.md) for reconcile flow and Git manifest layout.

## Rebuilding the controller

After any Go changes:

```bash
cd rancher-devportal
docker compose -f docker-compose.local.yml up -d --build
```

## Applying the CRD manually

The controller auto-applies the embedded CRD when the caller is allowed. Non-admin users require the CRD to be pre-installed. Manual apply:

```bash
kubectl apply -f deploy/crd/platformrequest.yaml
```

## UI hot-reload

The `krew-workstation` `vue.config.js` enables `poll: 500` watching on symlinked directories, so edits to `pkg/devportal/DevPortalPage.vue` (which is symlinked from `../rancher-devportal/pkg/devportal`) are picked up automatically without restarting the dev server.

## Architecture

See [architecture.md](architecture.md) for the full request flow and CRD design.
