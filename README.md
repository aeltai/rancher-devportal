# Geeko-Ops

**The ops marketplace for Rancher.** Geeko-Ops is a Rancher UI extension that turns cluster operations into a self-service catalog — browse offerings, request environments, get approvals, and ship Fleet GitOps manifests without opening a terminal.

Built for platform teams who want a storefront, not a ticket queue. Geeko optional. Jacket recommended.

![Geeko-Ops sidebar icon](pkg/devportal/assets/geeko-jacket-icon.png)

## Why Geeko-Ops?

| Without Geeko-Ops | With Geeko-Ops |
|-------------------|----------------|
| Slack threads for every namespace | Curated **catalog** (collections → offerings) |
| Tribal knowledge in runbooks | **Wizard** with validation and Git paths |
| Manual YAML and Fleet repos | **Operator** renders manifests and pushes to Git |
| Admins drowning in requests | **Ops queue** with approve / reject / preview |

**For developers:** browse the marketplace, pick an offering (namespace, Helm stack, Harvester VM, custom CRD, or… Geeko Drugs), submit, track status.

**For admins:** manage `platform.yaml` visually, tune approval rules, import the bundled catalog, and keep Fleet in sync.

## Architecture

```
 Rancher Dashboard (browser)
      │
      ▼
 Geeko-Ops UI extension  ──HTTP──▶  devportal-backend (Go)
      │                                    │
      │                                    ├── Rancher API (auth, RBAC)
      │                                    └── PlatformRequest CRs
      ▼
 platform-operator  ──▶  Git (Gitea / GitHub)  ──▶  Fleet  ──▶  clusters
```

| Component | Role |
|-----------|------|
| **UI** (`pkg/devportal/`) | Marketplace, request wizard, ops queue, catalog admin |
| **Backend** (`backend/`) | Catalog API, requests, auth, platform config |
| **Operator** (`operator/`) | Reconcile `PlatformRequest` → Git + Fleet |
| **Config** (`config/platform.yaml`) | Collections, offerings, Git repos, approval rules |

Works alongside [Krew Workstation](https://github.com/aeltai/krew-workstation) in the same Rancher sidebar — Geeko-Ops for provisioning, Krew for kubectl power tools.

---

## Quick install (cluster)

### Prerequisites

- Rancher **2.10+** with UI extensions enabled
- Kubernetes cluster with `kubectl` access
- A Git repo for Fleet (Gitea, GitHub, etc.)
- (Optional) `platform-operator` for automated Git push + reconciliation

### 1. Install CRD

```bash
kubectl apply -f deploy/crd/platformrequest.yaml
```

### 2. Install backend + UI plugin (Helm)

```bash
helm upgrade --install geeko-ops ./helm/devportal \
  --namespace devportal-system \
  --create-namespace \
  --set rancher.url=https://rancher.cattle-system.svc \
  --set rancher.token="token-xxxxx:yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy" \
  --set uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin"
```

Or apply the UIPlugin manifest directly if the backend is already running:

```bash
kubectl apply -f deploy/uiplugin.yaml   # adjust endpoint in file first
```

### 3. Seed platform catalog

```bash
kubectl create namespace devportal-system --dry-run=client -o yaml | kubectl apply -f -

kubectl create configmap platform-config \
  --from-file=platform.yaml=config/platform.yaml \
  -n devportal-system \
  --dry-run=client -o yaml | kubectl apply -f -
```

Restart the backend pod if it was already running so it picks up the ConfigMap.

### 4. Deploy the operator (GitOps reconciliation)

```bash
./scripts/deploy-operator-local.sh   # local / dev
# or apply deploy/operator/ manifests for your environment
```

### 5. Open Rancher

In the left app bar, click **Geeko-Ops** (Geeko in a jacket). Admins see **Ops queue**, **Catalog & config**, and the same marketplace as users.

---

## Local development

Geeko-Ops is **not a separate website** — it loads inside Rancher Dashboard as a UI extension.

From **[krew-workstation](https://github.com/aeltai/krew-workstation)**:

```bash
# Link this repo into the Shell dev UI
./scripts/link-devportal.sh

# Backend + Gitea + operator stack
docker compose -f ../rancher-devportal/docker-compose.local.yml up -d

# Rancher + Shell dev server (both Krew and Geeko-Ops)
cd ../krew-workstation
API=http://localhost:8089 yarn dev
```

Open **https://localhost:8005** → left sidebar **Geeko-Ops**.

| Service | URL |
|---------|-----|
| Shell dev UI | https://localhost:8005 |
| Rancher | https://localhost:8449 |
| Geeko-Ops backend | http://localhost:9010 |
| Gitea (Fleet Git) | http://localhost:3001 |

**Standalone UI dev** (Geeko-Ops only, no Krew):

```bash
yarn install
API=http://localhost:8089 yarn dev --port 8006
```

---

## Configuration

Catalog and approval rules live in **`platform.yaml`** (ConfigMap `platform-config`):

- **Collections** — marketplace categories (Namespaces, VMs, Helm, Custom…)
- **Offerings** — what users can request (`namespace`, `helm`, `crd`, `generic`, …)
- **Git** — Fleet repo URL, branch, credentials secret
- **Approval** — which requests need admin sign-off

Edit via **Geeko-Ops → Catalog & config** in the UI, or edit YAML directly. See [docs/platform-config.md](docs/platform-config.md).

---

## Publish UI extension (GitHub Pages)

The extension bundle is published on push to `main`:

**https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin/**

Manual build:

```bash
yarn install
yarn build-pkg
mkdir -p extensions/devportal/0.1.0/plugin
cp -r dist-pkg/devportal-0.1.0/* extensions/devportal/0.1.0/plugin/
```

Set `uiPlugin.endpoint` in Helm to that URL (directory path, no trailing filename).

---

## API (backend)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/auth/me` | Current Rancher user + capabilities |
| GET | `/api/portal/catalog` | Collections, offerings, charts |
| GET | `/api/portal/requests` | List environment requests |
| POST | `/api/portal/requests` | Submit new request |
| GET/PUT | `/api/portal/platform-config` | Read / write catalog YAML |

---

## Documentation

- [Platform config](docs/platform-config.md) — catalog schema and offerings
- [UI overview](docs/devportal-ui.md) — wizard, admin panels, status tabs
- [Architecture](docs/architecture.md) — backend, operator, Fleet flow
- [Local dev](docs/local-dev.md) — docker-compose and troubleshooting
- [Helm chart](helm/README.md) — production install options

---

## License

See repository license. Geeko is a SUSE mascot — our jacketed chameleon art is project fan art for demo purposes.

## Related

- [Krew Workstation](https://github.com/aeltai/krew-workstation) — kubectl plugin terminal in the same Rancher UI
