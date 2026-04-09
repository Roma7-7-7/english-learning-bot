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
    local message
    message="$1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $message" | tee -a "$LOG_FILE"
    return 0
}

log_colored() {
    local message
    local color
    message="$1"
    color="$2"
    echo -e "${color}[$(date '+%Y-%m-%d %H:%M:%S')] $message${NC}" | tee -a "$LOG_FILE"
    return 0
}

log "Starting deployment check..."

# Get latest release tag from GitHub
log "Fetching latest release information..."
LATEST_RELEASE=$(curl -sf "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [[ -z "$LATEST_RELEASE" ]]; then
    log_colored "Failed to fetch latest release information from GitHub" "$RED"
    exit 1
fi

log "Latest release: $LATEST_RELEASE"

# Check current version
CURRENT_VERSION=""
if [[ -f "$VERSION_FILE" ]]; then
    CURRENT_VERSION=$(cat "$VERSION_FILE")
    log "Current version: $CURRENT_VERSION"
else
    log_colored "No current version found (first deployment)" "$YELLOW"
fi

# Compare versions
if [[ "$CURRENT_VERSION" = "$LATEST_RELEASE" ]]; then
    log_colored "Already running the latest version ($LATEST_RELEASE). No deployment needed." "$GREEN"
    exit 0
fi

log_colored "New version available! Deploying $LATEST_RELEASE..." "$YELLOW"

# Detect architecture
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)
        ARCH_SUFFIX="amd64"
        ;;
    aarch64|arm64)
        ARCH_SUFFIX="arm64"
        ;;
    *)
        log_colored "Unsupported architecture: ${ARCH}" "$RED"
        log_colored "Supported architectures: x86_64 (amd64), aarch64/arm64" "$RED"
        exit 1
        ;;
esac

log "Detected architecture: ${ARCH} (will download ${ARCH_SUFFIX} binary)"

# Create temporary directory for downloads
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binary
log "Downloading binary..."

download_file() {
    local filename
    filename="$1"
    local url="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_RELEASE}/${filename}"

    log "  Downloading $filename..."
    if ! curl -sfL "$url" -o "${TMP_DIR}/${filename}"; then
        log_colored "Failed to download $filename" "$RED"
        return 1
    fi
    return 0
}

# Download architecture-specific binary
download_file "english-learning-bot-${ARCH_SUFFIX}" || exit 1
download_file "VERSION" || exit 1

# Rename binary to remove architecture suffix for local use
mv "${TMP_DIR}/english-learning-bot-${ARCH_SUFFIX}" "${TMP_DIR}/english-learning-bot"

# Make binary executable
chmod +x "${TMP_DIR}/english-learning-bot"

# Stop service
log "Stopping service..."
sudo systemctl stop english-learning-bot.service || log_colored "Bot service was not running" "$YELLOW"

# Backup old binary (optional but recommended)
if [[ -f "${BIN_DIR}/english-learning-bot" ]]; then
    log "Backing up old binary..."
    mkdir -p "${INSTALL_DIR}/backups"
    BACKUP_DIR="${INSTALL_DIR}/backups/backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    cp "${BIN_DIR}/english-learning-bot" "$BACKUP_DIR/" 2>/dev/null || true
    cp "$VERSION_FILE" "$BACKUP_DIR/" 2>/dev/null || true

    # Keep only last 5 backups
    cd "${INSTALL_DIR}/backups" && ls -t | tail -n +6 | xargs -r rm -rf
fi

# Copy new binary
log "Installing new binary..."
mkdir -p "$BIN_DIR"
cp "${TMP_DIR}/english-learning-bot" "${BIN_DIR}/"
cp "${TMP_DIR}/VERSION" "${BIN_DIR}/"

# Update version file
echo "$LATEST_RELEASE" > "$VERSION_FILE"

# Start service
log "Starting service..."
sudo systemctl start english-learning-bot.service

# Wait a moment and check if service is running
sleep 2

BOT_STATUS=$(sudo systemctl is-active english-learning-bot.service)

if [[ "$BOT_STATUS" = "active" ]]; then
    log_colored "Deployment successful! Service is running." "$GREEN"
    log "Status: $BOT_STATUS"
    log "Deployed version: $LATEST_RELEASE"
    exit 0
else
    log_colored "WARNING: Service may not be running properly!" "$RED"
    log "Status: $BOT_STATUS"
    log "Check logs with: journalctl -u english-learning-bot.service -f"
    exit 1
fi
