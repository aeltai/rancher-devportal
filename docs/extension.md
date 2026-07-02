# Developer Portal — extension docs

## Layout

```
pkg/devportal/
├── index.ts          # Plugin entry (addProduct, addRoutes)
├── product.ts        # Sidebar product "Platform → Developer Portal"
├── package.json      # Extension metadata (name must match UIPlugin.pluginName)
├── DevPortalPage.vue # Main UI
└── routing/
    └── extension-routing.ts
```

## Product name

The extension `package.json` **`name`** field must be `devportal`. This must match:

- Helm `uiPlugin.pluginName`
- UIPlugin CR `spec.plugin.name`
- Built bundle filename `devportal-0.1.0.umd.min.js`

## Local development

1. Start controller: `docker compose -f docker-compose.local.yml up -d devportal-backend`
2. Apply CRD: `kubectl apply -f deploy/crd/platformrequest.yaml`
3. Run UI: `API=http://localhost:9010 yarn dev`
4. Open Rancher shell dev URL (default `https://localhost:8005`)

The UI mints a Rancher session token via `ext.cattle.io/Token` and sends it as `Authorization: Bearer` to the controller.

## Building the package

```bash
yarn build-pkg
```

Output: `dist-pkg/devportal-0.1.0/` — copy to `extensions/devportal/0.1.0/plugin/` for GitHub Pages or raw Git hosting.

## Vue 3 / Shell patches

`scripts/patch-shell-vue3-refs.js` runs on `postinstall` to fix `$refs` array issues in `@rancher/shell` 3.0.x with Vue 3.2.

## Controller coupling

This extension expects the Geeko-Ops controller at `API` env (dev) or the cluster Service URL (production). It does **not** embed provisioning logic — all cluster operations go through the controller API.

See [architecture.md](architecture.md) for the full flow.
