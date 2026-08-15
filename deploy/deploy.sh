#!/usr/bin/env bash
# Build di mesin lokal/WSL → upload artefak production saja → restart systemd di server.
# Server TIDAK perlu Go/npm. Tetap butuh Node.js di server (jalankan frontend/build).
#
# Setup sekali:
#   cp deploy/deploy.env.example deploy/deploy.env
#   nano deploy/deploy.env
#
# Deploy:
#   ./deploy/deploy.sh
#   # atau: make deploy
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT/deploy/deploy.env"
STAGING="$ROOT/deploy/.staging"

if [[ ! -f "$ENV_FILE" ]]; then
	echo "ERROR: $ENV_FILE belum ada."
	echo "Copy dulu: cp deploy/deploy.env.example deploy/deploy.env"
	exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

: "${DEPLOY_HOST:?Set DEPLOY_HOST di deploy/deploy.env}"
: "${DEPLOY_USER:?Set DEPLOY_USER di deploy/deploy.env}"
DEPLOY_PATH="${DEPLOY_PATH:?Set DEPLOY_PATH di deploy/deploy.env}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_RUN_USER="${DEPLOY_RUN_USER:-www}"
DEPLOY_RESTART="${DEPLOY_RESTART:-true}"
DEPLOY_GOOS="${DEPLOY_GOOS:-linux}"
DEPLOY_GOARCH="${DEPLOY_GOARCH:-amd64}"

SSH_OPTS=(-p "$DEPLOY_PORT" -o StrictHostKeyChecking=accept-new)
if [[ -n "${DEPLOY_SSH_KEY:-}" ]]; then
	SSH_OPTS+=(-i "$DEPLOY_SSH_KEY")
fi
RSYNC_SSH="ssh ${SSH_OPTS[*]}"
REMOTE="${DEPLOY_USER}@${DEPLOY_HOST}"

echo "==> KasQ deploy → ${REMOTE}:${DEPLOY_PATH}"

echo "==> [1/4] Build backend (${DEPLOY_GOOS}/${DEPLOY_GOARCH})..."
mkdir -p "$STAGING/backend"
(
	cd "$ROOT/backend"
	GOOS="$DEPLOY_GOOS" GOARCH="$DEPLOY_GOARCH" CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o "$STAGING/backend/kasq-server" ./cmd/server
)

echo "==> [2/4] Build frontend..."
if [[ -f "$ROOT/frontend/.env.production" ]]; then
	echo "    Pakai frontend/.env.production"
else
	echo "    (tips) Buat frontend/.env.production — PUBLIC_API_URL kosong untuk same-origin"
fi
(
	cd "$ROOT/frontend"
	if [[ -f package-lock.json ]]; then npm ci; else npm install; fi
	npm run build
)
rm -rf "$STAGING/frontend"
mkdir -p "$STAGING/frontend"
cp -a "$ROOT/frontend/build" "$STAGING/frontend/"

echo "==> [3/4] Upload artefak (rsync)..."
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions' '${DEPLOY_PATH}/frontend/build'"

rsync -avz --delete -e "$RSYNC_SSH" \
	"$STAGING/backend/kasq-server" \
	"${REMOTE}:${DEPLOY_PATH}/backend/kasq-server"

rsync -avz --delete -e "$RSYNC_SSH" \
	"$STAGING/frontend/build/" \
	"${REMOTE}:${DEPLOY_PATH}/frontend/build/"

echo "==> [4/4] Set permission & restart..."
REMOTE_SCRIPT=$(cat <<EOF
set -e
chmod +x '${DEPLOY_PATH}/backend/kasq-server'
chown -R ${DEPLOY_RUN_USER}:${DEPLOY_RUN_USER} '${DEPLOY_PATH}/backend/kasq-server' '${DEPLOY_PATH}/frontend/build'
mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions'
chown -R ${DEPLOY_RUN_USER}:${DEPLOY_RUN_USER} '${DEPLOY_PATH}/backend/data'
if [[ "${DEPLOY_RESTART}" == "true" ]]; then
	systemctl restart kasq-backend kasq-frontend
	systemctl is-active kasq-backend kasq-frontend
fi
EOF
)
ssh "${SSH_OPTS[@]}" "$REMOTE" "bash -s" <<< "$REMOTE_SCRIPT"

echo ""
echo "==> Deploy selesai."
echo "    Backend:  ${DEPLOY_PATH}/backend/kasq-server"
echo "    Frontend: ${DEPLOY_PATH}/frontend/build/"
echo ""
echo "Pastikan backend/.env sudah ada di server (tidak ikut di-upload)."
