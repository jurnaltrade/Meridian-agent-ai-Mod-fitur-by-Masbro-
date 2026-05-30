#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LOG_DIR="$REPO_ROOT/logs"
BACKUP_CMD="$REPO_ROOT/scripts/backup-to-github.sh"
CRON_LINE="0 */6 * * * $BACKUP_CMD >> $LOG_DIR/backup.log 2>&1"

mkdir -p "$LOG_DIR"

current_crontab="$(mktemp)"
new_crontab="$(mktemp)"

cleanup() {
  rm -f "$current_crontab" "$new_crontab"
}
trap cleanup EXIT

if crontab -l >"$current_crontab" 2>/dev/null; then
  grep -F -v "$BACKUP_CMD" "$current_crontab" >"$new_crontab" || true
else
  : >"$new_crontab"
fi

printf '%s\n' "$CRON_LINE" >>"$new_crontab"
crontab "$new_crontab"

echo "Installed cron: $CRON_LINE"
