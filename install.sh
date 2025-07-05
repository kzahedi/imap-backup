#!/bin/bash

# IMAP Backup Automatic Installation Script
# This script installs imap-backup and sets up weekly automatic backups

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
BACKUP_DIR="$HOME/Documents/email_backups"
LOG_DIR="$HOME/Library/Logs/imap-backup"
SCRIPT_DIR="$HOME/.imap-backup"
BINARY_NAME="imap-backup"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Function to check if running on macOS
check_macos() {
    if [[ "$OSTYPE" != "darwin"* ]]; then
        print_error "This script is designed for macOS only."
        exit 1
    fi
}

# Function to check if binary exists
check_binary() {
    if [[ ! -f "./$BINARY_NAME" ]]; then
        print_error "Binary '$BINARY_NAME' not found in current directory."
        print_status "Please run 'go build -o $BINARY_NAME .' first"
        exit 1
    fi
}

# Function to install binary
install_binary() {
    print_header "Installing imap-backup binary..."
    
    # Create install directory if it doesn't exist
    sudo mkdir -p "$INSTALL_DIR"
    
    # Copy binary to install location
    sudo cp "./$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    print_status "Binary installed to $INSTALL_DIR/$BINARY_NAME"
}

# Function to create necessary directories
create_directories() {
    print_header "Creating necessary directories..."
    
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$SCRIPT_DIR"
    
    print_status "Created directories:"
    print_status "  Backups: $BACKUP_DIR"
    print_status "  Logs: $LOG_DIR"
    print_status "  Scripts: $SCRIPT_DIR"
}

# Function to create backup script
create_backup_script() {
    print_header "Creating backup script..."
    
    cat > "$SCRIPT_DIR/weekly-backup.sh" << 'EOF'
#!/bin/bash

# Weekly IMAP Backup Script
# This script runs the imap-backup tool and logs the results

# Configuration
BACKUP_DIR="$HOME/Documents/email_backups"
LOG_DIR="$HOME/Library/Logs/imap-backup"
BINARY="/usr/local/bin/imap-backup"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
LOG_FILE="$LOG_DIR/backup_$TIMESTAMP.log"

# Ensure directories exist
mkdir -p "$BACKUP_DIR"
mkdir -p "$LOG_DIR"

# Function to log with timestamp
log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to send notification
send_notification() {
    local title="$1"
    local message="$2"
    local sound="$3"
    
    osascript -e "display notification \"$message\" with title \"$title\" sound name \"$sound\""
}

# Start backup
log_message "Starting weekly IMAP backup..."
log_message "Backup directory: $BACKUP_DIR"

# Check if binary exists
if [[ ! -f "$BINARY" ]]; then
    log_message "ERROR: imap-backup binary not found at $BINARY"
    send_notification "IMAP Backup Failed" "Binary not found" "Basso"
    exit 1
fi

# Run the backup with verbose output
log_message "Running: $BINARY backup -o \"$BACKUP_DIR\" -v"

if "$BINARY" backup -o "$BACKUP_DIR" -v >> "$LOG_FILE" 2>&1; then
    log_message "Backup completed successfully"
    
    # Count backed up accounts
    ACCOUNT_COUNT=$(find "$BACKUP_DIR" -maxdepth 1 -type d | wc -l | tr -d ' ')
    ACCOUNT_COUNT=$((ACCOUNT_COUNT - 1)) # Subtract 1 for the backup dir itself
    
    send_notification "IMAP Backup Success" "Backed up $ACCOUNT_COUNT account(s)" "Glass"
else
    EXIT_CODE=$?
    log_message "Backup failed with exit code: $EXIT_CODE"
    send_notification "IMAP Backup Failed" "Check logs for details" "Basso"
    exit $EXIT_CODE
fi

# Clean up old log files (keep last 30 days)
find "$LOG_DIR" -name "backup_*.log" -type f -mtime +30 -delete

log_message "Weekly backup completed"
EOF

    chmod +x "$SCRIPT_DIR/weekly-backup.sh"
    print_status "Backup script created at $SCRIPT_DIR/weekly-backup.sh"
}

# Function to create wake script
create_wake_script() {
    print_header "Creating wake script..."
    
    cat > "$SCRIPT_DIR/wake-for-backup.sh" << 'EOF'
#!/bin/bash

# Wake for Backup Script
# This script wakes the Mac and runs the backup

# Wait a moment for the system to fully wake
sleep 30

# Set environment variables that might be needed
export PATH="/usr/local/bin:/usr/bin:/bin"
export HOME="/Users/$(whoami)"

# Run the backup script
exec "$HOME/.imap-backup/weekly-backup.sh"
EOF

    chmod +x "$SCRIPT_DIR/wake-for-backup.sh"
    print_status "Wake script created at $SCRIPT_DIR/wake-for-backup.sh"
}

# Function to setup cron job
setup_cron() {
    print_header "Setting up weekly cron job..."
    
    # Remove any existing cron job for imap-backup
    (crontab -l 2>/dev/null | grep -v "imap-backup\|weekly-backup") | crontab -
    
    # Add new cron job (runs every Sunday at 2:00 AM)
    (crontab -l 2>/dev/null; echo "0 2 * * 0 $SCRIPT_DIR/wake-for-backup.sh") | crontab -
    
    print_status "Cron job added: Weekly backup every Sunday at 2:00 AM"
    print_status "Current crontab:"
    crontab -l | grep -E "(imap-backup|weekly-backup|wake-for-backup)"
}

# Function to setup power management
setup_power_management() {
    print_header "Setting up power management for automatic wake..."
    
    # Schedule weekly wake (every Sunday at 1:55 AM)
    # This gives the system 5 minutes to fully wake before the backup starts
    sudo pmset repeat wakeorpoweron MTWRFSU 01:55:00
    
    print_status "Configured Mac to wake every day at 1:55 AM"
    print_status "The backup will run on Sundays at 2:00 AM"
    
    # Show current power management settings
    print_status "Current power management settings:"
    pmset -g sched
}

# Function to setup system permissions
setup_permissions() {
    print_header "Checking system permissions..."
    
    # Check if cron has Full Disk Access
    print_warning "IMPORTANT: You may need to grant Full Disk Access to cron/Terminal:"
    print_status "1. Go to System Preferences → Security & Privacy → Privacy"
    print_status "2. Select 'Full Disk Access' from the left sidebar"
    print_status "3. Click the lock to make changes"
    print_status "4. Add 'cron' and 'Terminal' if they're not already listed"
    print_status "5. Ensure they are checked/enabled"
    echo
    
    # Check if Terminal has permission to send notifications
    print_status "Testing notification system..."
    osascript -e 'display notification "Test notification from imap-backup installer" with title "Installation Test" sound name "Glass"' 2>/dev/null || {
        print_warning "Notifications may require permission. Grant access in System Preferences if prompted."
    }
}

# Function to test the installation
test_installation() {
    print_header "Testing installation..."
    
    # Test binary
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        print_status "✓ Binary is accessible in PATH"
        "$BINARY_NAME" --version 2>/dev/null || print_status "✓ Binary executes successfully"
    else
        print_error "✗ Binary not found in PATH"
        return 1
    fi
    
    # Test scripts
    if [[ -x "$SCRIPT_DIR/weekly-backup.sh" ]]; then
        print_status "✓ Backup script is executable"
    else
        print_error "✗ Backup script not found or not executable"
        return 1
    fi
    
    if [[ -x "$SCRIPT_DIR/wake-for-backup.sh" ]]; then
        print_status "✓ Wake script is executable"
    else
        print_error "✗ Wake script not found or not executable"
        return 1
    fi
    
    # Test cron job
    if crontab -l 2>/dev/null | grep -q "wake-for-backup.sh"; then
        print_status "✓ Cron job is installed"
    else
        print_error "✗ Cron job not found"
        return 1
    fi
    
    print_status "✓ All tests passed!"
}

# Function to show next steps
show_next_steps() {
    print_header "Installation completed successfully!"
    echo
    print_status "📋 NEXT STEPS:"
    echo
    print_status "1. 🔧 Configure your email accounts:"
    print_status "   Run: $BINARY_NAME setup"
    print_status "   Or:  $BINARY_NAME account add --name \"Gmail\" --username your@gmail.com"
    echo
    print_status "2. 🧪 Test manual backup:"
    print_status "   Run: $BINARY_NAME backup -v"
    echo
    print_status "3. 🔒 Grant necessary permissions:"
    print_status "   • System Preferences → Security & Privacy → Privacy → Full Disk Access"
    print_status "   • Add and enable 'cron' and 'Terminal'"
    echo
    print_status "4. 📊 Monitor logs:"
    print_status "   Check: $LOG_DIR/"
    print_status "   Or run: tail -f $LOG_DIR/backup_*.log"
    echo
    print_status "⏰ SCHEDULE:"
    print_status "   • Mac wakes: Every day at 1:55 AM"
    print_status "   • Backup runs: Every Sunday at 2:00 AM"
    print_status "   • Logs saved: $LOG_DIR/"
    print_status "   • Backups saved: $BACKUP_DIR/"
    echo
    print_status "🛠️  MANAGEMENT COMMANDS:"
    print_status "   • View cron jobs: crontab -l"
    print_status "   • Edit cron jobs: crontab -e"
    print_status "   • View power schedule: pmset -g sched"
    print_status "   • Manual backup: $SCRIPT_DIR/weekly-backup.sh"
    echo
    print_warning "⚠️  IMPORTANT: Ensure your Mac is plugged in or has sufficient battery"
    print_warning "    for the scheduled backup times, especially if using a laptop."
}

# Function to show uninstall instructions
show_uninstall() {
    cat > "$SCRIPT_DIR/uninstall.sh" << 'EOF'
#!/bin/bash

# IMAP Backup Uninstaller

echo "Uninstalling imap-backup..."

# Remove cron job
(crontab -l 2>/dev/null | grep -v "imap-backup\|weekly-backup\|wake-for-backup") | crontab -

# Remove power management schedule
sudo pmset repeat cancel

# Remove binary
sudo rm -f /usr/local/bin/imap-backup

# Remove scripts (but keep logs and backups)
rm -rf ~/.imap-backup

echo "Uninstallation completed."
echo "Note: Backup files and logs were preserved in:"
echo "  ~/Documents/email_backups/"
echo "  ~/Library/Logs/imap-backup/"
EOF

    chmod +x "$SCRIPT_DIR/uninstall.sh"
    
    echo
    print_status "🗑️  To uninstall later, run: $SCRIPT_DIR/uninstall.sh"
}

# Main installation function
main() {
    clear
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                   IMAP Backup Installer                     ║"
    echo "║              Automated Weekly Email Backups                 ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo
    
    print_status "Starting installation process..."
    echo
    
    # Pre-flight checks
    check_macos
    check_binary
    
    # Ask for confirmation
    echo -e "${YELLOW}This will:${NC}"
    echo "  • Install imap-backup to $INSTALL_DIR"
    echo "  • Create backup scripts in $SCRIPT_DIR"
    echo "  • Set up weekly cron job (Sundays at 2:00 AM)"
    echo "  • Configure Mac to wake for backups (daily at 1:55 AM)"
    echo "  • Create backup directory: $BACKUP_DIR"
    echo
    read -p "Continue with installation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Installation cancelled."
        exit 0
    fi
    echo
    
    # Installation steps
    install_binary
    create_directories
    create_backup_script
    create_wake_script
    setup_cron
    setup_power_management
    setup_permissions
    
    # Test and finish
    if test_installation; then
        show_uninstall
        show_next_steps
    else
        print_error "Installation completed with some issues. Please check the steps above."
        exit 1
    fi
}

# Run main function
main "$@"