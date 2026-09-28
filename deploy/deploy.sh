#!/usr/bin/env bash
# Build LinkMeQR locally and ship it to the server. Run from the repo root in
# Git Bash (Windows) or any POSIX shell:
#
#   bash deploy/deploy.sh
#
# The server needs neither Go nor Node: the backend is cross-compiled to a
# static Linux binary (pure-Go SQLite, CGO off) and the frontend is shipped
# already built. Data (data/, media/, .env, backups/) is never touched.
set -euo pipefail

SERVER="${LINKMEQR_SERVER:-wilber@172.16.13.216}"
SSH_KEY="${LINKMEQR_SSH_KEY:-$HOME/.ssh/id_ed25519_apps}"
APP_DIR=/opt/linkmeqr
SSH=(ssh -i "$SSH_KEY" -o IdentitiesOnly=yes "$SERVER")

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/deploy/.build"
rm -rf "$OUT" && mkdir -p "$OUT"

echo "==> backend (linux/amd64)"
(
	cd "$ROOT/backend"
	export CGO_ENABLED=0 GOOS=linux GOARCH=amd64
	go build -trimpath -ldflags="-s -w" -o "$OUT/linkmeqr" ./cmd/api
	go build -trimpath -ldflags="-s -w" -o "$OUT/seed" ./seed
)

echo "==> frontend"
(
	cd "$ROOT/frontend"
	[ -d node_modules ] || npm ci
	npm run build
)
cp -r "$ROOT/frontend/dist" "$OUT/dist"
cp "$ROOT/deploy/backup.sh" "$OUT/backup.sh"

echo "==> upload to $SERVER:$APP_DIR"
tar -C "$OUT" -czf - linkmeqr seed dist backup.sh | "${SSH[@]}" "
	set -e
	mkdir -p $APP_DIR/.incoming $APP_DIR/data $APP_DIR/media $APP_DIR/backups
	rm -rf $APP_DIR/.incoming/*
	tar -C $APP_DIR/.incoming -xzf -
	chmod +x $APP_DIR/.incoming/linkmeqr $APP_DIR/.incoming/seed $APP_DIR/.incoming/backup.sh
"

echo "==> swap + restart"
"${SSH[@]}" "
	set -e
	cd $APP_DIR
	# Snapshot the database before the new binary runs its migrations.
	if [ -f data/linkmeqr.db ] && [ -x linkmeqr ]; then
		./linkmeqr -backup backups/linkmeqr-predeploy-\$(date +%F-%H%M%S).db
	fi
	[ -f linkmeqr ] && mv -f linkmeqr linkmeqr.anterior
	mv -f .incoming/linkmeqr linkmeqr
	mv -f .incoming/seed seed
	mv -f .incoming/backup.sh backup.sh
	rm -rf dist.anterior && [ -d dist ] && mv dist dist.anterior
	mv .incoming/dist dist
	rmdir .incoming
	sudo -n systemctl restart linkmeqr
	sleep 2
	systemctl is-active linkmeqr
	curl -fsS http://127.0.0.1:8090/healthz && echo ' healthz OK'
"
rm -rf "$OUT"
echo "==> listo"
