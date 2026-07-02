# Geeko-Ops UI tour

Interactive slideshow: **[tour.html](../tour.html)** (GitHub Pages) · local: `docs/pages/tour.html`

Screenshots are captured from the real Rancher Shell dev UI (`https://localhost:8005`) — not generated mockups.

## Refresh screenshots

```bash
cd rancher-devportal
export RANCHER_PASSWORD='…'   # admin password
export TEST_PASSWORD='…'      # non-admin user (e.g. test)

node scripts/capture-ui-screenshots.mjs
node scripts/capture-admin-screenshots.mjs
```

Output: `docs/pages/screenshots/*.png`

## Components at a glance

| Screen | Vue component(s) | Role |
|--------|------------------|------|
| User marketplace | `DevPortalUserView.vue` | Catalog browse + **Request environment** |
| Request wizard | `RequestWizard.vue`, `OfferingFormField.vue` | 6-step `@rancher/shell` Wizard |
| My environments | `RequestStatusTabs.vue`, env cards | Per-user request list + status filters |
| Admin ops queue | `DevPortalAdminView.vue` | All requests, approve / reject, detail drawer |
| Catalog & config | `CatalogAdminSettings.vue`, `FieldBuilder.vue` | Collections, offerings, Git, YAML editor |

## Stack (controller, not “backend”)

```text
Rancher UI extension (pkg/devportal)
        │  Bearer token + REST
        ▼
Geeko-Ops controller (Go API, :9010)
        │  kubectl apply (per-user kubeconfig)
        ▼
PlatformRequest CR  ──►  platform-operator  ──►  Git + Fleet
```

| Piece | Image / deploy | Responsibility |
|-------|----------------|----------------|
| **UI extension** | GitHub Pages bundle + UIPlugin | Wizard, marketplace, admin tabs |
| **Controller** | `ghcr.io/aeltai/rancher-devportal-backend` | Auth, catalog API, create/list CRs |
| **Operator** | `ghcr.io/aeltai/rancher-devportal-operator` | Reconcile approved requests to Git |

Helm umbrella chart: `helm/geeko-ops` (controller + operator subcharts).

## Screenshots

### 1. User marketplace

![User marketplace — ops catalog and welcome hero](../screenshots/01-user-marketplace.png)

### 2. Request wizard

![Request wizard — basics step](../screenshots/02-request-wizard.png)

![Request wizard — configure step](../screenshots/03-wizard-configure.png)

### 3. My environments

![My environments with status tabs](../screenshots/04-my-environments.png)

### 4. Admin views

![Admin ops queue](../screenshots/05-admin-ops-queue.png)

![Admin request env tab](../screenshots/06-admin-request-env.png)

![Catalog and config editor](../screenshots/07-catalog-config.png)

![Expanded request detail](../screenshots/08-request-detail.png)

## Non-admin access

Users need **Cluster Member** on the target cluster (for kubeconfig) plus the requester RBAC in `deploy/rbac-requester.example.yaml`. The controller skips CRD install when the user lacks CRD permissions — the CRD must already be on the cluster.
