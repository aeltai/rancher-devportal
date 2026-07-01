# Developer Portal UI

## Overview

The Developer Portal is a Rancher UI extension under **Platform → Developer Portal**.  
It uses `@rancher/shell` **Wizard** for requests and a structured **DevPortal Admin Settings** panel for catalog management.

## Request wizard (6 steps)

Click **Request environment** to open the catalog-driven wizard:

| Step | Purpose |
|------|---------|
| **1. Basics** | Environment name (slug → `env-{name}`) and description |
| **2. Collection** | Choose catalog category (Namespaces, Clusters, Platform Services, VMs, Custom) |
| **3. Offering** | Pick a specific offering; clone-pattern offerings include cluster + namespace picker |
| **4. Configure** | Helm chart pickers, or dynamic form fields from admin `formSchema` |
| **5. Delivery** | Git repo URL, branch, path, target clusters (shown when GitOps required) |
| **6. Review** | Full summary including form values and Git target |

Components: `RequestWizard.vue`, `OfferingFormField.vue`.

## DevPortal Admin Settings

Admins open **DevPortal Admin Settings** to manage the platform catalog:

| Tab | Purpose |
|-----|---------|
| **Collections** | Group offerings (namespaces, clusters, services, VMs, custom) |
| **Offerings** | Per-kind catalog entries with field builder for CRD/generic |
| **Git connections** | Multi-repo config with **Test connection** |
| **Approval & CRD** | Approval toggles and CRD discovery cluster selector |

Toggle **Visual editor** / **YAML** for round-trip editing. Component: `CatalogAdminSettings.vue`, `FieldBuilder.vue`.

## Request table

Shows your environments (admins see all). Expand a row for Git preview, Fleet status, and YAML tabs.

## Admin approval

Offerings with `requiresApproval: true` (or global approval rules for charts/CRs) enter **Pending approval** until an admin approves.

## Local development

```bash
# From krew-workstation (Krew + DevPortal):
cd krew-workstation && API=http://localhost:8089 yarn dev   # https://localhost:8005

# DevPortal only:
cd rancher-devportal && yarn dev   # https://localhost:8006
```

Backend proxy: `/devportal-api` → `http://localhost:9010`
