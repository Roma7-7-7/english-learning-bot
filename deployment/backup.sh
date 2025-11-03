#!/bin/bash
set -e

# This script can be run as ec2-user (no sudo needed)
# Configuration
INSTALL_DIR="/opt/english-learning-bot"
BACKUP_CONFIG="${INSTALL_DIR}/.backup_config"
DB_PATH="${INSTALL_DIR}/data/english_learning.db"
LOG_FILE="${INSTALL_DIR}/backup.log"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

# Check if config exists
if [ ! -f "$BACKUP_CONFIG" ]; then
    log_error "Backup config not found at $BACKUP_CONFIG"
    exit 1
fi

# Load S3 path from config
source "$BACKUP_CONFIG"

if [ -z "$S3_BUCKET_PATH" ]; then
    log_error "S3_BUCKET_PATH not set in $BACKUP_CONFIG"
    exit 1
fi

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    log_error "Database not found at $DB_PATH"
    exit 1
fi

log "Starting database backup..."

# Create timestamp
TIMESTAMP=$(date +%Y-%m-%dT%H:%M:%S)
BACKUP_FILE="/tmp/english_learning_backup_${TIMESTAMP}.sqlite"
S3_DESTINATION="${S3_BUCKET_PATH}/${TIMESTAMP}.sqlite"

# Backup database using sqlite3 .backup command
log "Creating backup at $BACKUP_FILE"
if ! sqlite3 "$DB_PATH" ".backup '$BACKUP_FILE'"; then
    log_error "Failed to create backup"
    rm -f "$BACKUP_FILE"
    exit 1
fi

# Verify backup file was created
if [ ! -f "$BACKUP_FILE" ]; then
    log_error "Backup file was not created"
    exit 1
fi

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log "Backup created successfully (size: $BACKUP_SIZE)"

# Upload to S3
log "Uploading to S3: $S3_DESTINATION"
if aws s3 cp "$BACKUP_FILE" "$S3_DESTINATION" 2>> "$LOG_FILE"; then
    log "Backup uploaded successfully to S3"
else
    log_error "Failed to upload backup to S3"
    rm -f "$BACKUP_FILE"
    exit 1
fi

# Clean up local backup file
rm -f "$BACKUP_FILE"
log "Backup completed successfully"

# Optional: Clean up old backups (keep last 30 days)
# Uncomment to enable automatic cleanup
# log "Cleaning up old backups (keeping last 30 days)..."
# CUTOFF_DATE=$(date -d '30 days ago' +%Y-%m-%d 2>/dev/null || date -v-30d +%Y-%m-%d)
# aws s3 ls "$S3_BUCKET_PATH/" | while read -r line; do
#     BACKUP_DATE=$(echo "$line" | awk '{print $1}')
#     if [[ "$BACKUP_DATE" < "$CUTOFF_DATE" ]]; then
#         BACKUP_FILE=$(echo "$line" | awk '{print $4}')
#         log "Deleting old backup: $BACKUP_FILE"
#         aws s3 rm "${S3_BUCKET_PATH}/${BACKUP_FILE}"
#     fi
# done

exit 0
