#!/usr/bin/env bash
# Build di mesin lokal/WSL → upload artefak production saja → restart di server.
# Server TIDAK perlu Go/npm. Frontend jalan via PM2 (NVM user yang sudah ada).
#
# Setup sekali:
#   cp deploy/deploy.env.example deploy/deploy.env
#   nano deploy/deploy.env
#   ssh-copy-id -p PORT user@host
#
# Deploy:
#   make deploy
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
DEPLOY_STRIP_SOURCEMAPS="${DEPLOY_STRIP_SOURCEMAPS:-true}"
DEPLOY_RSYNC_COMPRESS="${DEPLOY_RSYNC_COMPRESS:-true}"
DEPLOY_FRONTEND="${DEPLOY_FRONTEND:-pm2}"
DEPLOY_PM2_USER="${DEPLOY_PM2_USER:-dianrp}"
DEPLOY_PUBLIC_URL="${DEPLOY_PUBLIC_URL:-https://${DEPLOY_HOST}}"

REMOTE="${DEPLOY_USER}@${DEPLOY_HOST}"
DEPLOY_IS_ROOT=false
[[ "$DEPLOY_USER" == "root" ]] && DEPLOY_IS_ROOT=true

if [[ "$DEPLOY_FRONTEND" == "pm2" ]]; then
	FRONTEND_OWNER="$DEPLOY_PM2_USER"
else
	FRONTEND_OWNER="$DEPLOY_RUN_USER"
fi

SSH_CONTROL_DIR="${XDG_RUNTIME_DIR:-/tmp}/kasq-deploy-$$"
mkdir -p "$SSH_CONTROL_DIR"
chmod 700 "$SSH_CONTROL_DIR"

SSH_OPTS=(
	-p "$DEPLOY_PORT"
	-o StrictHostKeyChecking=accept-new
	-o ControlMaster=auto
	-o "ControlPath=${SSH_CONTROL_DIR}/cm-%r@%h:%p"
	-o ControlPersist=30m
)
if [[ -n "${DEPLOY_SSH_KEY:-}" ]]; then
	SSH_OPTS+=(-i "$DEPLOY_SSH_KEY")
fi

SSH_WRAPPER="$STAGING/ssh-wrapper.sh"
cleanup() {
	ssh "${SSH_OPTS[@]}" -O exit "$REMOTE" 2>/dev/null || true
	rm -rf "$SSH_CONTROL_DIR"
}
trap cleanup EXIT

mkdir -p "$STAGING"
{
	printf '#!/usr/bin/env bash\nexec ssh'
	printf ' -p %q' "$DEPLOY_PORT"
	printf ' -o StrictHostKeyChecking=accept-new'
	printf ' -o ControlMaster=auto'
	printf ' -o ControlPath=%q' "${SSH_CONTROL_DIR}/cm-%r@%h:%p"
	printf ' -o ControlPersist=30m'
	if [[ -n "${DEPLOY_SSH_KEY:-}" ]]; then
		printf ' -i %q' "$DEPLOY_SSH_KEY"
	fi
	printf ' "$@"\n'
} >"$SSH_WRAPPER"
chmod +x "$SSH_WRAPPER"
export RSYNC_RSH="$SSH_WRAPPER"

rsync_opts_for() {
	local owner="$1"
	local -n _out=$2
	_out=(-rlptD --delete)
	if [[ "$DEPLOY_IS_ROOT" == "true" ]]; then
		_out+=(--chown="${owner}:${owner}" --no-owner --no-group)
	else
		_out+=(--chmod=Du=rwx,Dgo=rx,Fu=rwX,Fgo=rX)
	fi
	if [[ "$DEPLOY_RSYNC_COMPRESS" == "true" ]]; then
		_out+=(-z --compress-level=1)
	fi
}

echo "==> KasQ deploy → ${REMOTE}:${DEPLOY_PATH}"
echo "    Backend: systemd (${DEPLOY_RUN_USER}) | Frontend: ${DEPLOY_FRONTEND} (${FRONTEND_OWNER})"

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
sed "s|__KASQ_ORIGIN__|${DEPLOY_PUBLIC_URL}|g" \
	"$ROOT/deploy/aapanel/ecosystem.config.cjs" >"$STAGING/frontend/ecosystem.config.cjs"
echo "    PM2 ORIGIN=${DEPLOY_PUBLIC_URL}"

if [[ "$DEPLOY_STRIP_SOURCEMAPS" == "true" ]]; then
	map_count="$(find "$STAGING/frontend/build" -name '*.map' | wc -l | tr -d ' ')"
	if [[ "$map_count" != "0" ]]; then
		find "$STAGING/frontend/build" -name '*.map' -delete
		echo "    Hapus ${map_count} file *.map (tidak dipakai di production)"
	fi
fi

echo "==> [3/4] Upload artefak (rsync)..."
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions' '${DEPLOY_PATH}/frontend/build'"

rsync_opts_for "$DEPLOY_RUN_USER" BACKEND_RSYNC
rsync "${BACKEND_RSYNC[@]}" \
	"$STAGING/backend/kasq-server" \
	"${REMOTE}:${DEPLOY_PATH}/backend/kasq-server"

rsync_opts_for "$FRONTEND_OWNER" FRONTEND_RSYNC
rsync "${FRONTEND_RSYNC[@]}" \
	"$STAGING/frontend/build/" \
	"${REMOTE}:${DEPLOY_PATH}/frontend/build/"

# ecosystem.config.cjs — tanpa --delete
ECOSYSTEM_RSYNC=(-rlptD)
if [[ "$DEPLOY_IS_ROOT" == "true" ]]; then
	ECOSYSTEM_RSYNC+=(--chown="${FRONTEND_OWNER}:${FRONTEND_OWNER}" --no-owner --no-group)
fi
if [[ "$DEPLOY_RSYNC_COMPRESS" == "true" ]]; then
	ECOSYSTEM_RSYNC+=(-z --compress-level=1)
fi
rsync "${ECOSYSTEM_RSYNC[@]}" \
	"$STAGING/frontend/ecosystem.config.cjs" \
	"${REMOTE}:${DEPLOY_PATH}/frontend/ecosystem.config.cjs"

echo "==> [4/4] Set permission & restart..."
# shellcheck disable=SC2016
WAIT_BACKEND='
wait_backend() {
	local attempt
	for attempt in $(seq 1 15); do
		if systemctl is-active --quiet kasq-backend; then
			systemctl is-active kasq-backend
			return 0
		fi
		sleep 2
	done
	echo ""
	echo "WARN: kasq-backend belum active setelah ~30 detik (upload sudah selesai)."
	systemctl status kasq-backend --no-pager || true
	echo "Cek log: journalctl -u kasq-backend -n 30 --no-pager"
	return 0
}
'

# shellcheck disable=SC2016
PM2_RESTART='
restart_pm2_frontend() {
	local pm2_user="$1"
	local app_path="$2"
	local run_pm2
	run_pm2() {
		if [[ "$(id -un)" == "$pm2_user" ]]; then
			bash -lc "$1"
		else
			sudo -u "$pm2_user" bash -lc "$1"
		fi
	}
	if ! run_pm2 "command -v pm2 >/dev/null"; then
		echo "ERROR: pm2 tidak ditemukan untuk user ${pm2_user}."
		echo "       Setup sekali: su - ${pm2_user} && cd ${app_path}/frontend && pm2 start ecosystem.config.cjs && pm2 save"
		return 1
	fi
	run_pm2 "cd \"${app_path}/frontend\" && pm2 startOrReload ecosystem.config.cjs --only kasq-frontend --update-env && pm2 save"
	run_pm2 "pm2 status kasq-frontend" || true
}
'

if [[ "$DEPLOY_FRONTEND" == "pm2" ]]; then
	RESTART_FRONTEND="restart_pm2_frontend '${DEPLOY_PM2_USER}' '${DEPLOY_PATH}'"
	WAIT_FRONTEND=""
else
	RESTART_FRONTEND='systemctl restart kasq-frontend'
	WAIT_FRONTEND='
	for attempt in $(seq 1 15); do
		if systemctl is-active --quiet kasq-frontend; then
			systemctl is-active kasq-frontend
			break
		fi
		sleep 2
	done
	'
fi

if [[ "$DEPLOY_IS_ROOT" == "true" ]]; then
	REMOTE_SCRIPT=$(cat <<EOF
set -e
${WAIT_BACKEND}
${PM2_RESTART}
RUN_USER='${DEPLOY_RUN_USER}'
FE_USER='${FRONTEND_OWNER}'
chmod +x '${DEPLOY_PATH}/backend/kasq-server'
mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions'
chown "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/kasq-server'
chown -R "\${FE_USER}:\${FE_USER}" '${DEPLOY_PATH}/frontend/build' '${DEPLOY_PATH}/frontend/ecosystem.config.cjs'
chown -R "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/data/wa-sessions'
if [[ -f '${DEPLOY_PATH}/backend/.env' ]]; then
	chown "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/.env'
	chmod 640 '${DEPLOY_PATH}/backend/.env'
fi
if [[ "${DEPLOY_RESTART}" == "true" ]]; then
	systemctl restart kasq-backend
	${RESTART_FRONTEND} || true
	wait_backend
	${WAIT_FRONTEND}
fi
EOF
)
else
	REMOTE_SCRIPT=$(cat <<EOF
set -e
${WAIT_BACKEND}
${PM2_RESTART}
chmod 755 '${DEPLOY_PATH}/backend/kasq-server'
chmod -R a+rX '${DEPLOY_PATH}/frontend/build'
mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions'
chmod 777 '${DEPLOY_PATH}/backend/data/wa-sessions'
if [[ "${DEPLOY_RESTART}" == "true" ]]; then
	if sudo -n systemctl restart kasq-backend 2>/dev/null; then
		${RESTART_FRONTEND} || true
		wait_backend
		${WAIT_FRONTEND}
	else
		echo ""
		echo "WARN: restart backend gagal — sudo butuh password interaktif."
		echo "      File sudah ter-upload."
		echo ""
	fi
fi
EOF
)
fi
ssh "${SSH_OPTS[@]}" "$REMOTE" "bash -s" <<< "$REMOTE_SCRIPT"

echo ""
echo "==> Deploy selesai."
echo "    Backend:  ${DEPLOY_PATH}/backend/kasq-server (systemd)"
if [[ "$DEPLOY_FRONTEND" == "pm2" ]]; then
	echo "    Frontend: ${DEPLOY_PATH}/frontend/build/ (PM2: kasq-frontend)"
	echo "    Log FE:   su - ${DEPLOY_PM2_USER} -c 'pm2 logs kasq-frontend'"
else
	echo "    Frontend: ${DEPLOY_PATH}/frontend/build/ (systemd)"
fi
echo ""
echo "Pastikan backend/.env sudah ada di server (tidak ikut di-upload)."
