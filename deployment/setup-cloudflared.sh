#!/bin/bash
set -e

# Cloudflare Tunnel Setup Script
# This script automates the installation and configuration of cloudflared
# to expose the English Learning Bot API through Cloudflare Tunnel

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

print_usage() {
    cat << EOF
Usage: sudo $0 [OPTIONS]

Sets up Cloudflare Tunnel to expose the English Learning Bot API.

OPTIONS:
    -t, --token TOKEN          Cloudflare Tunnel token (required)
    -n, --tunnel-name NAME     Tunnel name for identification (default: english-learning-bot)
    -p, --api-port PORT        Local API port to expose (default: 8080)
    -h, --help                 Show this help message

EXAMPLES:
    # Basic setup with tunnel token
    sudo $0 --token 1234567890987654321

    # Custom tunnel name and port
    sudo $0 --token YOUR_TOKEN --tunnel-name my-bot --api-port 8080

NOTES:
    1. Get your tunnel token from Cloudflare Zero Trust Dashboard:
       - Go to https://one.dash.cloudflare.com/
       - Navigate to "Networks" -> "Tunnels"
       - Create a new tunnel or select existing one
       - Copy the installation token

    2. This script will:
       - Install cloudflared binary
       - Create systemd service
       - Configure tunnel to route traffic to localhost:PORT
       - Enable auto-start on boot

    3. After setup, your API will be accessible via:
       https://YOUR-TUNNEL-SUBDOMAIN.YOUR-DOMAIN.com

EOF
}

# Default values
TUNNEL_NAME="english-learning-bot"
API_PORT="8080"
TUNNEL_TOKEN=""
INSTALL_DIR="/opt/english-learning-bot"
CLOUDFLARED_CONFIG_DIR="/etc/cloudflared"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--token)
            TUNNEL_TOKEN="$2"
            shift 2
            ;;
        -n|--tunnel-name)
            TUNNEL_NAME="$2"
            shift 2
            ;;
        -p|--api-port)
            API_PORT="$2"
            shift 2
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Validate required parameters
if [ -z "$TUNNEL_TOKEN" ]; then
    print_error "Tunnel token is required"
    echo ""
    print_usage
    exit 1
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "Please run with sudo: sudo $0 [OPTIONS]"
    exit 1
fi

echo -e "${BLUE}"
cat << "EOF"
╔═══════════════════════════════════════════════════════╗
║   Cloudflare Tunnel Setup                             ║
║                                                       ║
║   This script will configure cloudflared to expose   ║
║   your English Learning Bot API securely             ║
╚═══════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo "Configuration:"
echo "  Tunnel Name: $TUNNEL_NAME"
echo "  Local Port:  $API_PORT"
echo "  Token:       ${TUNNEL_TOKEN:0:20}..."
echo ""

# Step 1: Install cloudflared
print_step "Installing cloudflared from official Cloudflare repository"

# Add Cloudflare repository
print_step "Adding Cloudflare yum repository"
if [ ! -f /etc/yum.repos.d/cloudflare.repo ]; then
    cat > /etc/yum.repos.d/cloudflare.repo << 'REPOEOF'
[cloudflare]
name=Cloudflare Repository
baseurl=https://pkg.cloudflare.com/cloudflared/rpm
enabled=1
gpgcheck=1
gpgkey=https://pkg.cloudflare.com/cloudflare-main.gpg
REPOEOF
    print_success "Cloudflare repository added"
else
    print_warning "Cloudflare repository already exists"
fi

# Install cloudflared using yum
print_step "Installing cloudflared package"
if yum install -y cloudflared; then
    print_success "cloudflared installed successfully via yum"
else
    print_error "Failed to install cloudflared via yum"
    print_warning "Falling back to GitHub release download..."

    # Fallback: Detect architecture and download from GitHub
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            CLOUDFLARED_ARCH="amd64"
            ;;
        aarch64|arm64)
            CLOUDFLARED_ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    CLOUDFLARED_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${CLOUDFLARED_ARCH}"
    if curl -sfL "$CLOUDFLARED_URL" -o /usr/local/bin/cloudflared; then
        chmod +x /usr/local/bin/cloudflared
        print_success "cloudflared installed from GitHub releases"
    else
        print_error "Failed to download cloudflared from GitHub"
        exit 1
    fi
fi

# Verify installation
CLOUDFLARED_VERSION=$(cloudflared --version 2>&1 | head -n1 || echo "unknown")
print_success "Installed version: $CLOUDFLARED_VERSION"

# Step 2: Create configuration directory
print_step "Creating configuration directory"
mkdir -p "$CLOUDFLARED_CONFIG_DIR"
chmod 755 "$CLOUDFLARED_CONFIG_DIR"
print_success "Configuration directory created: $CLOUDFLARED_CONFIG_DIR"

# Step 3: Create tunnel credentials file
print_step "Configuring tunnel credentials"

# Store the token securely
echo "$TUNNEL_TOKEN" > "${CLOUDFLARED_CONFIG_DIR}/tunnel_token"
chmod 600 "${CLOUDFLARED_CONFIG_DIR}/tunnel_token"
print_success "Tunnel token stored securely"

# Step 4: Create systemd service
print_step "Creating systemd service"

cat > /etc/systemd/system/cloudflared.service << SERVICEEOF
[Unit]
Description=Cloudflare Tunnel
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/cloudflared tunnel --no-autoupdate run --token ${TUNNEL_TOKEN}
Restart=always
RestartSec=10s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=cloudflared

# Resource limits
MemoryLimit=128M
CPUQuota=25%

[Install]
WantedBy=multi-user.target
SERVICEEOF

chmod 644 /etc/systemd/system/cloudflared.service
print_success "Systemd service created"

# Step 5: Reload systemd and enable service
print_step "Enabling cloudflared service"
systemctl daemon-reload
systemctl enable cloudflared.service
print_success "Service enabled for auto-start on boot"

# Step 6: Start the service
print_step "Starting cloudflared service"
if systemctl start cloudflared.service; then
    print_success "cloudflared service started successfully"
else
    print_error "Failed to start cloudflared service"
    echo "Check logs with: journalctl -u cloudflared.service -n 50"
    exit 1
fi

# Step 7: Wait a moment and verify service is running
sleep 3
if systemctl is-active --quiet cloudflared.service; then
    print_success "Service is running"
else
    print_warning "Service may not be running correctly"
    echo "Check status with: systemctl status cloudflared.service"
fi

# Step 8: Update sudoers for passwordless management
print_step "Configuring passwordless systemctl access for ec2-user"
SUDOERS_FILE="/etc/sudoers.d/cloudflared"

if [ ! -f "$SUDOERS_FILE" ]; then
    cat > "$SUDOERS_FILE" << 'SUDOEOF'
# Allow ec2-user to manage cloudflared service without password
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl start cloudflared.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop cloudflared.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart cloudflared.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl status cloudflared.service
ec2-user ALL=(ALL) NOPASSWD: /usr/bin/systemctl is-active cloudflared.service
SUDOEOF
    chmod 0440 "$SUDOERS_FILE"
    print_success "Sudoers configured - ec2-user can now manage cloudflared without password"
else
    print_warning "Sudoers file already exists - skipping"
fi

# Final summary
echo -e "\n${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════╗
║              Setup Complete!                          ║
╚═══════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

print_success "Cloudflare Tunnel is now active"
print_success "Your API is being routed through Cloudflare"

echo -e "\n${YELLOW}NEXT STEPS:${NC}"
echo "1. Go to Cloudflare Zero Trust Dashboard"
echo "   https://one.dash.cloudflare.com/"
echo ""
echo "2. Navigate to Networks -> Tunnels"
echo "   Find your tunnel: $TUNNEL_NAME"
echo ""
echo "3. Configure Public Hostname (if not already done):"
echo "   - Subdomain: your-api (or any name you want)"
echo "   - Domain: your-domain.com"
echo "   - Service Type: HTTP"
echo "   - URL: localhost:${API_PORT}"
echo ""
echo "4. Your API will be accessible at:"
echo "   https://your-api.your-domain.com"
echo ""

echo -e "${BLUE}USEFUL COMMANDS:${NC}"
echo "  Check service status:     systemctl status cloudflared.service"
echo "  View logs:                journalctl -u cloudflared.service -f"
echo "  Restart service:          systemctl restart cloudflared.service"
echo "  Stop service:             systemctl stop cloudflared.service"
echo "  Start service:            systemctl start cloudflared.service"
echo ""

echo -e "${YELLOW}SECURITY NOTES:${NC}"
echo "- Your tunnel token is stored at: ${CLOUDFLARED_CONFIG_DIR}/tunnel_token"
echo "- Keep this token secure (permissions set to 600)"
echo "- Your EC2 instance is NOT directly exposed to the internet"
echo "- All traffic is routed through Cloudflare's network"
echo "- Cloudflare provides DDoS protection and WAF capabilities"
echo ""

echo -e "${YELLOW}TROUBLESHOOTING:${NC}"
echo "If the tunnel is not working:"
echo "1. Check service status: systemctl status cloudflared.service"
echo "2. View recent logs: journalctl -u cloudflared.service -n 50"
echo "3. Verify API is running: curl http://localhost:${API_PORT}/health"
echo "4. Check Cloudflare Dashboard for tunnel status"
echo ""

print_success "Installation complete!"
