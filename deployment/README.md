# Deployment Guide

This directory contains all the necessary files and documentation for deploying the English Learning Bot to AWS EC2 (Amazon Linux 2).

## Overview

The deployment system is designed to be:
- **Manual and controlled**: GitHub Actions builds binaries, you manually deploy when ready
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
│  - You manually SSH in and run deploy.sh           │
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
- Optionally sets up daily S3 database backups (in ec2-user's crontab)
- Runs initial deployment

**When to use**: Once, when setting up a new EC2 instance.

### 1.5. `setup-cloudflared.sh`
**Purpose**: Automated setup script for Cloudflare Tunnel (cloudflared).

**What it does**:
- Adds official Cloudflare yum repository
- Installs cloudflared via yum (with GitHub fallback)
- Creates systemd service for cloudflared
- Configures tunnel with your token
- Enables auto-start on boot
- Sets up passwordless systemctl access for ec2-user

**When to use**: After initial EC2 setup, when you want to expose your API through Cloudflare Tunnel without exposing your EC2 instance directly.

**Usage**:
```bash
sudo ./setup-cloudflared.sh --token YOUR_TUNNEL_TOKEN
```

See "Setting Up Cloudflare Tunnel" section below for detailed instructions.

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
- Manually when you want to deploy the latest release: `/opt/english-learning-bot/deploy.sh`
- No sudo needed - sudoers is configured for passwordless systemctl access

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

### 5. Sudoers Configuration
The setup script creates `/etc/sudoers.d/english-learning-bot` to allow `ec2-user` to manage services without a password.

**Allowed commands** (passwordless):
- `systemctl start/stop/restart/status` for both services
- `systemctl is-active` for both services

This enables the deploy script to run without sudo while still managing systemd services.

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
2. You can manually deploy by running: `/opt/english-learning-bot/deploy.sh` (no sudo needed)
3. Services will restart with the new version

## Setting Up Cloudflare Tunnel

After completing the EC2 setup, you can optionally configure Cloudflare Tunnel to expose your API securely without opening ports on your EC2 instance. This provides:

- **Security**: No direct exposure of your EC2 instance
- **DDoS Protection**: Cloudflare's network protects your API
- **SSL/TLS**: Automatic HTTPS with Cloudflare certificates
- **Access Control**: Built-in authentication and authorization options
- **Analytics**: Traffic monitoring through Cloudflare Dashboard

### Prerequisites

1. **Cloudflare Account**: Free account at https://cloudflare.com
2. **Domain**: A domain managed by Cloudflare (or add one to Cloudflare)
3. **EC2 Setup**: Completed the EC2 setup steps above
4. **API Running**: Verify API is running with `systemctl status english-learning-api.service`

### Step-by-Step Cloudflare Tunnel Setup

#### 1. Create a Cloudflare Tunnel

Go to Cloudflare Zero Trust Dashboard:
1. Visit https://one.dash.cloudflare.com/
2. Navigate to **Networks** → **Tunnels**
3. Click **Create a tunnel**
4. Select **Cloudflared** as the tunnel type
5. Name your tunnel (e.g., `english-learning-bot`)
6. Click **Save tunnel**

#### 2. Get Your Tunnel Token

After creating the tunnel:
1. In the tunnel setup page, you'll see installation instructions
2. Look for the command that starts with `cloudflared service install`
3. Copy the **token** from that command (it's a long JWT string)
4. The token looks like: `1234567890987654321...`

#### 3. Configure Public Hostname (in Cloudflare Dashboard)

Before running the setup script, configure the public hostname:
1. In the tunnel configuration, go to **Public Hostname** tab
2. Click **Add a public hostname**
3. Configure:
   - **Subdomain**: `api` (or any name you prefer)
   - **Domain**: Select your domain from the dropdown
   - **Service Type**: `HTTP`
   - **URL**: `localhost:8080`
4. Click **Save hostname**

Your API will be accessible at `https://api.your-domain.com`

#### 4. Run the Setup Script on EC2

SSH into your EC2 instance and run:

```bash
# Download the setup script
curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/setup-cloudflared.sh -o setup-cloudflared.sh
chmod +x setup-cloudflared.sh

# Run with your tunnel token
sudo ./setup-cloudflared.sh --token YOUR_TUNNEL_TOKEN
```

**Advanced usage**:
```bash
# Custom tunnel name and port
sudo ./setup-cloudflared.sh \
  --token YOUR_TOKEN \
  --tunnel-name my-custom-name \
  --api-port 8080
```

#### 5. Verify the Tunnel is Working

```bash
# Check cloudflared service status
systemctl status cloudflared.service

# View logs
journalctl -u cloudflared.service -f

# Test your API through Cloudflare
curl https://api.your-domain.com/health
```

### Cloudflare Tunnel Management

#### Service Commands

```bash
# Check status
systemctl status cloudflared.service

# View logs
journalctl -u cloudflared.service -f

# Restart tunnel
sudo systemctl restart cloudflared.service

# Stop tunnel
sudo systemctl stop cloudflared.service

# Start tunnel
sudo systemctl start cloudflared.service
```

#### Updating Tunnel Configuration

If you need to change the tunnel token or configuration:

1. **Update the token**:
   ```bash
   sudo nano /etc/cloudflared/tunnel_token
   ```

2. **Update the systemd service**:
   ```bash
   sudo nano /etc/systemd/system/cloudflared.service
   # Update the token in ExecStart line
   ```

3. **Reload and restart**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart cloudflared.service
   ```

#### Updating cloudflared

If cloudflared was installed via yum, you can update it using:

```bash
# Update cloudflared package
sudo yum update -y cloudflared

# Restart the service
sudo systemctl restart cloudflared.service
```

#### Removing Cloudflare Tunnel

If you want to remove cloudflared:

```bash
# Stop and disable service
sudo systemctl stop cloudflared.service
sudo systemctl disable cloudflared.service

# Remove service file
sudo rm /etc/systemd/system/cloudflared.service
sudo systemctl daemon-reload

# Remove package (if installed via yum)
sudo yum remove -y cloudflared

# Or remove binary (if installed from GitHub)
sudo rm /usr/local/bin/cloudflared

# Remove config and repository
sudo rm -rf /etc/cloudflared
sudo rm /etc/yum.repos.d/cloudflare.repo

# Remove sudoers file
sudo rm /etc/sudoers.d/cloudflared
```

### Security Considerations

#### EC2 Security Groups

With Cloudflare Tunnel, you can **remove** public access to port 8080 in your EC2 security group:

1. Go to AWS EC2 Console → Security Groups
2. Find your instance's security group
3. **Remove** the inbound rule for port 8080
4. Keep only SSH (port 22) for management

This ensures your API is **only** accessible through Cloudflare, not directly.

#### Cloudflare Access (Optional)

For additional security, you can add authentication:

1. In Cloudflare Dashboard, go to **Access** → **Applications**
2. Click **Add an application**
3. Select **Self-hosted**
4. Configure authentication (Google, GitHub, email OTP, etc.)
5. Apply the policy to your API subdomain

Now users must authenticate before accessing your API.

### Troubleshooting Cloudflare Tunnel

#### Tunnel Not Connecting

**Check service status**:
```bash
systemctl status cloudflared.service
journalctl -u cloudflared.service -n 50
```

**Common issues**:
- Invalid token: Verify token in `/etc/cloudflared/tunnel_token`
- Network issues: Check internet connectivity from EC2
- Cloudflare outage: Check https://www.cloudflarestatus.com/

#### API Not Accessible

**Check API is running locally**:
```bash
curl http://localhost:8080/health
```

**Check tunnel configuration in Cloudflare Dashboard**:
- Verify public hostname is configured
- Ensure service URL is `localhost:8080`
- Check tunnel status shows "HEALTHY"

**Check DNS propagation**:
```bash
dig api.your-domain.com
nslookup api.your-domain.com
```

#### Logs Show Connection Errors

**Check API port matches**:
- Verify API is listening on the correct port (default 8080)
- Check `.env` file: `cat /opt/english-learning-bot/.env | grep API_PORT`
- Ensure tunnel is configured for the same port

**Check firewall rules**:
```bash
# On Amazon Linux 2, firewalld should allow localhost connections
sudo systemctl status firewalld
```

### Performance and Monitoring

#### View Tunnel Analytics

In Cloudflare Dashboard:
1. Go to **Networks** → **Tunnels**
2. Click on your tunnel
3. View **Analytics** tab for:
   - Request count
   - Bandwidth usage
   - Response times
   - Error rates

#### Resource Usage

Cloudflared is lightweight:
- **Memory**: ~20-30MB
- **CPU**: <5% under normal load
- **Network**: Minimal overhead (compression enabled)

#### Rate Limiting

Configure rate limiting in Cloudflare:
1. Go to **Security** → **WAF**
2. Create custom rules for your API
3. Set rate limits per IP or globally

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
# Check and deploy latest version
/opt/english-learning-bot/deploy.sh

# View current version
cat /opt/english-learning-bot/current_version
```

### Troubleshooting
```bash
# Check if backup cron job is set up
crontab -l | grep backup

# Test API endpoint
curl http://localhost:8080/health

# Check what releases are available
curl -s https://api.github.com/repos/Roma7-7-7/english-learning-bot/releases/latest | grep tag_name

# Verify sudoers configuration
sudo cat /etc/sudoers.d/english-learning-bot
```

## How Deployment Works

### 1. GitHub Actions Workflow
Located at `.github/workflows/release.yml`:
- **Triggers**: On push to `main` branch, if Go code changes
- **Steps**:
  1. Runs tests (`go test ./...`)
  2. Runs `go vet`
  3. Builds both binaries with optimizations (CGO_ENABLED=0, stripped symbols)
  4. Creates release with tag format: `vYYYYMMDD-HHMMSS-<commit-sha>`
  5. Uploads binaries as release assets

### 2. Manual Deployment
When you're ready to deploy, SSH into EC2 and run:
```bash
/opt/english-learning-bot/deploy.sh
```

The script uses sudoers configuration for passwordless systemctl access, so no sudo is needed.

### 3. Deployment Script Logic
```
Check GitHub for latest release tag
  ↓
Compare with current_version file
  ↓
If different:
  - Download binaries (curl, no auth needed for public repo)
  - Stop systemd services (via sudo, configured for passwordless access)
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
- Full control over when updates are applied

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

## Database Backup to S3

The setup script can configure automatic daily database backups to S3.

### During Initial Setup

When running `setup-ec2.sh`, you'll be prompted:
```
Configure S3 backup (optional):
If you want automatic daily backups to S3, provide your S3 bucket path.
Example: s3://my-bucket/english-learning-bot/backups

Enter S3 bucket path (or press Enter to skip):
```

Enter your S3 path (e.g., `s3://bucket/english-learning-bot/db_dumps`) and the setup will:
- Download the backup script
- Create `.backup_config` with your S3 path
- Add a daily cron job (runs at 20:00 UTC)

### Manual Setup (After Initial Setup)

If you skipped S3 configuration during setup, you can add it later:

1. **Create backup config**:
   ```bash
   sudo tee /opt/english-learning-bot/.backup_config << EOF
   # S3 Backup Configuration
   S3_BUCKET_PATH="s3://your-bucket/path"
   EOF

   sudo chmod 600 /opt/english-learning-bot/.backup_config
   ```

2. **Download backup script** (if not already present):
   ```bash
   sudo curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/backup.sh \
     -o /opt/english-learning-bot/backup.sh
   sudo chmod +x /opt/english-learning-bot/backup.sh
   ```

3. **Add cron job**:
   ```bash
   (sudo crontab -l 2>/dev/null; echo "0 20 * * * /opt/english-learning-bot/backup.sh >> /opt/english-learning-bot/backup.log 2>&1") | sudo crontab -
   ```

### Backup Details

- **Schedule**: Daily at 20:00 UTC
- **Method**: SQLite `.backup` command (creates consistent snapshot)
- **Format**: `english_learning_backup_YYYY-MM-DDTHH:MM:SS.sqlite`
- **Storage**: Uploaded to S3, then local copy deleted
- **Logs**: Written to `/opt/english-learning-bot/backup.log`

### IAM Permissions Required

Your EC2 instance needs an IAM role with S3 permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:PutObjectAcl"
      ],
      "Resource": "arn:aws:s3:::your-bucket/english-learning-bot/db_dumps/*"
    }
  ]
}
```

Attach this policy to your EC2 instance's IAM role.

### Manual Backup

Run a backup manually at any time:
```bash
sudo /opt/english-learning-bot/backup.sh
```

### View Backup Logs

```bash
# Follow live
tail -f /opt/english-learning-bot/backup.log

# View all
cat /opt/english-learning-bot/backup.log

# Check last backup
tail -20 /opt/english-learning-bot/backup.log
```

### Restore from S3 Backup

```bash
# List available backups
aws s3 ls s3://your-bucket/english-learning-bot/db_dumps/

# Download specific backup
aws s3 cp s3://your-bucket/path/backup.sqlite /tmp/restore.sqlite

# Stop services
sudo systemctl stop english-learning-api.service english-learning-bot.service

# Restore database
sudo cp /tmp/restore.sqlite /opt/english-learning-bot/data/english_learning.db
sudo chown ec2-user:ec2-user /opt/english-learning-bot/data/english_learning.db

# Start services
sudo systemctl start english-learning-api.service english-learning-bot.service
```

### Troubleshooting Backups

**Error: "S3_BUCKET_PATH not set"**
- Check `/opt/english-learning-bot/.backup_config` exists and contains `S3_BUCKET_PATH`

**Error: "Failed to upload backup to S3"**
- Verify IAM role has S3 permissions
- Check S3 bucket exists: `aws s3 ls s3://your-bucket/`
- Test AWS CLI: `aws s3 ls`

**Error: "Database not found"**
- Verify database path in config matches actual location
- Default: `/opt/english-learning-bot/data/english_learning.db`

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

### Q: Do I need to use Cloudflare Tunnel?
A: No, Cloudflare Tunnel is optional. You can expose your API directly by:
- Opening port 8080 in EC2 security group
- Using an Elastic IP or domain pointing to your EC2
- Setting up your own reverse proxy (nginx, Apache)

However, Cloudflare Tunnel provides:
- Better security (no exposed ports)
- Free DDoS protection
- Automatic HTTPS
- Built-in access control options

### Q: Can I use Cloudflare Tunnel for the Telegram bot too?
A: The Telegram bot doesn't need Cloudflare Tunnel because it uses polling (outbound connections only) rather than webhooks. Only the API needs public access.

### Q: How do I deploy to a different branch?
A: Edit `.github/workflows/release.yml` and change `branches: [main]` to your desired branch.

### Q: How do I deploy automatically on a schedule?
A: You can add a cron job to ec2-user's crontab if desired:
```bash
crontab -e
# Add: 0 * * * * /opt/english-learning-bot/deploy.sh >> /opt/english-learning-bot/deployment.log 2>&1
# This would check every hour at minute 0
```

### Q: What if I want to skip a deployment?
A: GitHub Actions only runs when Go code changes. If you push only documentation changes, no release will be created.

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
