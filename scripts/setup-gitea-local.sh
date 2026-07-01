#!/usr/bin/env bash
# Bootstrap local Gitea for Developer Portal Fleet manifests.
set -euo pipefail

GITEA_CONTAINER="${GITEA_CONTAINER:-krew-workstation-gitea-1}"
RANCHER_CONTAINER="${RANCHER_CONTAINER:-krew-workstation-rancher-1}"
GITEA_USER="${GITEA_USER:-platform}"
GITEA_PASS="${GITEA_PASS:-platform}"
GITEA_REPO="${GITEA_REPO:-fleet}"
GITEA_HOST_PORT="${GITEA_HOST_PORT:-3001}"
GITEA_INTERNAL_URL="http://gitea.devportal-system.svc:3000"
GITEA_REPO_URL="${GITEA_INTERNAL_URL}/${GITEA_USER}/${GITEA_REPO}.git"

wait_gitea() {
  echo "Waiting for Gitea on :${GITEA_HOST_PORT}..."
  for _ in $(seq 1 60); do
    if curl -sf "http://localhost:${GITEA_HOST_PORT}/api/v1/version" >/dev/null 2>&1; then
      echo "Gitea is up."
      return 0
    fi
    sleep 2
  done
  echo "Gitea did not become ready in time." >&2
  exit 1
}

ensure_admin() {
  if curl -sf -u "${GITEA_USER}:${GITEA_PASS}" "http://localhost:${GITEA_HOST_PORT}/api/v1/user" >/dev/null 2>&1; then
    echo "Gitea admin user ${GITEA_USER} already exists."
    return 0
  fi
  echo "Creating Gitea admin user ${GITEA_USER}..."
  docker exec "$GITEA_CONTAINER" gitea admin user create \
    --username "$GITEA_USER" \
    --password "$GITEA_PASS" \
    --email "${GITEA_USER}@local.dev" \
    --admin \
    --must-change-password=false
}

ensure_repo() {
  local auth="-u ${GITEA_USER}:${GITEA_PASS}"
  local base="http://localhost:${GITEA_HOST_PORT}/api/v1"

  if curl -sf $auth "${base}/repos/${GITEA_USER}/${GITEA_REPO}" >/dev/null 2>&1; then
    echo "Repo ${GITEA_USER}/${GITEA_REPO} already exists."
    return 0
  fi

  echo "Creating repo ${GITEA_USER}/${GITEA_REPO}..."
  curl -sf $auth -X POST "${base}/user/repos" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"${GITEA_REPO}\",\"auto_init\":true,\"default_branch\":\"main\",\"private\":false}" >/dev/null
}

create_token() {
  local auth="-u ${GITEA_USER}:${GITEA_PASS}"
  local base="http://localhost:${GITEA_HOST_PORT}/api/v1"
  local token
  token=$(curl -sf $auth -X POST "${base}/users/${GITEA_USER}/tokens" \
    -H 'Content-Type: application/json' \
    -d '{"name":"platform-operator","scopes":["write:repository","read:repository"]}' \
    | python3 -c 'import json,sys; print(json.load(sys.stdin)["sha1"])')
  if [[ -z "$token" ]]; then
    echo "Failed to create Gitea token." >&2
    exit 1
  fi
  echo "$token"
}

register_k8s_service() {
  local gitea_ip
  gitea_ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$GITEA_CONTAINER")
  if [[ -z "$gitea_ip" ]]; then
    echo "Could not resolve Gitea container IP." >&2
    exit 1
  fi
  echo "Registering in-cluster Gitea service at ${gitea_ip}:3000 ..."
  docker exec "$RANCHER_CONTAINER" kubectl create namespace devportal-system --dry-run=client -o yaml \
    | docker exec -i "$RANCHER_CONTAINER" kubectl apply -f -
  docker exec -i "$RANCHER_CONTAINER" kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: gitea
  namespace: devportal-system
spec:
  ports:
    - port: 3000
      targetPort: 3000
      protocol: TCP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: gitea
  namespace: devportal-system
subsets:
  - addresses:
      - ip: ${gitea_ip}
    ports:
      - port: 3000
EOF
}

create_git_secret() {
  local token="$1"
  echo "Creating platform-git-credentials secret..."
  docker exec -i "$RANCHER_CONTAINER" kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: platform-git-credentials
  namespace: devportal-system
type: Opaque
stringData:
  username: ${GITEA_USER}
  token: ${token}
EOF
}

main() {
  wait_gitea
  ensure_admin
  ensure_repo
  token=$(create_token)
  register_k8s_service
  create_git_secret "$token"

  cat <<EOF

Gitea ready.
  UI:        http://localhost:${GITEA_HOST_PORT}/${GITEA_USER}/${GITEA_REPO}
  Login:     ${GITEA_USER} / ${GITEA_PASS}
  Git (CR):  ${GITEA_REPO_URL}
  Branch:    main

Use ${GITEA_REPO_URL} as gitRepo in PlatformRequest (wizard pre-fills this via PLATFORM_GIT_REPO).
EOF
}

main "$@"
