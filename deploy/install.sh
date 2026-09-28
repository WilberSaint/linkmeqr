#!/usr/bin/env bash
# One-time server setup for LinkMeQR — the only step that needs root.
# Expects the app already uploaded to /opt/linkmeqr (see deploy/deploy.sh)
# and this script, linkmeqr.service and Caddyfile.snippet side by side.
#
#   sudo bash install.sh
#
# Safe to re-run: every step checks before changing anything.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
CADDYFILE=/etc/caddy/Caddyfile

[ "$(id -u)" -eq 0 ] || { echo "Correr con sudo: sudo bash $0"; exit 1; }
[ -x /opt/linkmeqr/linkmeqr ] || { echo "Falta /opt/linkmeqr/linkmeqr"; exit 1; }

echo "==> servicio systemd"
install -m 644 "$HERE/linkmeqr.service" /etc/systemd/system/linkmeqr.service
systemctl daemon-reload
systemctl enable --now linkmeqr
systemctl restart linkmeqr

echo "==> permiso para que deploy.sh reinicie el servicio sin contraseña"
echo 'wilber ALL=(root) NOPASSWD: /usr/bin/systemctl restart linkmeqr' > /etc/sudoers.d/linkmeqr
chmod 440 /etc/sudoers.d/linkmeqr
visudo -cf /etc/sudoers.d/linkmeqr

echo "==> Caddy"
if grep -q '^linkmeqr.org {' "$CADDYFILE"; then
	echo "    el bloque de linkmeqr.org ya existe, no se toca"
else
	cp "$CADDYFILE" "$CADDYFILE.antes-linkmeqr-$(date +%F-%H%M%S)"
	{ echo; grep -v '^#' "$HERE/Caddyfile.snippet"; } >> "$CADDYFILE"
	if ! caddy validate --config "$CADDYFILE" --adapter caddyfile >/dev/null 2>&1; then
		echo "    Caddyfile inválido, restaurando el anterior"
		cp "$(ls -t "$CADDYFILE".antes-linkmeqr-* | head -1)" "$CADDYFILE"
		exit 1
	fi
fi
# `caddy validate` above runs as root and can create the site's log file
# owned by root; Caddy itself runs as `caddy` and would then fail to open it.
touch /var/log/caddy/linkmeqr.log
chown caddy:caddy /var/log/caddy/linkmeqr.log
systemctl reload caddy

sleep 3
echo "==> verificación"
systemctl is-active linkmeqr
curl -fsS http://127.0.0.1:8090/healthz && echo "  <- app OK"
echo "Listo. Falta en Cloudflare: SSL/TLS -> modo 'Full (strict)'."
