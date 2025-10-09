# Quick Reference Card

## Initial Setup (One-Time)
```bash
# On EC2 instance
curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/setup-ec2.sh -o setup-ec2.sh
chmod +x setup-ec2.sh
sudo ./setup-ec2.sh

# Edit .env with your credentials
sudo nano /opt/english-learning-bot/.env

# Restart services
sudo systemctl restart english-learning-api.service english-learning-bot.service
```

## Daily Operations

### Check Status
```bash
# Quick check
sudo systemctl status english-learning-api.service english-learning-bot.service

# Is it running?
sudo systemctl is-active english-learning-api.service
```

### View Logs
```bash
# Follow live logs
sudo journalctl -u english-learning-api.service -f
sudo journalctl -u english-learning-bot.service -f

# Last 50 lines
sudo journalctl -u english-learning-api.service -n 50
```

### Control Services
```bash
# Restart
sudo systemctl restart english-learning-api.service
sudo systemctl restart english-learning-bot.service

# Stop
sudo systemctl stop english-learning-api.service
sudo systemctl stop english-learning-bot.service

# Start
sudo systemctl start english-learning-api.service
sudo systemctl start english-learning-bot.service
```

### Deployment
```bash
# Manual deployment (if you can't wait for hourly cron)
sudo /opt/english-learning-bot/deploy.sh

# Check current version
cat /opt/english-learning-bot/current_version

# View deployment log
tail -f /opt/english-learning-bot/deployment.log
```

## Troubleshooting

### Service won't start
```bash
# Check detailed status
sudo systemctl status english-learning-api.service

# Check recent logs
sudo journalctl -u english-learning-api.service -n 100

# Verify .env file exists and has correct permissions
ls -l /opt/english-learning-bot/.env
sudo cat /opt/english-learning-bot/.env
```

### Check deployment health
```bash
# View deployment log
tail -100 /opt/english-learning-bot/deployment.log

# Check for errors
grep ERROR /opt/english-learning-bot/deployment.log

# Verify cron job
crontab -l | grep deploy
```

### Rollback to previous version
```bash
# List available backups
ls -lh /opt/english-learning-bot/backups/

# Stop services
sudo systemctl stop english-learning-api.service english-learning-bot.service

# Restore from backup (replace timestamp)
sudo cp /opt/english-learning-bot/backups/backup-TIMESTAMP/english-learning-api /opt/english-learning-bot/bin/
sudo cp /opt/english-learning-bot/backups/backup-TIMESTAMP/english-learning-bot /opt/english-learning-bot/bin/
sudo cp /opt/english-learning-bot/backups/backup-TIMESTAMP/current_version /opt/english-learning-bot/

# Start services
sudo systemctl start english-learning-api.service english-learning-bot.service
```

## File Locations
- **Installation**: `/opt/english-learning-bot/`
- **Binaries**: `/opt/english-learning-bot/bin/`
- **Config**: `/opt/english-learning-bot/.env`
- **Database**: `/opt/english-learning-bot/data/`
- **Logs**: `journalctl -u SERVICE_NAME`
- **Deployment log**: `/opt/english-learning-bot/deployment.log`
- **Backups**: `/opt/english-learning-bot/backups/`
- **Service files**: `/etc/systemd/system/english-learning-*.service`

## Useful One-Liners
```bash
# Test API health
curl http://localhost:8080/health

# Check memory usage
ps aux | grep english-learning

# See when services last restarted
systemctl show english-learning-api.service | grep ActiveEnterTimestamp

# Count errors in logs (last hour)
sudo journalctl -u english-learning-api.service --since "1 hour ago" | grep -i error | wc -l

# See latest GitHub release
curl -s https://api.github.com/repos/Roma7-7-7/english-learning-bot/releases/latest | grep tag_name
```
