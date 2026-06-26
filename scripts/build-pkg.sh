#!/usr/bin/env bash
set -euo pipefail

PKG="${1:-devportal}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION=$(node -p "require('./pkg/${PKG}/package.json').version")
NAME="${PKG}-${VERSION}"
OUT="${ROOT}/dist-pkg/${NAME}"
SHELL_DIR="${ROOT}/node_modules/@rancher/shell"

echo "Building UI Package ${PKG} → ${OUT}"
rm -rf "${OUT}"
mkdir -p "${OUT}"

ln -sfn "${SHELL_DIR}" "${ROOT}/pkg/${PKG}/.shell"

ENTRY="index.js"
if [ -f "${ROOT}/pkg/${PKG}/index.ts" ]; then
  ENTRY="index.ts"
fi

"${ROOT}/node_modules/.bin/vue-cli-service" build \
  --name "${NAME}" \
  --target lib "pkg/${PKG}/${ENTRY}" \
  --dest "${OUT}" \
  --formats umd-min \
  --filename "${NAME}"

cp "${ROOT}/pkg/${PKG}/package.json" "${OUT}/package.json"
node "${SHELL_DIR}/scripts/pkgfile.js" "${OUT}/package.json"
rm -f "${ROOT}/pkg/${PKG}/.shell"

echo "Built ${OUT}/${NAME}.umd.min.js"
