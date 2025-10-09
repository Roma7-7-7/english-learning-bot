#!/bin/bash
set -e

# Configuration
GITHUB_REPO="Roma7-7-7/english-learning-bot"
INSTALL_DIR="/opt/english-learning-bot"
BIN_DIR="${INSTALL_DIR}/bin"
VERSION_FILE="${INSTALL_DIR}/current_version"
LOG_FILE="${INSTALL_DIR}/deployment.log"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_colored() {
    echo -e "${2}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ]; then
    log_colored "Please run with sudo: sudo $0" "$RED"
    exit 1
fi

log "Starting deployment check..."

# Get latest release tag from GitHub
log "Fetching latest release information..."
LATEST_RELEASE=$(curl -sf "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    log_colored "Failed to fetch latest release information from GitHub" "$RED"
    exit 1
fi

log "Latest release: $LATEST_RELEASE"

# Check current version
CURRENT_VERSION=""
if [ -f "$VERSION_FILE" ]; then
    CURRENT_VERSION=$(cat "$VERSION_FILE")
    log "Current version: $CURRENT_VERSION"
else
    log_colored "No current version found (first deployment)" "$YELLOW"
fi

# Compare versions
if [ "$CURRENT_VERSION" = "$LATEST_RELEASE" ]; then
    log_colored "Already running the latest version ($LATEST_RELEASE). No deployment needed." "$GREEN"
    exit 0
fi

log_colored "New version available! Deploying $LATEST_RELEASE..." "$YELLOW"

# Create temporary directory for downloads
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binaries
log "Downloading binaries..."

download_file() {
    local filename=$1
    local url="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_RELEASE}/${filename}"

    log "  Downloading $filename..."
    if ! curl -sfL "$url" -o "${TMP_DIR}/${filename}"; then
        log_colored "Failed to download $filename" "$RED"
        return 1
    fi
    return 0
}

# Download all required files
download_file "english-learning-api" || exit 1
download_file "english-learning-bot" || exit 1
download_file "VERSION" || exit 1

# Make binaries executable
chmod +x "${TMP_DIR}/english-learning-api"
chmod +x "${TMP_DIR}/english-learning-bot"

# Stop services
log "Stopping services..."
systemctl stop english-learning-api.service || log_colored "API service was not running" "$YELLOW"
systemctl stop english-learning-bot.service || log_colored "Bot service was not running" "$YELLOW"

# Backup old binaries (optional but recommended)
if [ -f "${BIN_DIR}/english-learning-api" ]; then
    log "Backing up old binaries..."
    mkdir -p "${INSTALL_DIR}/backups"
    BACKUP_DIR="${INSTALL_DIR}/backups/backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    cp "${BIN_DIR}/english-learning-api" "$BACKUP_DIR/" 2>/dev/null || true
    cp "${BIN_DIR}/english-learning-bot" "$BACKUP_DIR/" 2>/dev/null || true
    cp "$VERSION_FILE" "$BACKUP_DIR/" 2>/dev/null || true

    # Keep only last 5 backups
    cd "${INSTALL_DIR}/backups" && ls -t | tail -n +6 | xargs -r rm -rf
fi

# Copy new binaries
log "Installing new binaries..."
mkdir -p "$BIN_DIR"
cp "${TMP_DIR}/english-learning-api" "${BIN_DIR}/"
cp "${TMP_DIR}/english-learning-bot" "${BIN_DIR}/"
cp "${TMP_DIR}/VERSION" "${BIN_DIR}/"

# Update version file
echo "$LATEST_RELEASE" > "$VERSION_FILE"

# Set proper ownership
chown -R ec2-user:ec2-user "$BIN_DIR"

# Start services
log "Starting services..."
systemctl start english-learning-api.service
systemctl start english-learning-bot.service

# Wait a moment and check if services are running
sleep 2

API_STATUS=$(systemctl is-active english-learning-api.service)
BOT_STATUS=$(systemctl is-active english-learning-bot.service)

if [ "$API_STATUS" = "active" ] && [ "$BOT_STATUS" = "active" ]; then
    log_colored "Deployment successful! Both services are running." "$GREEN"
    log "API Status: $API_STATUS"
    log "Bot Status: $BOT_STATUS"
    log "Deployed version: $LATEST_RELEASE"
    exit 0
else
    log_colored "WARNING: Some services may not be running properly!" "$RED"
    log "API Status: $API_STATUS"
    log "Bot Status: $BOT_STATUS"
    log "Check logs with: journalctl -u english-learning-api.service -f"
    log "                 journalctl -u english-learning-bot.service -f"
    exit 1
fi
