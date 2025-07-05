#!/bin/bash

# IMAP Backup Uninstaller
# Removes automatic backup system while preserving backup files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

main() {
    clear
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                   IMAP Backup Uninstaller                   ║"
    echo "║                Remove Automatic Backup System               ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo
    
    print_warning "This will remove the automatic backup system."
    print_status "Backup files and logs will be PRESERVED in:"
    print_status "  • ~/Documents/email_backups/ (your email backups)"
    print_status "  • ~/Library/Logs/imap-backup/ (backup logs)"
    echo
    print_warning "The following will be REMOVED:"
    print_status "  • Cron job (weekly backup schedule)"
    print_status "  • Power management wake schedule"
    print_status "  • /usr/local/bin/imap-backup binary"
    print_status "  • ~/.imap-backup/ scripts directory"
    echo
    
    read -p "Continue with uninstallation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Uninstallation cancelled."
        exit 0
    fi
    echo
    
    print_header "Removing cron jobs..."
    (crontab -l 2>/dev/null | grep -v "imap-backup\|weekly-backup\|wake-for-backup") | crontab - 2>/dev/null || true
    print_status "✓ Cron jobs removed"
    
    print_header "Removing power management schedule..."
    sudo pmset repeat cancel 2>/dev/null || true
    print_status "✓ Wake schedule removed"
    
    print_header "Removing binary..."
    if [[ -f "/usr/local/bin/imap-backup" ]]; then
        sudo rm -f "/usr/local/bin/imap-backup"
        print_status "✓ Binary removed from /usr/local/bin/"
    else
        print_status "○ Binary not found (already removed)"
    fi
    
    print_header "Removing scripts..."
    if [[ -d "$HOME/.imap-backup" ]]; then
        rm -rf "$HOME/.imap-backup"
        print_status "✓ Scripts directory removed"
    else
        print_status "○ Scripts directory not found (already removed)"
    fi
    
    echo
    print_header "Uninstallation completed successfully!"
    echo
    print_status "📋 PRESERVATION NOTICE:"
    print_status "Your backup files and logs have been preserved:"
    echo
    if [[ -d "$HOME/Documents/email_backups" ]]; then
        BACKUP_SIZE=$(du -sh "$HOME/Documents/email_backups" 2>/dev/null | cut -f1)
        print_status "  📁 Backups: ~/Documents/email_backups/ (${BACKUP_SIZE:-unknown size})"
    else
        print_status "  📁 Backups: ~/Documents/email_backups/ (directory not found)"
    fi
    
    if [[ -d "$HOME/Library/Logs/imap-backup" ]]; then
        LOG_COUNT=$(find "$HOME/Library/Logs/imap-backup" -name "*.log" 2>/dev/null | wc -l | tr -d ' ')
        print_status "  📝 Logs: ~/Library/Logs/imap-backup/ (${LOG_COUNT} log files)"
    else
        print_status "  📝 Logs: ~/Library/Logs/imap-backup/ (directory not found)"
    fi
    
    echo
    print_status "🔧 To use imap-backup manually in the future:"
    print_status "   1. Rebuild: go build -o imap-backup ."
    print_status "   2. Install: sudo cp imap-backup /usr/local/bin/"
    print_status "   3. Run: imap-backup backup -v"
    echo
    print_status "💾 To clean up backup files manually (if desired):"
    print_status "   rm -rf ~/Documents/email_backups/"
    print_status "   rm -rf ~/Library/Logs/imap-backup/"
    echo
    print_status "🔄 To reinstall the automatic system:"
    print_status "   ./install.sh"
    echo
}

main "$@"