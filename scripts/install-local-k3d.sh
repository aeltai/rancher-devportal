#!/usr/bin/env bash
# Install Geeko-Ops v0.1.0 into krew-workstation Rancher (embedded k3s).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RANCHER_CONTAINER="${RANCHER_CONTAINER:-krew-workstation-rancher-1}"
GITEA_CONTAINER="${GITEA_CONTAINER:-krew-workstation-gitea-1}"
RELEASE="${RELEASE:-geeko-ops}"
NAMESPACE="${NAMESPACE:-devportal-system}"
CHART="${CHART:-$ROOT/helm/geeko-ops}"
VALUES="${VALUES:-$ROOT/deploy/local-k3d-values.yaml}"
VERSION="${VERSION:-v0.1.0}"
CONTROLLER_PORT="${CONTROLLER_PORT:-9010}"

if ! docker ps --format '{{.Names}}' | grep -qx "$RANCHER_CONTAINER"; then
  echo "Start krew-workstation first: cd krew-workstation && docker compose up -d" >&2
  exit 1
fi

RANCHER_TOKEN="${RANCHER_TOKEN:-}"
if [ -z "$RANCHER_TOKEN" ] && docker ps --format '{{.Names}}' | grep -q 'devportal-backend'; then
  RANCHER_TOKEN="$(docker exec rancher-devportal-devportal-backend-1 printenv RANCHER_TOKEN 2>/dev/null || true)"
fi
if [ -z "$RANCHER_TOKEN" ] && [ -f "${KREW_ROOT:-$ROOT/../krew-workstation}/.env" ]; then
  RANCHER_TOKEN="$(grep -E '^RANCHER_TOKEN=' "${KREW_ROOT:-$ROOT/../krew-workstation}/.env" | cut -d= -f2- || true)"
fi
if [ -z "$RANCHER_TOKEN" ]; then
  echo "Set RANCHER_TOKEN (admin API token) and re-run." >&2
  exit 1
fi

if docker ps --format '{{.Names}}' | grep -qx "$GITEA_CONTAINER"; then
  if docker exec "$RANCHER_CONTAINER" kubectl get secret platform-git-credentials -n "$NAMESPACE" >/dev/null 2>&1; then
    echo "Git secret platform-git-credentials already exists — skipping Gitea bootstrap."
  else
    echo "Ensuring Gitea + in-cluster Service for Fleet Git..."
    "$ROOT/scripts/setup-gitea-local.sh"
  fi
fi

echo "Importing Geeko-Ops ${VERSION} images (linux/amd64) into Rancher k3s..."
for img in \
  "ghcr.io/aeltai/rancher-devportal-backend:${VERSION}" \
  "ghcr.io/aeltai/rancher-devportal-operator:${VERSION}"; do
  docker pull --platform linux/amd64 "$img"
  docker save "$img" | docker exec -i "$RANCHER_CONTAINER" ctr images import - >/dev/null
  echo "  imported $img"
done

echo "Updating Helm chart dependencies..."
(cd "$CHART" && helm dependency update >/dev/null)

KCFG="$(mktemp)"
docker exec "$RANCHER_CONTAINER" cat /etc/rancher/k3s/k3s.yaml >"$KCFG"

echo "Installing ${RELEASE} into ${NAMESPACE}..."
docker run --rm \
  --network "container:${RANCHER_CONTAINER}" \
  -v "$KCFG:/kube/config:ro" \
  -v "$CHART:/chart:ro" \
  -v "$VALUES:/values.yaml:ro" \
  alpine/helm:3.14.4 upgrade --install "$RELEASE" /chart \
  --kubeconfig /kube/config \
  --namespace "$NAMESPACE" \
  --create-namespace \
  -f /values.yaml \
  --set "devportal.rancher.token=${RANCHER_TOKEN}" \
  --wait --timeout 5m

rm -f "$KCFG"

echo "Stopping docker-compose controller (in-cluster controller is canonical)..."
if docker ps --format '{{.Names}}' | grep -q 'rancher-devportal-devportal-backend-1'; then
  (cd "$ROOT" && docker compose -f docker-compose.local.yml stop devportal-backend 2>/dev/null) || true
fi

echo "Starting port-forward ${CONTROLLER_PORT} -> controller Service..."
pkill -f "kubectl port-forward.*${NAMESPACE}.*geeko-ops-controller.*${CONTROLLER_PORT}" 2>/dev/null || true
docker exec "$RANCHER_CONTAINER" kubectl port-forward -n "$NAMESPACE" "svc/geeko-ops-controller" "${CONTROLLER_PORT}:3000" \
  --address=0.0.0.0 >/tmp/geeko-ops-pf.log 2>&1 &
sleep 2

echo "Applying UIPlugin (Geeko-Ops) if missing..."
docker exec -i "$RANCHER_CONTAINER" kubectl apply -f - <<EOF
apiVersion: catalog.cattle.io/v1
kind: UIPlugin
metadata:
  name: geeko-ops
  namespace: cattle-ui-plugin-system
spec:
  plugin:
    name: devportal
    version: "0.1.0"
    endpoint: "https://aeltai.github.io/rancher-devportal/extensions/devportal/0.1.0/plugin"
    noCache: true
    metadata:
      catalog.cattle.io/display-name: Geeko-Ops
      catalog.cattle.io/kube-version: ">= 1.16.0-0"
      catalog.cattle.io/rancher-version: ">= 2.10.0-0"
      catalog.cattle.io/ui-extensions-version: ">= 3.0.0 < 4.0.0"
EOF

echo ""
echo "Geeko-Ops installed."
echo "  Rancher UI:  https://localhost:8449"
echo "  Controller:  http://localhost:${CONTROLLER_PORT} (port-forward inside Rancher container)"
echo "  Pods:        docker exec $RANCHER_CONTAINER kubectl -n $NAMESPACE get pods"
echo ""
echo "Enable the extension: Rancher → ☰ → Extensions → Geeko-Ops → Enable"
echo "Then open Platform → Geeko-Ops in the sidebar."
