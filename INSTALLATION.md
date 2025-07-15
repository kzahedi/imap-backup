# IMAP Backup - Automatic Installation Guide

This guide will help you set up automatic weekly email backups on your Mac with wake scheduling.

## 🚀 Quick Install

1. **Build the binary:**
   ```bash
   go build -o imap-backup .
   ```

2. **Run the installer:**
   ```bash
   ./install.sh
   ```

3. **Follow the prompts** and grant necessary permissions when asked.

## 📋 What the Installer Does

### System Setup
- ✅ Installs `imap-backup` binary to `/usr/local/bin`
- ✅ Creates backup directory: `~/Documents/email_backups`
- ✅ Creates log directory: `~/Library/Logs/imap-backup`
- ✅ Creates script directory: `~/.imap-backup`

### Automation Setup
- 🔄 **Cron Job**: Runs backup every Sunday at 2:00 AM
- ⏰ **Wake Schedule**: Mac wakes daily at 1:55 AM (5 min buffer)
- 📝 **Logging**: Automatic logging with rotation (30-day retention)
- 💬 **Notifications**: macOS notifications for success/failure

### Files Created
```
/usr/local/bin/imap-backup              # Main binary
~/.imap-backup/weekly-backup.sh         # Backup script  
~/.imap-backup/wake-for-backup.sh       # Wake script
~/.imap-backup/uninstall.sh            # Uninstaller
~/Documents/email_backups/             # Backup storage
~/Library/Logs/imap-backup/            # Log files
```

## ⚙️ Configuration Required

### 1. Setup Email Accounts

After installation, configure your email accounts:

```bash
# Interactive setup
imap-backup setup

# Or add accounts individually  
imap-backup account add --name "Gmail" --username your@gmail.com
imap-backup account add --name "Work" --username work@example.com
```

### 2. Grant System Permissions

**Required for automated backups:**

1. **Open System Preferences** → **Security & Privacy** → **Privacy**
2. **Select "Full Disk Access"** from the left sidebar
3. **Click the lock** and enter your password
4. **Add these applications:**
   - `cron` (allows scheduled execution)
   - `Terminal` (if running manually)
5. **Ensure they are checked/enabled**

### 3. Test Manual Backup

Before relying on automation, test manually:

```bash
# Test backup with verbose output
imap-backup backup -v

# Check logs
tail -f ~/Library/Logs/imap-backup/backup_*.log
```

## 📅 Backup Schedule

| Time | Action | Purpose |
|------|--------|---------|
| **1:55 AM Daily** | Mac wakes up | Ensures system is ready |
| **2:00 AM Sunday** | Backup runs | Weekly email backup |
| **2:XX AM Sunday** | Notification sent | Success/failure alert |

## 🔧 Management Commands

### View Current Schedule
```bash
# View cron jobs
crontab -l

# View power schedule  
pmset -g sched

# View recent logs
ls -la ~/Library/Logs/imap-backup/
```

### Manual Operations
```bash
# Run backup manually
~/.imap-backup/weekly-backup.sh

# Test wake script
~/.imap-backup/wake-for-backup.sh

# Check account status
imap-backup accounts -v
```

### Modify Schedule
```bash
# Edit cron schedule
crontab -e

# Example: Change to daily at 3:00 AM
# 0 3 * * * ~/.imap-backup/wake-for-backup.sh

# Update wake schedule
sudo pmset repeat wakeorpoweron MTWRFSU 02:55:00
```

## 📊 Monitoring

### Check Backup Status
```bash
# View latest log
tail -50 ~/Library/Logs/imap-backup/backup_$(date +%Y-%m-%d)*.log

# Check backup size
du -sh ~/Documents/email_backups/

# Count backed up emails
find ~/Documents/email_backups -name "*.eml" | wc -l
```

### Notifications

The system sends macOS notifications for:
- ✅ **Successful backups** (with account count)
- ❌ **Failed backups** (with error indicator)
- 📋 **System status** (startup/completion)

## 🛠 Troubleshooting

### Common Issues

1. **Permission Denied**
   ```bash
   # Solution: Grant Full Disk Access to cron
   # System Preferences → Security & Privacy → Privacy → Full Disk Access
   ```

2. **Mac Doesn't Wake**
   ```bash
   # Check power schedule
   pmset -g sched
   
   # Ensure Mac is plugged in or has sufficient battery
   # Check Energy Saver settings
   ```

3. **Backup Fails**
   ```bash
   # Check logs for details
   tail -100 ~/Library/Logs/imap-backup/backup_*.log
   
   # Test manually
   imap-backup backup -v
   
   # Verify account configuration
   imap-backup accounts
   ```

4. **No Notifications**
   ```bash
   # Test notification system
   osascript -e 'display notification "Test" with title "Test"'
   
   # Grant notification permissions if prompted
   ```

### Log Analysis
```bash
# View all backup attempts
grep "Starting weekly" ~/Library/Logs/imap-backup/*.log

# Check for errors
grep -i "error\|failed" ~/Library/Logs/imap-backup/*.log

# View successful completions
grep "completed successfully" ~/Library/Logs/imap-backup/*.log
```

## 🔄 Backup Retention

### Automatic Cleanup
- **Log files**: Kept for 30 days, then auto-deleted
- **Backup files**: Never deleted automatically (you control this)

### Manual Cleanup
```bash
# Remove old backups (example: older than 90 days)
find ~/Documents/email_backups -type f -mtime +90 -delete

# Archive old backups
tar -czf ~/Documents/email_backups-archive-$(date +%Y%m%d).tar.gz ~/Documents/email_backups/
```

## 🗑 Uninstallation

To completely remove the automatic backup system:

```bash
# Run the uninstaller
~/.imap-backup/uninstall.sh

# This removes:
# - Cron jobs
# - Power wake schedule  
# - Binary and scripts
# - But preserves backup files and logs
```

## 💡 Advanced Configuration

### Custom Backup Schedule
```bash
# Edit cron for different schedule
crontab -e

# Examples:
# Daily at 3 AM:     0 3 * * * ~/.imap-backup/wake-for-backup.sh
# Weekdays at 6 AM:  0 6 * * 1-5 ~/.imap-backup/wake-for-backup.sh  
# Twice weekly:      0 2 * * 0,3 ~/.imap-backup/wake-for-backup.sh
```

### Multiple Backup Locations
```bash
# Edit the backup script to use multiple destinations
nano ~/.imap-backup/weekly-backup.sh

# Add additional backup commands:
# imap-backup backup -o ~/Dropbox/email-backups
# imap-backup backup -o /Volumes/ExternalDrive/backups
```

### Email Notifications
```bash
# Install mail command and configure for email alerts
# Add to backup script:
# echo "Backup completed" | mail -s "Email Backup Status" admin@example.com
```

## ⚠️ Important Notes

- **Power Requirements**: Ensure your Mac is plugged in or has sufficient battery for scheduled wake times
- **Network Access**: Backup requires internet connectivity to reach email servers
- **Disk Space**: Monitor available space in the backup directory
- **Account Limits**: Some providers may have API rate limits for frequent access
- **Security**: Backup files contain your emails - ensure proper disk encryption

## 📞 Support

If you encounter issues:

1. **Check the logs** first: `~/Library/Logs/imap-backup/`
2. **Test manually** to isolate the problem
3. **Verify permissions** in System Preferences
4. **Check network connectivity** during backup times
5. **Review account configurations** with `imap-backup accounts`

The automated backup system is designed to be robust and self-monitoring, with comprehensive logging to help diagnose any issues.