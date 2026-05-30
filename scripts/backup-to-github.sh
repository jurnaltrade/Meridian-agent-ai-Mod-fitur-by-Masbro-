#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MANIFEST_FILE="$REPO_ROOT/backup-files.txt"
ARTIFACT_DIR="$REPO_ROOT/backup-artifacts"
WORKTREE_DIR="$REPO_ROOT/backup-worktree"
ENV_FILE="$REPO_ROOT/.env"
BACKUP_ENV_FILE="$REPO_ROOT/.backup.env"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

if [[ -f "$BACKUP_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$BACKUP_ENV_FILE"
  set +a
fi

BACKUP_GIT_REMOTE="${BACKUP_GIT_REMOTE:-git@github.com:nzf210/meridian_bot_backup.git}"
BACKUP_BRANCH="${BACKUP_BRANCH:-main}"
BACKUP_PASSPHRASE="${BACKUP_PASSPHRASE:-}"

if [[ -z "$BACKUP_PASSPHRASE" ]]; then
  echo "BACKUP_PASSPHRASE is not set in $ENV_FILE or $BACKUP_ENV_FILE" >&2
  exit 1
fi

mkdir -p "$ARTIFACT_DIR"
rm -rf "$WORKTREE_DIR"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
host_name="$(hostname -s)"
archive_name="meridian-backup-${host_name}-${timestamp}.tar.gz"
encrypted_name="${archive_name}.enc"
archive_path="$ARTIFACT_DIR/$archive_name"
encrypted_path="$ARTIFACT_DIR/$encrypted_name"
manifest_copy="$ARTIFACT_DIR/backup-files-${host_name}-${timestamp}.txt"
metadata_path="$ARTIFACT_DIR/backup-metadata-${host_name}-${timestamp}.json"
staging_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$staging_dir" "$WORKTREE_DIR"
}
trap cleanup EXIT

cp "$MANIFEST_FILE" "$manifest_copy"

while IFS= read -r relpath; do
  [[ -z "$relpath" ]] && continue
  src="$REPO_ROOT/$relpath"
  if [[ ! -f "$src" ]]; then
    echo "Skipping missing file: $relpath"
    continue
  fi
  mkdir -p "$staging_dir/$(dirname "$relpath")"
  cp "$src" "$staging_dir/$relpath"
done < "$MANIFEST_FILE"

tar -C "$staging_dir" -czf "$archive_path" .
openssl enc -aes-256-cbc -pbkdf2 -salt -in "$archive_path" -out "$encrypted_path" -pass "pass:$BACKUP_PASSPHRASE"
rm -f "$archive_path"

archive_sha256="$(sha256sum "$encrypted_path" | awk '{print $1}')"
cat > "$metadata_path" <<EOF
{
  "created_at_utc": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "hostname": "$host_name",
  "branch": "$BACKUP_BRANCH",
  "source_repo": "$(git -C "$REPO_ROOT" remote get-url origin 2>/dev/null || echo unknown)",
  "encrypted_archive": "$(basename "$encrypted_path")",
  "sha256": "$archive_sha256"
}
EOF

if git ls-remote --exit-code --heads "$BACKUP_GIT_REMOTE" "$BACKUP_BRANCH" >/dev/null 2>&1; then
  git clone --branch "$BACKUP_BRANCH" "$BACKUP_GIT_REMOTE" "$WORKTREE_DIR"
else
  git clone "$BACKUP_GIT_REMOTE" "$WORKTREE_DIR"
  git -C "$WORKTREE_DIR" checkout --orphan "$BACKUP_BRANCH"
fi
cp "$encrypted_path" "$WORKTREE_DIR/"
cp "$manifest_copy" "$WORKTREE_DIR/"
cp "$metadata_path" "$WORKTREE_DIR/"

git -C "$WORKTREE_DIR" add .
if git -C "$WORKTREE_DIR" diff --cached --quiet; then
  echo "No backup changes to commit"
  exit 0
fi

git -C "$WORKTREE_DIR" commit -m "backup: ${host_name} ${timestamp}"
git -C "$WORKTREE_DIR" push origin "$BACKUP_BRANCH"

echo "Backup pushed: $(basename "$encrypted_path")"
