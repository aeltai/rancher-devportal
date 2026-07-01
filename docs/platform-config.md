# Platform configuration (`platform.yaml`)

The Developer Portal and `platform-operator` read **`platform.yaml`** from ConfigMap `platform-config` in `devportal-system`.

## Admin UI

**Platform → Developer Portal → DevPortal Admin Settings**

- **Visual editor** — manage Collections, Offerings, Git connections, and approval/CRD settings
- **YAML tab** — full `platform.yaml` for GitOps-style editing
- Changes are saved to ConfigMap `platform-config` via the backend API

## Catalog model

```yaml
collections:
  - id: namespaces
    label: Namespaces & Projects
    description: ...
    icon: namespace
    weight: 10

offerings:
  - id: sandbox-ns
    collectionId: namespaces
    label: Sandbox namespace
    kind: namespace          # namespace | cluster | helm | crd | generic
    template: sandbox        # legacy template id for operator
    gitOps: false
    requiresApproval: false
```

### Offering kinds

| Kind | Purpose |
|------|---------|
| `namespace` | Creates `env-{name}` namespace; optional `cloneFrom: true` for reference picker |
| `cluster` | Virtual/dedicated cluster profile (GitOps manifests) |
| `helm` | Fleet Helm chart delivery (`charts: [id, …]`) |
| `crd` | Custom resource with admin-defined `formSchema` → rendered `specYaml` |
| `generic` | Freeform `manifestTemplate` (Go templates) + optional `formSchema` |

### Form schema (field builder)

```yaml
formSchema:
  - key: cpu
    label: vCPUs
    type: number
    specPath: spec.template.spec.domain.cpu.cores
    default: "2"
    required: true
```

Supported field types: `text`, `number`, `boolean`, `select` (with `options`).

## Git connections

Any HTTP(S) Git host works (Gitea, GitHub, GitLab, Bitbucket):

```yaml
git:
  mode: single
  defaultRepo: http://gitea.devportal-system.svc:3000/platform/fleet.git
  repos:
    - id: platform-fleet
      name: Platform Fleet
      url: http://gitea.devportal-system.svc:3000/platform/fleet.git
      branch: main
      secretName: platform-git-credentials
      authType: token
```

Secret keys: `username`, `token` (or `password`).

Use **Test connection** in the admin UI or `POST /api/portal/git/test-connection`.

## CRD discovery

```yaml
crdDiscovery:
  enabled: true
  clusters: local          # or downstream cluster id (e.g. harvester)
  excludeGroups: [...]
```

The `clusters` field selects which kubeconfig context CRDs are listed from.

## Legacy fields

`templates[]` and `charts[]` are still supported. If `collections`/`offerings` are absent, the backend auto-migrates them at runtime.

## Apply manually

```bash
kubectl apply -f deploy/platform-config.yaml
./scripts/deploy-operator-local.sh
```
