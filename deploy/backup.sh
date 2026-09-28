#!/usr/bin/env bash
# Consistent online copy of the SQLite database (safe while the app runs:
# VACUUM INTO takes a snapshot, unlike cp on a WAL-mode file). Keeps 14 days.
#
# Cron (crontab -e as wilber):
#   30 3 * * * /opt/linkmeqr/backup.sh >/dev/null 2>&1
set -euo pipefail

APP_DIR=/opt/linkmeqr
DEST="$APP_DIR/backups"
mkdir -p "$DEST"

"$APP_DIR/linkmeqr" -backup "$DEST/linkmeqr-$(date +%F-%H%M).db"
find "$DEST" -name 'linkmeqr-*.db' -mtime +14 -delete
