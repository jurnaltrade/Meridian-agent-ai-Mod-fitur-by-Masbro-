#!/bin/bash

# Configuration
BACKUP_REPO="git@github.com:nzf210/bc_meridian_go.git"
BACKUP_GIT_DIR="/home/deploy/meridian_backup_git"
PROJECT_DIR="/home/deploy/meridian-go/go-rewrite"
DOT_MERIDIAN="/home/deploy/.meridian"

# Unset Git environment variables injected by the shell runner environment to prevent repository hijacking
unset GIT_DIR GIT_WORK_TREE

# Log output to a file directly (prevents background job suspension SIGTTOU)
LOG_FILE="/home/deploy/meridian-go/go-rewrite/logs/backup.log"
mkdir -p "$(dirname "$LOG_FILE")"

exec >> "$LOG_FILE" 2>&1

echo "=================================================="
echo "Backup Started: $(date)"
echo "=================================================="

# 1. Initialize backup git repo if not exists
if [ ! -d "$BACKUP_GIT_DIR" ]; then
    echo "Cloning backup repository..."
    git clone "$BACKUP_REPO" "$BACKUP_GIT_DIR"
    if [ $? -ne 0 ]; then
        echo "Error: Failed to clone backup repository. Check credentials or SSH keys."
        exit 1
    fi
fi

cd "$BACKUP_GIT_DIR" || exit 1

# Ensure we are on the latest remote commit
echo "Pulling latest changes from remote..."
git pull origin main --rebase 2>/dev/null || git pull origin master --rebase 2>/dev/null

# 2. Create target structure in backup repository
mkdir -p "$BACKUP_GIT_DIR/dot-meridian"

# 3. Copy files to backup repo (EXCLUDING .env)
echo "Copying files..."
# Copy ~/.meridian/ files (excluding logs or lock files)
if [ -d "$DOT_MERIDIAN" ]; then
    cp -r "$DOT_MERIDIAN"/* "$BACKUP_GIT_DIR/dot-meridian/"
    rm -f "$BACKUP_GIT_DIR/dot-meridian"/*.log 2>/dev/null
    rm -f "$BACKUP_GIT_DIR/dot-meridian"/*.lock 2>/dev/null
fi

# Copy wallet.enc (mutlak)
if [ -f "$PROJECT_DIR/wallet.enc" ]; then
    cp "$PROJECT_DIR/wallet.enc" "$BACKUP_GIT_DIR/"
fi

# Copy user-config.json (optional)
if [ -f "$PROJECT_DIR/user-config.json" ]; then
    cp "$PROJECT_DIR/user-config.json" "$BACKUP_GIT_DIR/"
fi

# Copy hivemind-cache.json (optional)
if [ -f "$PROJECT_DIR/hivemind-cache.json" ]; then
    cp "$PROJECT_DIR/hivemind-cache.json" "$BACKUP_GIT_DIR/"
fi

# 4. Commit and push
echo "Checking for changes..."
git add .
if git diff-index --quiet HEAD --; then
    echo "No changes to backup."
else
    echo "Changes detected. Committing and pushing..."
    git commit -m "Automated Backup: $(date -u +'%Y-%m-%d %H:%M:%S UTC')"
    
    # Try to push to main first, then master if main fails
    git push origin main || git push origin master
    if [ $? -eq 0 ]; then
        echo "Success: Backup pushed successfully."
    else
        echo "Error: Failed to push backup to GitHub."
        exit 1
    fi
fi

echo "=================================================="
echo "Backup Completed: $(date)"
echo "=================================================="
