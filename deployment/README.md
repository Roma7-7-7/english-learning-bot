# Deployment Guide

This directory contains all the necessary files and documentation for deploying the English Learning Bot to AWS EC2 (Amazon Linux 2).

## Overview

The deployment system is designed to be:
- **Automated**: GitHub Actions builds binaries, EC2 polls for updates hourly
- **Resource-efficient**: No build tools needed on EC2 (no Go, no Node.js)
- **Zero-auth**: Uses public GitHub releases (no SSH keys, no secrets)
- **Documented**: Everything is version-controlled and reproducible

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  Developer pushes to main branch                    │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│  GitHub Actions (Automated CI/CD)                   │
│  1. Runs tests (go test, go vet)                    │
│  2. Builds binaries for Linux (CGO_ENABLED=0)       │
│  3. Creates GitHub Release with binaries as assets  │
└────────────────────────┬────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────┐
│  EC2 Instance (Amazon Linux 2)                     │
│  - Cron job checks for new releases every hour     │
│  - Downloads new binaries if version changed       │
│  - Restarts systemd services with new binaries     │
│  - No build tools installed (just curl + systemd)  │
└────────────────────────────────────────────────────┘
```

## Files in This Directory

### 1. `setup-ec2.sh`
**Purpose**: Initial setup script for a fresh EC2 instance.

**What it does**:
- Creates `/opt/english-learning-bot` directory structure
- Downloads deployment scripts and systemd service files from GitHub
- Creates `.env` file template
- Installs and enables systemd services
- Sets up hourly cron job for automatic deployment
- Runs initial deployment

**When to use**: Once, when setting up a new EC2 instance.

### 2. `deploy.sh`
**Purpose**: Deployment script that checks for and applies updates.

**What it does**:
- Fetches latest release version from GitHub API
- Compares with current version
- If different:
  - Downloads new binaries via curl (no authentication needed)
  - Backs up old binaries
  - Stops services → installs new binaries → starts services
- If same: exits without doing anything

**When to use**:
- Automatically via cron (every hour)
- Manually when you want to force an update: `sudo /opt/english-learning-bot/deploy.sh`

### 3. `systemd/english-learning-api.service`
**Purpose**: Systemd service definition for the API server.

**Key features**:
- Automatic restart on crash
- Runs as `ec2-user` (not root)
- Reads configuration from `.env` file
- Logs to journald
- Memory limit: 256MB, CPU limit: 50%

### 4. `systemd/english-learning-bot.service`
**Purpose**: Systemd service definition for the Telegram bot.

**Key features**: Same as API service (restart, user, env, logs, limits).

## Setting Up a New EC2 Instance

### Prerequisites
- AWS EC2 instance running **Amazon Linux 2** (free tier eligible)
- Security group allows inbound on port 8080 (API) and 443 (Telegram bot webhook)
- SSH access to the instance

### Step-by-Step Setup

#### 1. SSH into your EC2 instance
```bash
ssh -i your-key.pem ec2-user@your-ec2-ip
```

#### 2. Download and run the setup script
```bash
curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/setup-ec2.sh -o setup-ec2.sh
chmod +x setup-ec2.sh
sudo ./setup-ec2.sh
```

This will:
- Install everything needed
- Create directory structure
- Set up systemd services
- Configure automatic deployment
- Run the first deployment

#### 3. Configure your environment
Edit the `.env` file with your actual credentials:
```bash
sudo nano /opt/english-learning-bot/.env
```

Required values:
```bash
BOT_TOKEN=your_telegram_bot_token_here
BOT_ALLOWED_CHAT_IDS=your_chat_id_here
API_JWT_SECRET=generate_a_random_secret_here
```

Optional values (adjust as needed):
- `SCHEDULE_INTERVAL` - how often to send word checks (default: 4h)
- `SCHEDULE_START_HOUR` / `SCHEDULE_END_HOUR` - active hours (default: 9-22)
- `SCHEDULE_TIMEZONE` - your timezone (default: America/New_York)

#### 4. Restart services with new configuration
```bash
sudo systemctl restart english-learning-api.service
sudo systemctl restart english-learning-bot.service
```

#### 5. Verify everything is running
```bash
sudo systemctl status english-learning-api.service
sudo systemctl status english-learning-bot.service
```

Both should show `active (running)` in green.

### Done! 🎉
From now on, whenever you push to `main` branch:
1. GitHub Actions will build and release new binaries
2. Within an hour, EC2 will detect the new version
3. Services will automatically restart with the new version

## Useful Commands

### Service Management
```bash
# Check service status
sudo systemctl status english-learning-api.service
sudo systemctl status english-learning-bot.service

# Start/stop/restart services
sudo systemctl start english-learning-api.service
sudo systemctl stop english-learning-api.service
sudo systemctl restart english-learning-api.service

# Enable/disable auto-start on boot
sudo systemctl enable english-learning-api.service
sudo systemctl disable english-learning-api.service
```

### Viewing Logs
```bash
# Follow live logs
sudo journalctl -u english-learning-api.service -f
sudo journalctl -u english-learning-bot.service -f

# View last 100 lines
sudo journalctl -u english-learning-api.service -n 100

# View logs from last hour
sudo journalctl -u english-learning-api.service --since "1 hour ago"

# View deployment logs
tail -f /opt/english-learning-bot/deployment.log
```

### Manual Deployment
```bash
# Force check and deploy latest version
sudo /opt/english-learning-bot/deploy.sh

# View current version
cat /opt/english-learning-bot/current_version
```

### Troubleshooting
```bash
# Check if cron job is set up
crontab -l | grep deploy

# Test API endpoint
curl http://localhost:8080/health

# Check what releases are available
curl -s https://api.github.com/repos/Roma7-7-7/english-learning-bot/releases/latest | grep tag_name
```

## How Automatic Deployment Works

### 1. GitHub Actions Workflow
Located at `.github/workflows/release.yml`:
- **Triggers**: On push to `main` branch, if Go code changes
- **Steps**:
  1. Runs tests (`go test ./...`)
  2. Runs `go vet`
  3. Builds both binaries with optimizations (CGO_ENABLED=0, stripped symbols)
  4. Creates release with tag format: `vYYYYMMDD-HHMMSS-<commit-sha>`
  5. Uploads binaries as release assets

### 2. EC2 Cron Job
Runs every hour (at minute 0):
```bash
0 * * * * /opt/english-learning-bot/deploy.sh >> /opt/english-learning-bot/deployment.log 2>&1
```

### 3. Deployment Script Logic
```
Check GitHub for latest release tag
  ↓
Compare with current_version file
  ↓
If different:
  - Download binaries (curl, no auth needed for public repo)
  - Stop systemd services
  - Backup old binaries
  - Install new binaries
  - Start systemd services
  - Update current_version file
Else:
  - Exit (no action needed)
```

### 4. Smart Restart Policy
Services only restart if there's a new version. This means:
- No unnecessary downtime
- Logs clearly show when deployments happen
- Easy to audit deployment history

## Resource Usage on EC2

### What's NOT needed on EC2:
- ❌ Go compiler/SDK
- ❌ Node.js/npm
- ❌ Git (only needed for initial setup script download)
- ❌ Docker daemon

### What IS needed on EC2:
- ✅ curl (pre-installed on Amazon Linux 2)
- ✅ systemd (built into Amazon Linux 2)
- ✅ cron (built into Amazon Linux 2)

### Estimated Resource Usage:
- **API service**: ~30-50MB RAM (idle), up to 100MB under load
- **Bot service**: ~30-50MB RAM (idle), up to 80MB under load
- **Total**: ~60-100MB RAM, easily fits in 1GB free tier
- **No build overhead**: Saves 200-400MB during deployments

## Rollback Procedure

If a deployment breaks something:

1. **Check backups**:
   ```bash
   ls -lh /opt/english-learning-bot/backups/
   ```

2. **Stop services**:
   ```bash
   sudo systemctl stop english-learning-api.service
   sudo systemctl stop english-learning-bot.service
   ```

3. **Restore previous binaries**:
   ```bash
   sudo cp /opt/english-learning-bot/backups/backup-<timestamp>/english-learning-api /opt/english-learning-bot/bin/
   sudo cp /opt/english-learning-bot/backups/backup-<timestamp>/english-learning-bot /opt/english-learning-bot/bin/
   ```

4. **Update version file** (to prevent auto-deployment from overwriting):
   ```bash
   sudo cp /opt/english-learning-bot/backups/backup-<timestamp>/current_version /opt/english-learning-bot/
   ```

5. **Start services**:
   ```bash
   sudo systemctl start english-learning-api.service
   sudo systemctl start english-learning-bot.service
   ```

Backups are kept for the last 5 deployments automatically.

## Security Considerations

### Why This Setup is Secure:
1. **No secrets in GitHub**: EC2 doesn't need any credentials to pull from public releases
2. **No SSH from GitHub**: No need to store SSH keys in GitHub Actions
3. **Runs as non-root**: Services run as `ec2-user`, not root
4. **Resource limits**: systemd enforces memory/CPU limits to prevent abuse
5. **Read-only binaries**: Once deployed, binaries are not modified

### What You Should Secure:
1. **`.env` file permissions**: Should be `600` (read/write by owner only)
   ```bash
   sudo chmod 600 /opt/english-learning-bot/.env
   ```
2. **JWT secret**: Use a strong random string (32+ characters)
3. **Telegram bot token**: Keep it secret, never commit to git
4. **EC2 security groups**: Only allow necessary ports (22 for SSH, 8080 for API)

## Monitoring and Alerts

### Check Deployment History
```bash
# View deployment log
tail -100 /opt/english-learning-bot/deployment.log

# Find successful deployments
grep "Deployment successful" /opt/english-learning-bot/deployment.log

# Find failed deployments
grep "WARNING" /opt/english-learning-bot/deployment.log
```

### Monitor Service Health
```bash
# Quick status check
sudo systemctl is-active english-learning-api.service
sudo systemctl is-active english-learning-bot.service

# Detailed status
sudo systemctl status english-learning-api.service english-learning-bot.service
```

### Optional: Set Up CloudWatch
For production monitoring, consider setting up AWS CloudWatch:
- Monitor EC2 CPU/memory usage
- Collect systemd logs
- Set up alerts for service failures

## FAQ

### Q: How do I deploy to a different branch?
A: Edit `.github/workflows/release.yml` and change `branches: [main]` to your desired branch.

### Q: How do I change the deployment check interval?
A: Edit the cron job:
```bash
crontab -e
# Change: 0 * * * * (every hour)
# To: */30 * * * * (every 30 minutes)
```

### Q: What if I want to skip a deployment?
A: GitHub Actions only runs when Go code changes. If you push only documentation changes, no release will be created.

### Q: Can I use this with PostgreSQL instead of SQLite?
A: Yes! Just change the `.env` file:
```bash
DB_TYPE=postgres
DB_HOST=your-rds-endpoint
DB_PORT=5432
DB_NAME=english_learning
DB_USER=your_user
DB_PASSWORD=your_password
```

### Q: How do I update the deployment scripts themselves?
A: The scripts are version-controlled in GitHub. To update on EC2:
```bash
# Re-download deployment script
sudo curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/deploy.sh -o /opt/english-learning-bot/deploy.sh
sudo chmod +x /opt/english-learning-bot/deploy.sh

# Re-download systemd service files
sudo curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/systemd/english-learning-api.service -o /etc/systemd/system/english-learning-api.service
sudo curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/systemd/english-learning-bot.service -o /etc/systemd/system/english-learning-bot.service
sudo systemctl daemon-reload
```

## Support

If you encounter issues:
1. Check the logs (see "Viewing Logs" section above)
2. Verify `.env` file has correct values
3. Ensure security groups allow necessary ports
4. Check GitHub Actions workflow run status
5. Review deployment log for errors

For more information, see the main project README or check the codebase documentation in `CLAUDE.md`.
