# IMAP Backup

A native macOS menubar app for backing up emails from IMAP servers. Supports Gmail (via App Passwords), IONOS, and custom IMAP servers.

![macOS](https://img.shields.io/badge/macOS-14.0+-blue)
![Swift](https://img.shields.io/badge/Swift-5.9+-orange)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- **Multi-account support** - Gmail, IONOS, and custom IMAP servers
- **Scheduled backups** - Manual, hourly, daily, or weekly with custom time selection
- **iCloud Drive sync** - Automatically sync backups across your Mac devices
- **Menubar app** - Quick access to backup status and controls
- **Real-time progress** - See download progress, speed, and ETA per account
- **Statistics dashboard** - View total emails, storage size, and folder counts
- **Folder hierarchy preservation** - Mirrors your email folder structure
- **Human-readable filenames** - `YYYYMMDD_HHMMSS_sender.eml` format
- **Complete .eml files** - Full RFC 5322 emails with embedded attachments
- **International character support** - Proper RFC 2047 MIME decoding for subjects

## Screenshots

### Main Window
The main window shows account details, backup statistics, and progress.

![Main Window](screenshots/main-window.png)

### Menubar
Quick access to backup status, scheduling, and controls from your menubar.

![Menubar](screenshots/menubar.png)

### Settings
Configure storage location (local or iCloud), backup schedule, and manage accounts.

![Settings](screenshots/settings.png)

## Requirements

- macOS 14.0 (Sonoma) or later
- Xcode 15.0+ (for building from source)

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/kzahedi/imap-backup.git
   cd imap-backup
   ```

2. Build with Xcode:
   ```bash
   xcodebuild -project IMAPBackup.xcodeproj -scheme IMAPBackup -configuration Release build
   ```

3. Copy to Applications:
   ```bash
   cp -R ~/Library/Developer/Xcode/DerivedData/IMAPBackup-*/Build/Products/Release/IMAPBackup.app ~/Applications/
   ```

Or open `IMAPBackup.xcodeproj` in Xcode and build with ⌘R.

## Usage

### Adding an Account

1. Click the "+" button in the sidebar or go to **Settings → Accounts**
2. Select your account type (Gmail, IONOS, or Custom)
3. Enter your email and password
4. Click **Test Connection** to verify
5. Click **Add Account**

### Gmail Setup

Gmail requires an App Password instead of your regular password:

1. Enable 2-Factor Authentication on your Google account
2. Go to [Google App Passwords](https://myaccount.google.com/apppasswords)
3. Generate a new app password for "Mail"
4. Use this 16-character password in IMAP Backup

### IONOS Setup

Use your regular IONOS email password with server `imap.ionos.de` on port 993 (SSL).

### Running a Backup

- **Backup All**: Click in toolbar or menubar to backup all enabled accounts
- **Single Account**: Select an account and click "Start Backup"
- **Scheduled**: Set automatic backups in Settings → Schedule

### Scheduling Options

| Schedule | Description |
|----------|-------------|
| Manual | Only backup when you click the button |
| Hourly | Backup every hour |
| Daily | Backup once per day at your chosen time |
| Weekly | Backup once per week at your chosen time |

### Storage Options

- **Local Storage**: Backups saved to `~/Documents/IMAPBackup/`
- **iCloud Drive**: Sync backups across all your Macs automatically
- **Custom Location**: Choose any folder via Settings → General

## Backup Structure

```
IMAPBackup/
└── user@example.com/
    ├── INBOX/
    │   ├── 20240115_143022_John_Smith.eml
    │   └── 20240115_091544_Jane_Doe.eml
    ├── Sent/
    └── Work/
        └── Projects/
```

### File Naming

Emails are saved with human-readable names:
```
YYYYMMDD_HHMMSS_sender.eml
```

Example: `20240115_143022_John_Smith.eml`

### File Format

Each `.eml` file is a complete RFC 5322 email containing:
- All headers (From, To, Subject, Date, Message-ID, etc.)
- Plain text and HTML body
- Attachments (embedded as MIME parts)

You can open `.eml` files directly in Apple Mail or any email client.

## Architecture

```
IMAPBackup/
├── App/                    # App entry point, delegate, menubar
├── Models/
│   ├── EmailAccount.swift  # Account configuration
│   ├── BackupState.swift   # Progress tracking
│   └── Email.swift         # Email metadata
├── Services/
│   ├── BackupManager.swift # Backup coordination, scheduling
│   ├── IMAPService.swift   # IMAP protocol implementation
│   ├── StorageService.swift# File system operations
│   └── EmailParser.swift   # RFC 2047/5322 parsing
└── Views/
    ├── MainWindow/         # Account list, details, settings
    ├── MenubarView.swift   # Menubar dropdown
    └── Components/         # Reusable UI components
```

### Key Technologies

- **SwiftUI** - Modern declarative UI
- **Network.framework** - Low-level IMAP over TLS
- **Swift Concurrency** - async/await for all network operations
- **IMAP Protocol** - Direct implementation (LIST, SELECT, UID FETCH)

## Troubleshooting

### Authentication Failed

- **Gmail**: Make sure you're using an App Password, not your regular password
- **IONOS**: Try both `imap.ionos.de` and `imap.1und1.de`
- Verify your password doesn't have trailing spaces

### Connection Issues

- Check your firewall allows outbound connections on port 993
- Verify the IMAP server address is correct
- Ensure SSL is enabled for port 993

### Missing Emails

- The app currently downloads emails marked as "unseen"
- Future versions will support full mailbox sync

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with Swift and SwiftUI for macOS.
