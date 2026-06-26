# Publishing the extension

## GitHub Pages (recommended)

The workflow `.github/workflows/build-extension-pages.yml`:

1. Runs `yarn build-pkg`
2. Copies artifacts to `_site/extensions/devportal/0.1.0/plugin/`
3. Deploys to GitHub Pages

After the first successful run, enable **Settings → Pages → Source: GitHub Actions**.

### UIPlugin endpoint

Use the Pages base URL for the plugin directory:

```
https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin
```

Helm:

```yaml
uiPlugin:
  endpoint: "https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin"
```

### Verify

```bash
curl -sI "https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin/devportal-0.1.0.umd.min.js" | head -1
```

## Raw GitHub (alternative)

```
https://raw.githubusercontent.com/aeltai/rancher-devportal/main/extensions/devportal/0.1.0/plugin
```

Commit built files under `extensions/` on `main` if you prefer raw hosting over Pages.

## Version bumps

1. Bump `version` in `pkg/devportal/package.json` and root `package.json`
2. Update `helm/devportal/values.yaml` `uiPlugin.version` and path segment `0.1.0`
3. Rebuild and redeploy Pages
4. Upgrade Helm release or patch UIPlugin CR

## Catalog registration (optional)

Set `bootstrap.enabled=true` and `catalog.gitRepo` in Helm to register this repo in Rancher's extension catalog UI.
