# IMAP Backup Tool

A fast, efficient command-line tool for backing up IMAP email accounts written in Go.

## Features

- **Preserves read/unread status**: Uses IMAP EXAMINE command to access folders in read-only mode
- **Incremental backups**: Only downloads new emails, avoiding duplicates
- **Attachment preservation**: Saves attachments with original filenames
- **Folder structure mirroring**: Preserves IMAP folder hierarchies with nested directories
- **Mac Integration**: Reads email accounts from Mail.app and Internet Accounts
- **OAuth2 Support**: Handles OAuth2 authentication for Gmail, Outlook, and other providers
- **Multiple account support**: Backup multiple email accounts in one run
- **Concurrent processing**: Fast backups with configurable concurrency
- **Integrity verification**: Uses checksums to verify backup integrity
- **Charset support**: Handles emails in various character encodings (ISO-8859-1, Windows-1252, etc.)

## Installation

### Option 1: Automatic Installation (Recommended)

For automatic weekly backups with Mac wake scheduling:

```bash
# Build the binary
go build -o imap-backup .

# Run the automated installer
./install.sh
```

This sets up:
- ✅ Weekly automatic backups (Sundays at 2:00 AM)
- ⏰ Mac wake scheduling (daily at 1:55 AM)
- 📝 Automatic logging and notifications
- 🔧 Complete system integration

**📋 See [INSTALLATION.md](INSTALLATION.md) for detailed setup guide**

### Option 2: Manual Installation

#### Prerequisites

- Go 1.21 or later

#### Build from source

```bash
git clone <repository-url>
cd imap-backup
go mod download
go build -o imap-backup
```

## Usage

### Discover Accounts

First, check what email accounts are available:

```bash
./imap-backup accounts
```

This will attempt to discover email accounts from:
1. Mail.app preferences 
2. Mac's Internet Accounts system
3. Your configuration file (if it exists)

### Setup

If no accounts are automatically discovered, create a configuration file:

```bash
./imap-backup setup
```

This creates `~/.imap-backup.yaml` with sample configuration. Edit this file to add your IMAP account details.

### Backup

Backup all configured accounts:

```bash
./imap-backup backup
```

Backup a specific account:

```bash
./imap-backup backup -a "Gmail"
```

Dry run (show what would be backed up):

```bash
./imap-backup backup --dry-run
```

Specify output directory:

```bash
./imap-backup backup -o /path/to/backup
```

### Command Line Options

- `-o, --output`: Output directory for backups (default: ./backup)
- `-a, --account`: Specific account to backup (default: all configured accounts)
- `-d, --dry-run`: Show what would be backed up without actually downloading
- `-c, --max-concurrent`: Maximum concurrent connections per account (default: 5)
- `--ignore-charset-errors`: Continue backup even when charset parsing fails
- `-v, --verbose`: Verbose output
- `--config`: Custom config file path

#### Account Management

**Discovery:**
- `accounts`: List discovered email accounts from all sources
- `accounts -p`: Show passwords (use with caution)

**JSON Store Management:**
- `account add --name "Gmail" --username user@gmail.com`: Add account (auto-detects settings)
- `account list`: List configured accounts from JSON store
- `account list --verbose`: Show detailed information including timestamps
- `account remove [account-id]`: Remove account and password from keychain
- `account remove [account-id] --keep-password`: Remove account but keep password in keychain
- `account test [account-id]`: Test account connectivity

**Keychain Management:**
- `keychain test`: Test keychain access
- `keychain list`: List IMAP-related passwords in keychain
- `keychain add [server] [username]`: Add password to keychain
- `keychain remove [server] [username]`: Remove password from keychain

**Troubleshooting:**
- `test-charset [charset-name]`: Test if a charset is supported
- `test-folders`: Test folder structure conversion

## Configuration

### Configuration File Format

```yaml
accounts:
  - name: "Gmail"
    host: "imap.gmail.com"
    port: 993
    username: "your-email@gmail.com"
    password: "your-app-password"
    use_ssl: true
  - name: "Outlook"
    host: "outlook.office365.com"
    port: 993
    username: "your-email@outlook.com"
    password: "your-password"
    use_ssl: true
```

### Gmail Setup

For Gmail accounts:

1. Enable 2-factor authentication
2. Generate an app-specific password
3. Use the app password in the configuration

### Mac Integration

The tool provides comprehensive Mac integration:

1. **Mail.app Integration**: Reads account settings from Mail.app's preferences
2. **Internet Accounts**: Accesses accounts configured in System Preferences
3. **Keychain Access**: Retrieves passwords from Mac's keychain  
4. **OAuth2 Support**: Handles OAuth2 tokens for modern email providers

The tool automatically detects authentication types:
- **OAuth2**: For Gmail, Outlook, Yahoo (automatically handles token refresh)
- **Password**: For traditional IMAP accounts with username/password

If Mac integration fails, the tool falls back to the configuration file.

## Account Storage Options

The tool supports multiple account storage methods with different priorities:

### 1. JSON Store + Keychain (Recommended)
- **Configuration**: `~/.imap-backup-accounts.json`
- **Passwords**: Mac's keychain (secure)
- **Features**: Full account management, password reuse, timestamps
- **Usage**: `imap-backup account add --name "Gmail" --username user@gmail.com`

### 2. Mac Integration (Auto-discovery)
- **Source**: Mail.app preferences and Internet Accounts
- **Passwords**: Keychain and OAuth2 tokens
- **Features**: Automatic discovery, OAuth2 support
- **Usage**: Automatic when accounts are configured in Mail.app

### 3. YAML Configuration (Fallback)
- **Configuration**: `~/.imap-backup.yaml`
- **Passwords**: Plain text (not recommended)
- **Features**: Manual configuration, cross-platform compatibility
- **Usage**: `imap-backup setup`

### Account Storage Priority

The tool loads accounts in this order:
1. JSON store (if exists and has accounts)
2. Mac Internet Accounts (if accounts found)
3. YAML configuration file (fallback)

### Password Management Features

- **Secure Storage**: Passwords stored in Mac's keychain, not config files
- **Password Reuse**: Keep passwords when removing/re-adding accounts
- **OAuth2 Support**: Automatic token management for modern providers
- **Independent Management**: Manage keychain passwords separately from accounts

## Backup Structure

Backups are organized as follows:

```
backup/
├── AccountName/
│   ├── INBOX/
│   │   ├── 1.eml          # Raw email message
│   │   ├── 1.json         # Message metadata
│   │   ├── 2.eml
│   │   ├── 2.json
│   │   └── attachments/
│   │       └── 1/         # Attachments for message UID 1
│   │           ├── document.pdf
│   │           └── image.jpg
│   ├── Sent/
│   ├── Drafts/
│   └── Friends/           # Nested folder structure
│       ├── Mario/         
│       │   ├── 10.eml
│       │   └── 10.json
│       └── Luigi/
│           └── Photos/
└── AnotherAccount/
    └── ...
```

Each message is stored as:
- `.eml` file: Raw email message in standard format
- `.json` file: Metadata including subject, from, to, date, flags, headers, and checksums
- `attachments/` directory: Contains subdirectories for each message's attachments

## Security Notes

- Configuration files containing passwords are created with restricted permissions (600)
- The tool uses SSL/TLS connections by default
- Passwords are stored in plain text in the configuration file - consider using Mac's Keychain integration instead
- For production use, consider encrypting the backup directory

## Troubleshooting

### Common Issues

1. **Authentication errors**: 
   - For Gmail, make sure you're using an app-specific password
   - For corporate accounts, check if OAuth2 is required

2. **Connection errors**:
   - Verify the IMAP server hostname and port
   - Check if your firewall allows outbound connections on the IMAP port

3. **Permission errors**:
   - Make sure the output directory is writable
   - Check that the configuration file has correct permissions

4. **Charset/encoding errors**:
   - Use `--ignore-charset-errors` flag to continue backup despite encoding issues
   - Test specific charsets with: `imap-backup test-charset iso-8859-1`
   - Emails with unsupported charsets are saved with raw content and marked with parsing errors

### Debug Mode

Run with verbose output to see detailed information:

```bash
./imap-backup backup -v
```

## Development

### Code Quality

This project uses comprehensive code quality tools:

```bash
# Run all quality checks
make quality

# Individual checks
make test        # Run tests
make coverage    # Generate coverage report
make lint        # Run golangci-lint
make staticcheck # Run staticcheck
make security    # Run gosec security scanner

# Full CI pipeline
make ci
```

### SonarQube Integration

The project includes SonarQube integration for continuous code quality monitoring:

- **Local Setup**: See [SONAR_SETUP.md](SONAR_SETUP.md) for detailed instructions
- **Quick Start**: `docker-compose -f docker-compose.sonar.yml up -d`
- **Analysis**: `make sonar-prepare && make sonar-docker`

Quality metrics tracked:
- Test coverage (target: >80%)
- Security vulnerabilities and hotspots
- Code smells and technical debt
- Duplicated code blocks
- Maintainability rating

### Available Make Targets

```bash
make help    # Show all available targets
make build   # Build the binary
make test    # Run tests
make clean   # Clean build artifacts
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run quality checks: `make quality`
5. Add tests if applicable
6. Submit a pull request

All contributions are automatically analyzed for:
- Code quality and maintainability
- Security vulnerabilities
- Test coverage
- Coding standards compliance

## License

This project is licensed under the MIT License - see the LICENSE file for details.
