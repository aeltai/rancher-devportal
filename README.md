# Rancher Developer Portal

A **standalone** Rancher UI extension for developer self-service: request environments (namespaces, Fleet GitOps, Helm charts/operators) from the Dashboard.

This project is separate from [Krew Workstation](https://github.com/aeltai/krew-workstation).

## Architecture

```
 Rancher Dashboard (browser)
      |
      v
 DevPortalPage.vue  ──HTTP──>  devportal-backend (Go)  ──> Rancher API (auth)
                                                        ──> kubectl (PlatformRequest CR, namespaces)
```

| Component | Role |
|-----------|------|
| **UI extension** (`pkg/devportal/`) | Wizard UI, request list, Rancher session tokens |
| **Backend** (`backend/`) | Catalog API, PlatformRequest CR lifecycle |
| **CRD** (`deploy/crd/`) | `PlatformRequest` custom resource |
| **Helm** (`helm/devportal/`) | Backend Deployment + UIPlugin |

## Local development (inside Krew Workstation UI)

Developer Portal is **not a separate website**. It appears in the **same Rancher Dashboard** as Krew Workstation.

From **krew-workstation**:

```bash
./scripts/link-devportal.sh
docker compose -f ../rancher-devportal/docker-compose.local.yml up -d
cd ../krew-workstation && API=http://localhost:8089 yarn dev
```

Open **https://localhost:8005** → sidebar shows **Tools → Krew Workstation** and **Platform → Developer Portal**.

Do not run `yarn dev` in this repo on a separate port unless you are developing the portal UI in isolation.

## Standalone dev (portal UI only)

```bash
yarn install
API=http://localhost:8089 yarn dev --port 8006
```

Use only when working on DevPortalPage.vue without Krew linked.

## Install on cluster

```bash
helm install devportal ./helm/devportal \
  --set rancher.url=https://rancher.cattle-system.svc \
  --set rancher.token="token-xxx" \
  --set uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin" \
  -n devportal-system --create-namespace

kubectl apply -f deploy/crd/platformrequest.yaml
```

## Extension bundle (GitHub Pages)

Published automatically on push to `main`:

**https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin/**

Set `uiPlugin.endpoint` in Helm to that URL (no trailing filename).

Manual build:

```bash
yarn build-pkg
cp -r dist-pkg/devportal-0.1.0/* extensions/devportal/0.1.0/plugin/
```

## Documentation

- [Extension development](docs/extension.md)
- [Publishing](docs/publishing.md)
- [Architecture](docs/architecture.md)
- [Helm chart](helm/README.md)

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/auth/me` | Current Rancher user |
| GET | `/api/portal/catalog` | Chart + template catalog |
| GET | `/api/portal/stack` | Recommended stack info |
| GET | `/api/portal/requests` | List PlatformRequests |
| POST | `/api/portal/requests` | Create environment request |

## Related

- [Krew Workstation](https://github.com/aeltai/krew-workstation) — kubectl plugin terminal workstation
