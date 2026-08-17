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
#
# GitHub Actions: push ke main → .github/workflows/deploy.yml
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
DEPLOY_SKIP_NPM_CI="${DEPLOY_SKIP_NPM_CI:-auto}"
DEPLOY_PARALLEL_BUILD="${DEPLOY_PARALLEL_BUILD:-true}"
DEPLOY_ONLY="${DEPLOY_ONLY:-all}"
DEPLOY_FRONTEND="${DEPLOY_FRONTEND:-pm2}"
DEPLOY_PM2_USER="${DEPLOY_PM2_USER:-dianrp}"
DEPLOY_PM2_APP_NAME="${DEPLOY_PM2_APP_NAME:-kasq-fe}"
DEPLOY_PM2_BIN="${DEPLOY_PM2_BIN:-}"
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
echo "    Mode: ${DEPLOY_ONLY} | npm ci: ${DEPLOY_SKIP_NPM_CI} | parallel build: ${DEPLOY_PARALLEL_BUILD}"
echo "    (deploy tidak menjalankan svelte-check/TS — hanya vite build + go build)"

NPM_CACHE_DIR="$ROOT/deploy/.cache"
NPM_LOCK_HASH_FILE="$NPM_CACHE_DIR/npm-lock.sha256"
mkdir -p "$NPM_CACHE_DIR"

should_run_npm_ci() {
	local lock_file="$ROOT/frontend/package-lock.json"
	if [[ ! -f "$lock_file" ]]; then
		return 0
	fi
	if [[ "$DEPLOY_SKIP_NPM_CI" == "true" ]]; then
		return 1
	fi
	if [[ "$DEPLOY_SKIP_NPM_CI" == "false" ]]; then
		return 0
	fi
	# auto: skip jika lockfile tidak berubah dan node_modules ada
	if [[ ! -d "$ROOT/frontend/node_modules" ]]; then
		return 0
	fi
	local current_hash
	current_hash="$(sha256sum "$lock_file" | awk '{print $1}')"
	if [[ -f "$NPM_LOCK_HASH_FILE" ]] && [[ "$(cat "$NPM_LOCK_HASH_FILE")" == "$current_hash" ]]; then
		return 1
	fi
	return 0
}

build_backend() {
	echo "==> Build backend (${DEPLOY_GOOS}/${DEPLOY_GOARCH})..."
	mkdir -p "$STAGING/backend"
	(
		cd "$ROOT/backend"
		GOOS="$DEPLOY_GOOS" GOARCH="$DEPLOY_GOARCH" CGO_ENABLED=0 \
			go build -ldflags="-s -w" -o "$STAGING/backend/kasq-server" ./cmd/server
	)
}

build_frontend() {
	echo "==> Build frontend..."
	if [[ -f "$ROOT/frontend/.env.production" ]]; then
		echo "    Pakai frontend/.env.production"
	else
		echo "    (tips) Buat frontend/.env.production — PUBLIC_API_URL kosong untuk same-origin"
	fi
	(
		cd "$ROOT/frontend"
		if should_run_npm_ci; then
			echo "    npm ci (package-lock berubah atau node_modules belum ada)..."
			if [[ -f package-lock.json ]]; then npm ci; else npm install; fi
			if [[ -f package-lock.json ]]; then
				sha256sum package-lock.json | awk '{print $1}' >"$NPM_LOCK_HASH_FILE"
			fi
		else
			echo "    npm ci dilewati — lockfile sama, pakai node_modules lokal"
		fi
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
}

case "$DEPLOY_ONLY" in
	all)
		if [[ "$DEPLOY_PARALLEL_BUILD" == "true" ]]; then
			echo "==> [1-2/4] Build backend + frontend (parallel)..."
			build_backend &
			BB=$!
			build_frontend &
			BF=$!
			wait "$BB" "$BF"
		else
			echo "==> [1/4] Build backend..."
			build_backend
			echo "==> [2/4] Build frontend..."
			build_frontend
		fi
		;;
	backend)
		echo "==> [1/2] Build backend saja (DEPLOY_ONLY=backend)..."
		build_backend
		;;
	frontend)
		echo "==> [1/2] Build frontend saja (DEPLOY_ONLY=frontend)..."
		build_frontend
		;;
	*)
		echo "ERROR: DEPLOY_ONLY tidak valid: $DEPLOY_ONLY (gunakan all|backend|frontend)"
		exit 1
		;;
esac

echo "==> Upload artefak (rsync)..."
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions' '${DEPLOY_PATH}/frontend/build'"

if [[ "$DEPLOY_ONLY" == "all" || "$DEPLOY_ONLY" == "backend" ]]; then
	rsync_opts_for "$DEPLOY_RUN_USER" BACKEND_RSYNC
	rsync "${BACKEND_RSYNC[@]}" \
		"$STAGING/backend/kasq-server" \
		"${REMOTE}:${DEPLOY_PATH}/backend/kasq-server"
fi

if [[ "$DEPLOY_ONLY" == "all" || "$DEPLOY_ONLY" == "frontend" ]]; then
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
fi

echo "==> Set permission & restart..."
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

PM2_RESTART=$(cat <<EOF
restart_pm2_frontend() {
	local pm2_user="\$1"
	local app_path="\$2"
	local pm2_app='${DEPLOY_PM2_APP_NAME}'
	local pm2_bin='${DEPLOY_PM2_BIN}'
	local pm2_home="/home/\${pm2_user}"

	resolve_pm2_bin() {
		if [[ -n "\$pm2_bin" && -x "\$pm2_bin" ]]; then
			return 0
		fi
		pm2_bin=\$(su - "\${pm2_user}" -c 'command -v pm2' 2>/dev/null || true)
		if [[ -n "\$pm2_bin" && -x "\$pm2_bin" ]]; then
			return 0
		fi
		local candidate
		shopt -s nullglob
		for candidate in "\${pm2_home}/.nvm/versions/node/"*/bin/pm2; do
			if [[ -x "\$candidate" ]]; then
				pm2_bin="\$candidate"
				return 0
			fi
		done
		return 1
	}

	if ! resolve_pm2_bin; then
		echo "ERROR: pm2 tidak ditemukan untuk user \${pm2_user}."
		echo "       Set DEPLOY_PM2_BIN di deploy.env, mis.:"
		echo "       DEPLOY_PM2_BIN=/home/dianrp/.nvm/versions/node/v24.19.0/bin/pm2"
		return 1
	fi

	echo "    PM2: \${pm2_bin}"
	local node_bin
	node_bin="\$(dirname "\$pm2_bin")"
	echo "    Node: \${node_bin}"
	su - "\${pm2_user}" -c "export PATH=\"\${node_bin}:\\\$PATH\"; cd '\${app_path}/frontend' && '\${pm2_bin}' startOrReload ecosystem.config.cjs --only \${pm2_app} --update-env && '\${pm2_bin}' save"
	su - "\${pm2_user}" -c "export PATH=\"\${node_bin}:\\\$PATH\"; '\${pm2_bin}' status \${pm2_app}" || true
}
EOF
)

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
DEPLOY_ONLY='${DEPLOY_ONLY}'
if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "backend" ]]; then
	chmod +x '${DEPLOY_PATH}/backend/kasq-server'
	mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions'
	chown "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/kasq-server'
	chown -R "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/data/wa-sessions'
	if [[ -f '${DEPLOY_PATH}/backend/.env' ]]; then
		chown "\${RUN_USER}:\${RUN_USER}" '${DEPLOY_PATH}/backend/.env'
		chmod 640 '${DEPLOY_PATH}/backend/.env'
	fi
fi
if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "frontend" ]]; then
	chown -R "\${FE_USER}:\${FE_USER}" '${DEPLOY_PATH}/frontend/build' '${DEPLOY_PATH}/frontend/ecosystem.config.cjs'
fi
if [[ "${DEPLOY_RESTART}" == "true" ]]; then
	if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "backend" ]]; then
		systemctl restart kasq-backend
		wait_backend
	fi
	if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "frontend" ]]; then
		${RESTART_FRONTEND} || true
		${WAIT_FRONTEND}
	fi
fi
EOF
)
else
	REMOTE_SCRIPT=$(cat <<EOF
set -e
${WAIT_BACKEND}
${PM2_RESTART}
DEPLOY_ONLY='${DEPLOY_ONLY}'
if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "backend" ]]; then
	chmod 755 '${DEPLOY_PATH}/backend/kasq-server'
	mkdir -p '${DEPLOY_PATH}/backend/data/wa-sessions'
	chmod 777 '${DEPLOY_PATH}/backend/data/wa-sessions'
fi
if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "frontend" ]]; then
	chmod -R a+rX '${DEPLOY_PATH}/frontend/build'
fi
if [[ "${DEPLOY_RESTART}" == "true" ]]; then
	if sudo -n systemctl restart kasq-backend 2>/dev/null; then
		if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "backend" ]]; then
			wait_backend
		fi
		if [[ "\$DEPLOY_ONLY" == "all" || "\$DEPLOY_ONLY" == "frontend" ]]; then
			${RESTART_FRONTEND} || true
			${WAIT_FRONTEND}
		fi
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
	echo "    Frontend: ${DEPLOY_PATH}/frontend/build/ (PM2: ${DEPLOY_PM2_APP_NAME})"
	echo "    Log FE:   su - ${DEPLOY_PM2_USER} -c 'pm2 logs ${DEPLOY_PM2_APP_NAME}'"
else
	echo "    Frontend: ${DEPLOY_PATH}/frontend/build/ (systemd)"
fi
echo ""
echo "Pastikan backend/.env sudah ada di server (tidak ikut di-upload)."
