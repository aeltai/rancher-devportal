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

### Full stack (recommended)

```bash
cd helm/geeko-ops && helm dependency update

helm upgrade --install geeko-ops . \
  --namespace devportal-system \
  --create-namespace \
  --set devportal.rancher.token="token-xxxxx:yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy" \
  --set devportal.rancher.publicHost="rancher.example.com" \
  --set devportal.uiPlugin.endpoint="https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin" \
  --set devportal.image.tag=v0.1.0 \
  --set operator.image.tag=v0.1.0
```

Installs **backend**, **UIPlugin**, **platform-config** ConfigMap, **PlatformRequest CRD**, and **platform-operator**.

Git credentials for Fleet (create before requesting GitOps offerings):

```bash
kubectl create secret generic platform-git-credentials \
  -n devportal-system \
  --from-literal=username=git-user \
  --from-literal=password=your-token
```

See [helm/geeko-ops/README.md](helm/geeko-ops/README.md) for subchart-only installs and values.

### From GitHub Release

```bash
# After tagging v0.1.0 — download geeko-ops-0.1.0.tgz from Releases
helm upgrade --install geeko-ops geeko-ops-0.1.0.tgz \
  --namespace devportal-system --create-namespace \
  --set devportal.rancher.token="..." \
  --set devportal.image.tag=v0.1.0 \
  --set operator.image.tag=v0.1.0
```

---

## Releases & container images

Push a semver tag to build and publish:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

GitHub Actions publishes:

| Artifact | Location |
|----------|----------|
| Backend image | `ghcr.io/aeltai/rancher-devportal-backend:v0.1.0` |
| Operator image | `ghcr.io/aeltai/rancher-devportal-operator:v0.1.0` |
| Helm charts | GitHub Release attachments (`geeko-ops`, `devportal`, `operator`) |

Or run **Actions → Release → Run workflow** manually.

After first publish, make GHCR packages public (Settings → Packages) or set `imagePullSecrets` on the charts.

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
