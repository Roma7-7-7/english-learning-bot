#!/bin/bash
set -e

# Configuration
GITHUB_REPO="Roma7-7-7/english-learning-bot"
INSTALL_DIR="/opt/english-learning-bot"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() {
    echo -e "\n${BLUE}===> $1${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}! $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "Please run with sudo: sudo $0"
    exit 1
fi

echo -e "${BLUE}"
cat << "EOF"
╔═══════════════════════════════════════════════════════╗
║   English Learning Bot - EC2 Initial Setup            ║
║                                                       ║
║   This script will set up your EC2 instance to run   ║
║   the English Learning Bot with automated deployment ║
╚═══════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

# Step 1: Create installation directory
print_step "Creating installation directory"
mkdir -p "$INSTALL_DIR"
mkdir -p "${INSTALL_DIR}/bin"
mkdir -p "${INSTALL_DIR}/data"
mkdir -p "${INSTALL_DIR}/backups"
chown -R ec2-user:ec2-user "$INSTALL_DIR"
print_success "Installation directory created at $INSTALL_DIR"

# Step 2: Download deployment scripts from GitHub
print_step "Downloading deployment scripts"
REPO_URL="https://raw.githubusercontent.com/${GITHUB_REPO}/main"

curl -sfL "${REPO_URL}/deployment/deploy.sh" -o "${INSTALL_DIR}/deploy.sh"
chmod +x "${INSTALL_DIR}/deploy.sh"
print_success "Deployment script installed"

# Step 3: Install systemd service files
print_step "Installing systemd services"
curl -sfL "${REPO_URL}/deployment/systemd/english-learning-api.service" -o /etc/systemd/system/english-learning-api.service
curl -sfL "${REPO_URL}/deployment/systemd/english-learning-bot.service" -o /etc/systemd/system/english-learning-bot.service
systemctl daemon-reload
print_success "Systemd services installed"

# Step 4: Create .env file template
print_step "Creating .env file template"
if [ ! -f "${INSTALL_DIR}/.env" ]; then
    cat > "${INSTALL_DIR}/.env" << 'ENVEOF'
# Bot Configuration
BOT_DB_PATH=file:/opt/english-learning-bot/data/english_learning.db?cache=shared&mode=rwc

# API Configuration
API_DB_PATH=file:/opt/english-learning-bot/data/english_learning.db?cache=shared&mode=rwc
API_HTTP_CORS_ALLOW_ORIGINS=https://bot.domain
API_HTTP_JWT_AUDIENCE=https://api.bot.domain
API_HTTP_COOKIE_DOMAIN=bot.domain

# AWS
AWS_REGION=eu-central-1
ENVEOF
    chown ec2-user:ec2-user "${INSTALL_DIR}/.env"
    print_warning ".env file created - YOU MUST EDIT THIS FILE WITH YOUR CREDENTIALS"
    print_warning "Edit it with: sudo nano ${INSTALL_DIR}/.env"
else
    print_warning ".env file already exists - skipping creation"
fi

# Step 5: Run initial deployment
print_step "Running initial deployment"
"${INSTALL_DIR}/deploy.sh"

# Step 6: Enable services to start on boot
print_step "Enabling services to start on boot"
systemctl enable english-learning-api.service
systemctl enable english-learning-bot.service
print_success "Services enabled"

# Step 6.5: Configure sudoers for passwordless systemctl
print_step "Configuring passwordless systemctl access for ec2-user"
SUDOERS_FILE="/etc/sudoers.d/english-learning-bot"
cat > "$SUDOERS_FILE" << 'SUDOEOF'
# Allow ec2-user to manage english-learning-bot services without password
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl start english-learning-api.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop english-learning-api.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart english-learning-api.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl start english-learning-bot.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop english-learning-bot.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart english-learning-bot.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl status english-learning-api.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl status english-learning-bot.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl is-active english-learning-api.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl is-active english-learning-bot.service
SUDOEOF
chmod 0440 "$SUDOERS_FILE"
print_success "Sudoers configured - ec2-user can now manage services without password"

# Step 7: Set up database backup to S3
print_step "Setting up database backup to S3"
echo ""
echo -e "${YELLOW}Configure S3 backup (optional):${NC}"
echo "If you want automatic daily backups to S3, provide your S3 bucket path."
echo "Example: s3://my-bucket/english-learning-bot/backups"
echo ""
read -p "Enter S3 bucket path (or press Enter to skip): " S3_BUCKET_PATH

if [ -n "$S3_BUCKET_PATH" ]; then
    # Download backup script
    curl -sfL "${REPO_URL}/deployment/backup.sh" -o "${INSTALL_DIR}/backup.sh"
    chmod +x "${INSTALL_DIR}/backup.sh"
    chown ec2-user:ec2-user "${INSTALL_DIR}/backup.sh"

    # Create backup config
    cat > "${INSTALL_DIR}/.backup_config" << BACKUPEOF
# S3 Backup Configuration
S3_BUCKET_PATH="${S3_BUCKET_PATH}"
BACKUPEOF
    chown ec2-user:ec2-user "${INSTALL_DIR}/.backup_config"
    chmod 600 "${INSTALL_DIR}/.backup_config"

    # Add backup cron job to ec2-user's crontab (daily at 20:00)
    BACKUP_CRON="0 20 * * * ${INSTALL_DIR}/backup.sh >> ${INSTALL_DIR}/backup.log 2>&1"
    if sudo -u ec2-user crontab -l 2>/dev/null | grep -Fq "${INSTALL_DIR}/backup.sh"; then
        print_warning "Backup cron job already exists - skipping"
    else
        (sudo -u ec2-user crontab -l 2>/dev/null; echo "$BACKUP_CRON") | sudo -u ec2-user crontab -
        print_success "Daily backup configured (runs at 20:00 UTC)"
        echo "  S3 path: $S3_BUCKET_PATH"
    fi
else
    print_warning "S3 backup skipped - you can configure it later"
    echo "  To set up later: edit ${INSTALL_DIR}/.backup_config and add cron job"
fi

# Final summary
echo -e "\n${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════╗
║              Setup Complete!                          ║
╚═══════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

print_success "Installation directory: $INSTALL_DIR"
print_success "Services: english-learning-api, english-learning-bot"
if [ -f "${INSTALL_DIR}/.backup_config" ]; then
    print_success "Database backup: Daily at 20:00 UTC to S3"
fi

echo -e "\n${YELLOW}NEXT STEPS:${NC}"
echo "1. Edit the .env file with your credentials:"
echo "   sudo nano ${INSTALL_DIR}/.env"
echo ""
echo "2. After editing .env, restart the services:"
echo "   sudo systemctl restart english-learning-api.service"
echo "   sudo systemctl restart english-learning-bot.service"
echo ""
echo "${BLUE}USEFUL COMMANDS:${NC}"
echo "  Check service status:     systemctl status english-learning-api.service"
echo "  View logs:                journalctl -u english-learning-api.service -f"
echo "  Restart services:         systemctl restart english-learning-api.service"
echo "  Manual deployment:        ${INSTALL_DIR}/deploy.sh"
echo "  View deployment log:      tail -f ${INSTALL_DIR}/deployment.log"
if [ -f "${INSTALL_DIR}/.backup_config" ]; then
    echo "  Manual backup:            ${INSTALL_DIR}/backup.sh"
    echo "  View backup log:          tail -f ${INSTALL_DIR}/backup.log"
fi
echo ""
echo -e "${YELLOW}NOTE:${NC}"
echo "- Deployment is MANUAL ONLY (no automatic updates)"
echo "- Run '${INSTALL_DIR}/deploy.sh' when you want to update"
echo "- No sudo needed - passwordless systemctl is configured"
echo ""
