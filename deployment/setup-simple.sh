#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="Roma7-7-7/english-learning-bot"
INSTALL_DIR="/opt/english-learning-bot"
BIN_DIR="${INSTALL_DIR}/bin"
DATA_DIR="${INSTALL_DIR}/data"
BACKUP_DIR="${INSTALL_DIR}/backups"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}English Learning Bot Simple Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if running as root
if [[ "$EUID" -ne 0 ]]; then
    echo -e "${RED}Please run as root or with sudo${NC}"
    exit 1
fi

# Determine which user to run the service as
if [[ -n "$SUDO_USER" ]]; then
    SERVICE_USER="$SUDO_USER"
else
    echo -e "${YELLOW}Enter the username to run the service as (default: current user):${NC}"
    read -r SERVICE_USER
    if [[ -z "$SERVICE_USER" ]]; then
        SERVICE_USER=$(whoami)
    fi
fi

echo -e "${GREEN}Services will run as user: ${SERVICE_USER}${NC}"
echo ""

# Verify user exists
if ! id "$SERVICE_USER" &>/dev/null; then
    echo -e "${RED}User $SERVICE_USER does not exist${NC}"
    exit 1
fi

# Check if already installed
if [[ -d "${INSTALL_DIR}" ]]; then
    echo -e "${YELLOW}Warning: Installation directory ${INSTALL_DIR} already exists${NC}"

    # Check if database exists
    DB_FILE="${DATA_DIR}/english_learning.db"
    if [[ -f "$DB_FILE" ]]; then
        echo -e "${YELLOW}Database found at: ${DB_FILE}${NC}"

        # Create backup of existing database
        BACKUP_TIMESTAMP=$(date +'%Y%m%d_%H%M%S')
        DB_BACKUP_FILE="${BACKUP_DIR}/english_learning.db.backup.${BACKUP_TIMESTAMP}"

        echo -e "${GREEN}Creating safety backup of database...${NC}"
        mkdir -p "${BACKUP_DIR}"
        cp "$DB_FILE" "$DB_BACKUP_FILE"
        echo -e "${GREEN}✓ Database backed up to: ${DB_BACKUP_FILE}${NC}"
        echo ""
    fi

    echo -e "${YELLOW}This will update the installation (binary and scripts only).${NC}"
    echo -e "${GREEN}Your database and existing data will NOT be affected.${NC}"
    read -p "Do you want to continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Setup cancelled."
        exit 1
    fi
fi

echo -e "${GREEN}[1/7] Creating directory structure...${NC}"
mkdir -p "${BIN_DIR}"
mkdir -p "${DATA_DIR}"
mkdir -p "${BACKUP_DIR}"
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"
echo "✓ Directories created"

echo ""
echo -e "${GREEN}[2/7] Downloading deploy.sh script...${NC}"
curl -L -o "${INSTALL_DIR}/deploy.sh" \
    "https://raw.githubusercontent.com/${GITHUB_REPO}/main/deployment/deploy.sh"
chmod +x "${INSTALL_DIR}/deploy.sh"
chown "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}/deploy.sh"
echo "✓ Deploy script installed"

echo ""
echo -e "${GREEN}[3/7] Installing systemd services...${NC}"
# Download and install API service
curl -L -s "https://raw.githubusercontent.com/${GITHUB_REPO}/main/deployment/systemd/english-learning-api-simple.service" | \
    sed "s/{{SERVICE_USER}}/${SERVICE_USER}/g" > /etc/systemd/system/english-learning-api.service

# Download and install Bot service
curl -L -s "https://raw.githubusercontent.com/${GITHUB_REPO}/main/deployment/systemd/english-learning-bot-simple.service" | \
    sed "s/{{SERVICE_USER}}/${SERVICE_USER}/g" > /etc/systemd/system/english-learning-bot.service

systemctl daemon-reload
echo "✓ Systemd services installed"

echo ""
echo -e "${GREEN}[4/7] Configuring sudoers for passwordless service management...${NC}"
# Create sudoers file for the service user
cat > /etc/sudoers.d/english-learning-bot <<EOF
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start english-learning-api.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop english-learning-api.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart english-learning-api.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status english-learning-api.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl is-active english-learning-api.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start english-learning-bot.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop english-learning-bot.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart english-learning-bot.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status english-learning-bot.service
${SERVICE_USER} ALL=(ALL) NOPASSWD: /bin/systemctl is-active english-learning-bot.service
EOF
chmod 0440 /etc/sudoers.d/english-learning-bot
echo "✓ Sudoers configuration installed"

echo ""
echo -e "${GREEN}[5/7] Setting up environment file...${NC}"
ENV_FILE="${INSTALL_DIR}/.env"

if [[ -f "$ENV_FILE" ]]; then
    echo -e "${YELLOW}Environment file already exists at: ${ENV_FILE}${NC}"
    echo -e "${YELLOW}Skipping environment file creation to preserve existing configuration${NC}"
else
    echo ""
    echo -e "${BLUE}Configuration Setup:${NC}"
    echo ""

    # Bot configuration
    echo -e "${YELLOW}--- Bot Configuration ---${NC}"
    echo -e "${BLUE}Enter your Telegram bot token (from @BotFather):${NC}"
    read -r BOT_TELEGRAM_TOKEN

    echo -e "${BLUE}Enter allowed Telegram chat IDs (comma-separated):${NC}"
    read -r BOT_ALLOWED_CHAT_IDS

    # API configuration
    echo ""
    echo -e "${YELLOW}--- API Configuration ---${NC}"
    echo -e "${BLUE}Enter JWT secret (random string for session security):${NC}"
    read -r API_JWT_SECRET

    echo -e "${BLUE}Enter allowed CORS origins (e.g., https://yourdomain.com):${NC}"
    read -r API_CORS_ORIGINS

    echo -e "${BLUE}Enter cookie domain (e.g., yourdomain.com):${NC}"
    read -r API_COOKIE_DOMAIN

    echo -e "${BLUE}Enter JWT audience (comma-separated, e.g., web,mobile):${NC}"
    read -r API_JWT_AUDIENCE

    cat > "$ENV_FILE" <<EOF
# Bot Configuration
BOT_TELEGRAM_TOKEN=${BOT_TELEGRAM_TOKEN}
BOT_ALLOWED_CHAT_IDS=${BOT_ALLOWED_CHAT_IDS}
BOT_DB_PATH=${DATA_DIR}/english_learning.db?cache=shared&mode=rwc

# API Configuration
API_TELEGRAM_TOKEN=${BOT_TELEGRAM_TOKEN}
API_TELEGRAM_ALLOWED_CHAT_IDS=${BOT_ALLOWED_CHAT_IDS}
API_DB_PATH=${DATA_DIR}/english_learning.db?cache=shared&mode=rwc
API_HTTP_JWT_SECRET=${API_JWT_SECRET}
API_HTTP_CORS_ALLOW_ORIGINS=${API_CORS_ORIGINS}
API_HTTP_COOKIE_DOMAIN=${API_COOKIE_DOMAIN}
API_HTTP_JWT_AUDIENCE=${API_JWT_AUDIENCE}

# Optional: Uncomment and modify these to override defaults
# BOT_SCHEDULE_PUBLISH_INTERVAL=15m
# BOT_SCHEDULE_HOUR_FROM=9
# BOT_SCHEDULE_HOUR_TO=22
# BOT_SCHEDULE_LOCATION=Europe/Kyiv
# API_SERVER_ADDR=:8080
# API_DEV=false
# BOT_DEV=false
EOF

    chmod 600 "$ENV_FILE"
    chown "${SERVICE_USER}:${SERVICE_USER}" "$ENV_FILE"
    echo "✓ Environment file created at: ${ENV_FILE}"
fi

echo ""
echo -e "${GREEN}[6/7] Running initial deployment...${NC}"
sudo -u "${SERVICE_USER}" "${INSTALL_DIR}/deploy.sh"
echo "✓ Initial deployment completed"

echo ""
echo -e "${GREEN}[7/7] Enabling service auto-start...${NC}"
systemctl enable english-learning-api.service
systemctl enable english-learning-bot.service
echo "✓ Services will start automatically on boot"

echo ""
echo -e "${GREEN}[8/8] Verifying installation...${NC}"

# Check service status
API_ACTIVE=$(systemctl is-active english-learning-api.service || echo "inactive")
BOT_ACTIVE=$(systemctl is-active english-learning-bot.service || echo "inactive")

if [[ "$API_ACTIVE" = "active" && "$BOT_ACTIVE" = "active" ]]; then
    echo "✓ Both services are running"
else
    echo -e "${YELLOW}⚠ Some services may not be running:${NC}"
    echo "  API: $API_ACTIVE"
    echo "  Bot: $BOT_ACTIVE"
    echo -e "${YELLOW}Check configuration and logs${NC}"
fi

# Check version
if [[ -f "${INSTALL_DIR}/current_version" ]]; then
    VERSION=$(cat "${INSTALL_DIR}/current_version")
    echo "✓ Installed version: ${VERSION}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Show database backup info if one was created
if [[ -n "${DB_BACKUP_FILE}" && -f "${DB_BACKUP_FILE}" ]]; then
    echo -e "${GREEN}Database Safety Info:${NC}"
    echo "  A backup of your existing database was created at:"
    echo "  ${DB_BACKUP_FILE}"
    echo ""
fi

echo -e "${YELLOW}Important Information:${NC}"
echo ""
echo "Configuration file: ${ENV_FILE}"
echo "  Edit this file to configure the bot and API (requires service restart)"
echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo "  API Status:  sudo systemctl status english-learning-api.service"
echo "  Bot Status:  sudo systemctl status english-learning-bot.service"
echo "  API Logs:    sudo journalctl -u english-learning-api.service -f"
echo "  Bot Logs:    sudo journalctl -u english-learning-bot.service -f"
echo "  Deploy:      ${INSTALL_DIR}/deploy.sh"
echo "  Stop All:    sudo systemctl stop english-learning-api.service english-learning-bot.service"
echo "  Start All:   sudo systemctl start english-learning-api.service english-learning-bot.service"
echo "  Restart All: sudo systemctl restart english-learning-api.service english-learning-bot.service"
echo ""
echo -e "${YELLOW}Manual Backups:${NC}"
echo "  Database location: ${DATA_DIR}/english_learning.db"
echo "  Backup with: scp ${SERVICE_USER}@your-server:${DATA_DIR}/english_learning.db ~/backups/"
echo ""
