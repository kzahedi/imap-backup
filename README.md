# IMAP Backup

A native macOS app for backing up emails from IMAP servers (Gmail, IONOS, and custom servers).

## Features

- **Multi-account support**: Gmail (via App Passwords), IONOS, and custom IMAP servers
- **Incremental backup**: Only downloads new emails
- **Folder hierarchy preservation**: Mirrors your email folder structure
- **Human-readable filenames**: `<timestamp>_<sender>.eml`
- **Attachment extraction**: Saves attachments in organized subfolders
- **Menubar app**: Quick access to backup status
- **Real-time progress**: See download progress per account
- **Scheduled backups**: Automatic hourly/daily/weekly backups

## Requirements

- macOS 14.0 (Sonoma) or later
- Xcode 15.0+ (for building from source)

## Building

1. Clone the repository:
   ```bash
   git clone https://github.com/kzahedi/imap-backup.git
   cd imap-backup
   ```

2. Open in Xcode:
   ```bash
   open IMAPBackup.xcodeproj
   ```

3. Build and run (⌘R)

## Usage

### Adding an Account

1. Click the "+" button in the sidebar or go to Settings → Accounts
2. Select your account type (Gmail, IONOS, or Custom)
3. Enter your email and password (App Password for Gmail)
4. Test the connection
5. Click "Add Account"

### Gmail Setup

Gmail requires an App Password instead of your regular password:

1. Go to [Google App Passwords](https://myaccount.google.com/apppasswords)
2. Generate a new app password for "Mail"
3. Use this 16-character password in IMAP Backup

### Running a Backup

- Click "Backup All" in the toolbar to backup all enabled accounts
- Or select an account and click "Start Backup" for a single account

### Backup Location

Backups are stored in your Documents folder by default:
```
~/Documents/IMAPBackup/
└── your@email.com/
    ├── INBOX/
    ├── Sent/
    └── Work/
        └── Projects/
```

You can change the location in Settings → General.

## File Format

Emails are saved as standard `.eml` files with human-readable names:
```
20240115_143022_John_Smith.eml
```

Attachments are saved in a subfolder:
```
20240115_143022__John_Smith_attachments/
├── document.pdf
└── image.png
```

## Development

### Project Structure

```
IMAPBackup/
├── App/           # App entry point and delegate
├── Models/        # Data models
├── Services/      # IMAP, storage, backup logic
├── Views/         # SwiftUI views
└── Resources/     # Info.plist, entitlements
```

### Key Technologies

- SwiftUI for the UI
- Network.framework for IMAP connections
- Swift Concurrency (async/await)

## License

MIT License
