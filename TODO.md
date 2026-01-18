# IMAP Backup - TODO

## Core Features

- [x] **Resume Interrupted Downloads** - Atomic writes with temp files, cleanup on startup
- [x] **Verify Incomplete Downloads** - Validate email structure before saving
- [x] **Retry Failed Downloads** - Exponential backoff (1s, 2s, 4s), max 3 attempts

## Account Management

- [ ] **Internet Accounts Integration** - Read credentials from macOS Accounts.framework
- [ ] **OAuth2 for Google** - Use AuthenticationServices for Google account OAuth tokens
- [x] **Secure Credential Storage** - Passwords stored in macOS Keychain

## User Interface

- [x] **Backup History/Log View** - Show history of past backups with details
- [x] **Notifications** - System notifications on backup completion or errors
- [x] **Start at Login** - Launch app automatically on macOS startup (SMAppService)
- [x] **Dock Icon Toggle** - Option to hide dock icon (menubar-only mode)
- [x] **Password Manager Integration** - macOS password autofill in SwiftUI sheets (using NSSecureTextField)

## Storage & Sync

- [ ] **Attachment Extraction** - Option to extract attachments to separate folders
- [x] **Retention Policies** - Auto-delete old backups based on age or count
- [ ] **Backup Verification** - Verify backed up emails match server state

## Performance

- [x] **Parallel Downloads** - Download multiple emails concurrently (v0.2.0)
- [x] **Rate Limiting** - Respect server limits with configurable throttling
- [ ] **Large Attachment Streaming** - Stream large attachments to disk instead of memory

## Error Handling

- [x] **Detailed Error Logging** - Write errors to log file for debugging
- [x] **Connection Recovery** - Automatically reconnect on network failures
- [ ] **Conflict Resolution** - Handle email modifications between syncs

## Testing

- [ ] **Unit Tests** - Add tests for IMAP parsing, storage, and database
- [ ] **Integration Tests** - Test with real IMAP servers
- [ ] **UI Tests** - Automated UI testing

## Documentation

- [ ] **Screenshots** - Add screenshots to README
- [ ] **User Guide** - Detailed usage documentation
- [ ] **Contributing Guide** - Guidelines for contributors

---

## Completed

- [x] Multi-account support (Gmail, IONOS, custom IMAP)
- [x] Full mailbox sync (downloads all emails)
- [x] Incremental backups (file-based UID tracking, no database)
- [x] Parallel downloads (no database locking issues)
- [x] Scheduled backups (manual, hourly, daily, weekly with time selection)
- [x] iCloud Drive storage option
- [x] Menubar app with quick controls
- [x] Real-time progress display
- [x] Statistics dashboard
- [x] Folder hierarchy preservation
- [x] Human-readable filenames: `<UID>_<timestamp>_<sender>.eml`
- [x] RFC 2047 MIME header decoding
- [x] Complete .eml files with embedded attachments
- [x] App icon (blue envelope with green download arrow)
- [x] **Email Search Feature** - Full-text search across all downloaded emails
  - [x] Search by sender/author name and email
  - [x] Search by subject
  - [x] Search email body text (plain text and HTML)
  - [x] Search attachment filenames
  - [x] File-based search (reads .eml files directly, no database)
  - [x] Search results view with highlighted snippets
  - [x] Open email file directly from search results
  - [x] Keyboard shortcut (Cmd+F) and menubar integration
