# Simple Deployment Guide

This guide covers the simplified deployment approach for English Learning Bot, designed for running on any Linux server (VPS, dedicated server, etc.) without AWS dependencies.

## Overview

**What this deployment includes:**
- Automated binary deployment from GitHub releases
- Single systemd service (runs both bot and API server) with auto-restart
- Simple `.env` file configuration
- Passwordless service management
- Manual backup strategy

**What this deployment does NOT include:**
- AWS SSM Parameter Store integration
- Automated backups to S3
- Cloud-specific configurations

Perfect for: Personal projects, small deployments, cost-conscious hosting (Hetzner, Contabo, OVH, etc.)

## Prerequisites

- Linux server with systemd (Ubuntu, Debian, CentOS, etc.)
- Root or sudo access
- Telegram bot token from [@BotFather](https://t.me/BotFather)
- Internet access (to download releases from GitHub)

## Quick Start

### 1. Download and run setup script

```bash
# Download setup script
curl -L -o setup-simple.sh https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/setup-simple.sh

# Make it executable
chmod +x setup-simple.sh

# Run with sudo
sudo ./setup-simple.sh
```

The script will:
1. Ask for the user to run the service as (default: current user)
2. Create directory structure at `/opt/english-learning-bot`
3. Download deployment script
4. Install the systemd service
5. Configure sudoers for passwordless service management
6. Prompt for configuration (bot token, HTTP settings, etc.)
7. Download and start the latest release

### 2. Verify installation

```bash
# Check service status
sudo systemctl status english-learning-bot.service

# View logs
sudo journalctl -u english-learning-bot.service -f
```

You're done! The service should now be running.

## Configuration

All configuration is stored in `/opt/english-learning-bot/.env`

### Environment Variables

```bash
# Telegram Configuration
BOT_TELEGRAM_TOKEN=your_bot_token_here
BOT_TELEGRAM_ALLOWED_CHAT_IDS=123456,789012  # Comma-separated Telegram chat IDs

# Database
BOT_DB_PATH=/opt/english-learning-bot/data/english_learning.db?cache=shared&mode=rwc

# HTTP / Authentication
BOT_HTTP_JWT_SECRET=your_random_secret_here
BOT_HTTP_CORS_ALLOW_ORIGINS=https://yourdomain.com
BOT_HTTP_COOKIE_DOMAIN=yourdomain.com
BOT_HTTP_JWT_AUDIENCE=web,mobile

# Optional: Uncomment and modify to override defaults
# BOT_SERVER_ADDR=:8080
# BOT_SCHEDULE_PUBLISH_INTERVAL=15m
# BOT_SCHEDULE_HOUR_FROM=9
# BOT_SCHEDULE_HOUR_TO=22
# BOT_SCHEDULE_LOCATION=Europe/Kyiv
# BOT_DEV=false
```

### Changing Configuration

1. Edit the environment file:
```bash
sudo nano /opt/english-learning-bot/.env
```

2. Restart the service:
```bash
sudo systemctl restart english-learning-bot.service
```

## Management Commands

All commands can be run by the service user without password (configured via sudoers).

### Service Control

```bash
# Start service
sudo systemctl start english-learning-bot.service

# Stop service
sudo systemctl stop english-learning-bot.service

# Restart service
sudo systemctl restart english-learning-bot.service

# Check status
sudo systemctl status english-learning-bot.service

# View logs (follow mode)
sudo journalctl -u english-learning-bot.service -f

# View last 100 lines of logs
sudo journalctl -u english-learning-bot.service -n 100
```

### Deployment

To update to the latest release:

```bash
/opt/english-learning-bot/deploy.sh
```

The deployment script:
- Fetches the latest release from GitHub
- Detects your server architecture (AMD64 or ARM64)
- Stops the service
- Backs up the current binary
- Installs the new binary
- Starts the service
- Keeps the last 5 binary backups

## Manual Backup Strategy

### From Your Local Machine

**Daily automated backup** (add to your local crontab):

```bash
# Add to crontab (crontab -e)
0 21 * * * scp user@your-server:/opt/english-learning-bot/data/english_learning.db ~/backups/english-learning/backup-$(date +\%Y\%m\%d).db
```

**One-time manual backup:**

```bash
scp user@your-server:/opt/english-learning-bot/data/english_learning.db ~/backups/
```

### From the Server

**Create a local backup:**

```bash
cp /opt/english-learning-bot/data/english_learning.db ~/english-learning-backup-$(date +%Y%m%d).db
```

**Schedule local backups** (add to crontab):

```bash
# Add to crontab (crontab -e)
0 2 * * * cp /opt/english-learning-bot/data/english_learning.db ~/backups/english-learning-$(date +\%Y\%m\%d).db
```

## Directory Structure

```
/opt/english-learning-bot/
├── bin/
│   ├── english-learning-bot      # Application binary (bot + API server)
│   └── VERSION                   # Version info
├── data/
│   └── english_learning.db       # SQLite database
├── backups/
│   ├── backup_20240315_120000/   # Binary backups (auto-created by deploy.sh)
│   └── english_learning.db.backup.*  # DB safety backups
├── .env                          # Environment configuration (IMPORTANT: contains secrets)
├── deploy.sh                     # Deployment script
├── current_version               # Current release version
└── deployment.log                # Deployment logs
```

## Troubleshooting

### Service won't start

1. Check logs:
```bash
sudo journalctl -u english-learning-bot.service -n 50
```

2. Common issues:
   - Missing required variables in `.env`
   - Invalid bot token
   - Database file permissions
   - Network connectivity issues
   - Port 8080 already in use

### Configuration errors

Make sure your `.env` file has all required variables:
```bash
cat /opt/english-learning-bot/.env
```

Required variables:
- `BOT_TELEGRAM_TOKEN`
- `BOT_TELEGRAM_ALLOWED_CHAT_IDS`
- `BOT_HTTP_JWT_SECRET`
- `BOT_HTTP_CORS_ALLOW_ORIGINS`
- `BOT_HTTP_COOKIE_DOMAIN`
- `BOT_HTTP_JWT_AUDIENCE`

### Bot not responding

1. Verify the service is running:
```bash
sudo systemctl status english-learning-bot.service
```

2. Check if the token is valid:
```bash
TOKEN=$(grep BOT_TELEGRAM_TOKEN /opt/english-learning-bot/.env | cut -d= -f2)
curl -s "https://api.telegram.org/bot${TOKEN}/getMe"
```

### API not accessible

1. Verify the service is running:
```bash
sudo systemctl status english-learning-bot.service
```

2. Check if port 8080 is listening:
```bash
sudo netstat -tlnp | grep 8080
```

3. Test API health endpoint:
```bash
curl http://localhost:8080/health
```

### Permission denied errors

Ensure all files are owned by the service user:
```bash
sudo chown -R your-user:your-user /opt/english-learning-bot
```

### Database corrupted

Restore from backup:
```bash
# Stop service
sudo systemctl stop english-learning-bot.service

# Restore from your backup
scp ~/backups/english_learning.db your-user@your-server:/opt/english-learning-bot/data/

# Start service
sudo systemctl start english-learning-bot.service
```

## Security Notes

### File Permissions

The `.env` file contains secrets and should be protected:

```bash
# Verify permissions (should be 600)
ls -l /opt/english-learning-bot/.env

# Fix if needed
sudo chmod 600 /opt/english-learning-bot/.env
sudo chown your-user:your-user /opt/english-learning-bot/.env
```

### Rotating Secrets

1. Generate new values for secrets
2. Update `.env` file
3. Restart the service

```bash
sudo nano /opt/english-learning-bot/.env
# Update BOT_TELEGRAM_TOKEN and BOT_HTTP_JWT_SECRET
sudo systemctl restart english-learning-bot.service
```

### Server Access

- Use SSH keys instead of passwords
- Disable root login
- Keep your server updated: `sudo apt update && sudo apt upgrade`
- Configure firewall to only expose necessary ports

## Migration from AWS EC2

If you're migrating from AWS EC2 to a simple deployment:

1. **Backup your database** from EC2:
```bash
scp ec2-user@ec2-host:/opt/english-learning-bot/data/english_learning.db ~/
```

2. **Run setup on new server** (follow Quick Start above)

3. **Stop service** on new server:
```bash
sudo systemctl stop english-learning-bot.service
```

4. **Copy database** to new server:
```bash
scp ~/english_learning.db user@new-server:/opt/english-learning-bot/data/
```

5. **Fix permissions**:
```bash
# On new server
sudo chown your-user:your-user /opt/english-learning-bot/data/english_learning.db
```

6. **Start service**:
```bash
sudo systemctl start english-learning-bot.service
```

All your data and user progress will be preserved!

## Re-running Setup

You can safely re-run `setup-simple.sh` on an existing installation:

- Existing database will be backed up automatically
- Configuration (`.env`) will NOT be overwritten
- Only binaries and scripts will be updated
- Services will be restarted with new versions

## Uninstallation

To completely remove English Learning Bot:

```bash
# Stop and disable service
sudo systemctl stop english-learning-bot.service
sudo systemctl disable english-learning-bot.service

# Remove systemd service
sudo rm /etc/systemd/system/english-learning-bot.service
sudo systemctl daemon-reload

# Remove sudoers configuration
sudo rm /etc/sudoers.d/english-learning-bot

# Optional: backup database first
cp /opt/english-learning-bot/data/english_learning.db ~/english-learning-final-backup.db

# Remove installation directory
sudo rm -rf /opt/english-learning-bot
```

## Cost Comparison

Example monthly costs for different hosting providers (as of 2024):

| Provider | Configuration | Cost/Month | Notes |
|----------|---------------|------------|-------|
| **Hetzner** | CPX11 (2 vCPU, 2GB RAM) | €4.51 (~$5) | Recommended |
| **Contabo** | VPS S (4 vCPU, 8GB RAM) | €5.99 (~$6.50) | More resources |
| **OVH** | VPS Starter (1 vCPU, 2GB RAM) | ~$7 | |
| **AWS EC2** | t3.micro (2 vCPU, 1GB RAM) | ~$10+ | Previous setup |

All configurations are more than enough for running the service. Savings: ~$5-15/month compared to AWS.

## Support

- Issues: https://github.com/Roma7-7-7/english-learning-bot/issues
- View this on GitHub: https://github.com/Roma7-7-7/english-learning-bot/tree/main/deployment

## Advanced: Custom Installation Directory

To install to a different directory, modify the script variables before running:

```bash
# Edit setup-simple.sh
INSTALL_DIR="/custom/path/english-learning-bot"

# Then run the modified script
sudo ./setup-simple.sh
```

Note: You'll also need to update the systemd service files manually.
