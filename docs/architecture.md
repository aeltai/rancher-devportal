# Architecture

## Request flow

```mermaid
sequenceDiagram
  participant User
  participant UI as DevPortalPage
  participant Rancher
  participant API as devportal-backend
  participant K8s as Management cluster

  User->>UI: Request environment
  UI->>Rancher: POST ext.cattle.io/Token
  Rancher-->>UI: bearerToken
  UI->>API: POST /api/portal/requests (Bearer)
  API->>Rancher: GET /v3/users?me=true
  API->>K8s: apply PlatformRequest CR
  API->>K8s: create namespace env-{name}
  API->>K8s: patch status Ready
  UI->>API: GET /api/portal/requests
  API-->>UI: request list
```

## PlatformRequest CRD

Each self-service request becomes a namespaced `PlatformRequest` in `devportal-system` (configurable via `PLATFORM_NAMESPACE`).

| Field | Purpose |
|-------|---------|
| `spec.template` | `sandbox`, `team`, or `vcluster` guardrails |
| `spec.charts` | Selected catalog chart IDs (Fleet delivery planned) |
| `spec.requester` | Rancher username |
| `status.phase` | `Pending` → `Provisioning` → `Ready` / `Failed` |

## Roadmap

| Layer | Current | Planned |
|-------|---------|---------|
| Namespace | Created synchronously | Quotas, NetworkPolicy templates |
| Fleet | Annotations only | GitRepo + Bundle per request |
| Virtual cluster | Template flag | vCluster / SUSE Virtual Clusters operator |
| RBAC | Per-user request filter | Rancher Project/Namespace RBAC binding |

## Separation from Krew Workstation

| | Krew Workstation | Developer Portal |
|--|------------------|------------------|
| Repo | `krew-workstation` | `rancher-devportal` |
| Product | Tools → Krew | Platform → Developer Portal |
| Backend | Terminal, krew, backups | PlatformRequest provisioning |
| Port (dev) | 9000 | 9010 |

Both extensions can be installed on the same Rancher instance independently.
