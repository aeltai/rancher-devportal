#!/usr/bin/env bash
# Build platform-operator, bootstrap Gitea, and deploy into the krew-workstation Rancher cluster.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RANCHER_CONTAINER="${RANCHER_CONTAINER:-krew-workstation-rancher-1}"
GITEA_CONTAINER="${GITEA_CONTAINER:-krew-workstation-gitea-1}"

if ! docker ps --format '{{.Names}}' | grep -qx "$RANCHER_CONTAINER"; then
  echo "Rancher container $RANCHER_CONTAINER is not running. Start krew-workstation first." >&2
  exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -qx "$GITEA_CONTAINER"; then
  echo "Gitea container $GITEA_CONTAINER is not running." >&2
  echo "Start it with: cd krew-workstation && docker compose up -d gitea" >&2
  exit 1
fi

echo "Setting up Gitea (org, repo, token, in-cluster Service)..."
"$ROOT/scripts/setup-gitea-local.sh"

echo "Building platform-operator image..."
docker build -t platform-operator:local "$ROOT/operator"

echo "Importing image into Rancher k3s..."
docker save platform-operator:local | docker exec -i "$RANCHER_CONTAINER" ctr images import - >/dev/null

echo "Applying CRD and operator manifests..."
docker exec "$RANCHER_CONTAINER" kubectl create namespace devportal-system --dry-run=client -o yaml \
  | docker exec -i "$RANCHER_CONTAINER" kubectl apply -f -

docker exec -i "$RANCHER_CONTAINER" kubectl apply -f - < "$ROOT/deploy/crd/platformrequest.yaml"

docker exec "$RANCHER_CONTAINER" kubectl create configmap platform-config \
  --from-file=platform.yaml="$ROOT/config/platform.yaml" \
  -n devportal-system --dry-run=client -o yaml \
  | docker exec -i "$RANCHER_CONTAINER" kubectl apply -f -

docker exec -i "$RANCHER_CONTAINER" kubectl apply -f - < "$ROOT/deploy/operator/deployment.yaml"

echo "Done."
echo "  Operator: docker exec $RANCHER_CONTAINER kubectl -n devportal-system get pods -l app=platform-operator"
echo "  Gitea UI: http://localhost:3001/platform/fleet"
