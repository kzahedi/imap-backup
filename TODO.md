# IMAP Backup - TODO

## In Progress

- [ ] **Email Search Feature** - Full-text search across all downloaded emails
  - [ ] Search by sender/author name and email
  - [ ] Search by subject
  - [ ] Search email body text (plain text and HTML)
  - [ ] Search attachment filenames
  - [ ] Extract and search PDF text content (using PDFKit)
  - [ ] Extract and search plain text attachments (.txt, .md, .csv, .log)
  - [ ] SQLite FTS5 full-text search index for speed
  - [ ] Search results view with highlighted snippets
  - [ ] Open email file directly from search results
  - [ ] Keyboard shortcut (Cmd+F) and menubar integration

## Core Features

- [ ] **Resume Interrupted Downloads** - Track incomplete downloads and resume from last position
- [ ] **Verify Incomplete Downloads** - Check file integrity and re-download corrupted files
- [ ] **Retry Failed Downloads** - Automatic retry with exponential backoff for failed emails

## Account Management

- [ ] **Internet Accounts Integration** - Read credentials from macOS Accounts.framework
- [ ] **OAuth2 for Google** - Use AuthenticationServices for Google account OAuth tokens
- [ ] **Secure Credential Storage** - Store passwords in macOS Keychain instead of UserDefaults

## User Interface

- [ ] **Backup History/Log View** - Show history of past backups with details
- [ ] **Notifications** - System notifications on backup completion or errors
- [ ] **Start at Login** - Launch app automatically on macOS startup (LoginItems)
- [ ] **Dock Icon Toggle** - Option to hide dock icon (menubar-only mode)
- [ ] **Password Manager Integration** - macOS password autofill in SwiftUI sheets (using NSSecureTextField workaround)

## Storage & Sync

- [ ] **Attachment Extraction** - Option to extract attachments to separate folders
- [ ] **Retention Policies** - Auto-delete old backups based on age or count
- [ ] **Backup Verification** - Verify backed up emails match server state

## Performance

- [ ] **Parallel Downloads** - Download multiple emails concurrently
- [ ] **Rate Limiting** - Respect server limits with configurable throttling
- [ ] **Large Attachment Streaming** - Stream large attachments to disk instead of memory

## Error Handling

- [ ] **Detailed Error Logging** - Write errors to log file for debugging
- [ ] **Connection Recovery** - Automatically reconnect on network failures
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
- [x] Incremental backups (SQLite database tracking)
- [x] Scheduled backups (manual, hourly, daily, weekly with time selection)
- [x] iCloud Drive storage option
- [x] Menubar app with quick controls
- [x] Real-time progress display
- [x] Statistics dashboard
- [x] Folder hierarchy preservation
- [x] Human-readable filenames
- [x] RFC 2047 MIME header decoding
- [x] Complete .eml files with embedded attachments
- [x] App icon (blue envelope with green download arrow)
