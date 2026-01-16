import Foundation
import SQLite3

/// Service for tracking backed up emails using SQLite
actor DatabaseService {
    private var db: OpaquePointer?
    private let dbPath: String

    init(backupLocation: URL) {
        self.dbPath = backupLocation.appendingPathComponent(".imap_backup.db").path
    }

    // MARK: - Database Setup

    func open() throws {
        if sqlite3_open(dbPath, &db) != SQLITE_OK {
            throw DatabaseError.failedToOpen(String(cString: sqlite3_errmsg(db)))
        }
        try createTables()
    }

    func close() {
        if db != nil {
            sqlite3_close(db)
            db = nil
        }
    }

    private func createTables() throws {
        let createEmailsTable = """
            CREATE TABLE IF NOT EXISTS emails (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                account_id TEXT NOT NULL,
                message_id TEXT NOT NULL,
                uid INTEGER NOT NULL,
                mailbox TEXT NOT NULL,
                sender TEXT,
                subject TEXT,
                date DATETIME,
                file_path TEXT,
                downloaded_at DATETIME,
                has_attachments BOOLEAN,
                attachment_count INTEGER,
                download_complete BOOLEAN DEFAULT FALSE,
                UNIQUE(account_id, mailbox, uid)
            );
            """

        let createAttachmentsTable = """
            CREATE TABLE IF NOT EXISTS attachments (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                email_id INTEGER REFERENCES emails(id),
                filename TEXT,
                file_path TEXT,
                size INTEGER,
                downloaded BOOLEAN DEFAULT FALSE
            );
            """

        let createSyncStateTable = """
            CREATE TABLE IF NOT EXISTS sync_state (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                account_id TEXT NOT NULL,
                mailbox TEXT NOT NULL,
                last_uid INTEGER,
                last_sync DATETIME,
                UNIQUE(account_id, mailbox)
            );
            """

        let createIndexes = """
            CREATE INDEX IF NOT EXISTS idx_emails_account_mailbox ON emails(account_id, mailbox);
            CREATE INDEX IF NOT EXISTS idx_emails_message_id ON emails(message_id);
            CREATE INDEX IF NOT EXISTS idx_emails_uid ON emails(account_id, mailbox, uid);
            """

        try execute(createEmailsTable)
        try execute(createAttachmentsTable)
        try execute(createSyncStateTable)
        try execute(createIndexes)
    }

    // MARK: - Email Tracking

    /// Check if an email has already been backed up
    func isEmailBackedUp(accountId: String, mailbox: String, uid: UInt32) throws -> Bool {
        let query = """
            SELECT COUNT(*) FROM emails
            WHERE account_id = ? AND mailbox = ? AND uid = ? AND download_complete = 1
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)
        sqlite3_bind_int(statement, 3, Int32(uid))

        if sqlite3_step(statement) == SQLITE_ROW {
            return sqlite3_column_int(statement, 0) > 0
        }

        return false
    }

    /// Check if an email exists by Message-ID (for cross-folder deduplication)
    func isMessageIdBackedUp(accountId: String, messageId: String) throws -> Bool {
        let query = """
            SELECT COUNT(*) FROM emails
            WHERE account_id = ? AND message_id = ? AND download_complete = 1
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, messageId, -1, SQLITE_TRANSIENT)

        if sqlite3_step(statement) == SQLITE_ROW {
            return sqlite3_column_int(statement, 0) > 0
        }

        return false
    }

    /// Record a backed up email
    func recordEmail(
        accountId: String,
        messageId: String,
        uid: UInt32,
        mailbox: String,
        sender: String?,
        subject: String?,
        date: Date?,
        filePath: String,
        hasAttachments: Bool = false,
        attachmentCount: Int = 0
    ) throws {
        let query = """
            INSERT OR REPLACE INTO emails
            (account_id, message_id, uid, mailbox, sender, subject, date, file_path,
             downloaded_at, has_attachments, attachment_count, download_complete)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, messageId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_int(statement, 3, Int32(uid))
        sqlite3_bind_text(statement, 4, mailbox, -1, SQLITE_TRANSIENT)

        if let sender = sender {
            sqlite3_bind_text(statement, 5, sender, -1, SQLITE_TRANSIENT)
        } else {
            sqlite3_bind_null(statement, 5)
        }

        if let subject = subject {
            sqlite3_bind_text(statement, 6, subject, -1, SQLITE_TRANSIENT)
        } else {
            sqlite3_bind_null(statement, 6)
        }

        if let date = date {
            sqlite3_bind_double(statement, 7, date.timeIntervalSince1970)
        } else {
            sqlite3_bind_null(statement, 7)
        }

        sqlite3_bind_text(statement, 8, filePath, -1, SQLITE_TRANSIENT)
        sqlite3_bind_double(statement, 9, Date().timeIntervalSince1970)
        sqlite3_bind_int(statement, 10, hasAttachments ? 1 : 0)
        sqlite3_bind_int(statement, 11, Int32(attachmentCount))

        if sqlite3_step(statement) != SQLITE_DONE {
            throw DatabaseError.insertFailed(String(cString: sqlite3_errmsg(db)))
        }
    }

    // MARK: - Sync State

    /// Get all backed up UIDs for a mailbox
    func getBackedUpUIDs(accountId: String, mailbox: String) throws -> Set<UInt32> {
        let query = """
            SELECT uid FROM emails
            WHERE account_id = ? AND mailbox = ? AND download_complete = 1
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)

        var uids = Set<UInt32>()
        while sqlite3_step(statement) == SQLITE_ROW {
            uids.insert(UInt32(sqlite3_column_int(statement, 0)))
        }

        return uids
    }

    /// Update sync state for a mailbox
    func updateSyncState(accountId: String, mailbox: String, lastUID: UInt32) throws {
        let query = """
            INSERT OR REPLACE INTO sync_state (account_id, mailbox, last_uid, last_sync)
            VALUES (?, ?, ?, ?)
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)
        sqlite3_bind_int(statement, 3, Int32(lastUID))
        sqlite3_bind_double(statement, 4, Date().timeIntervalSince1970)

        if sqlite3_step(statement) != SQLITE_DONE {
            throw DatabaseError.insertFailed(String(cString: sqlite3_errmsg(db)))
        }
    }

    /// Get last synced UID for a mailbox
    func getLastUID(accountId: String, mailbox: String) throws -> UInt32? {
        let query = "SELECT last_uid FROM sync_state WHERE account_id = ? AND mailbox = ?"

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)

        if sqlite3_step(statement) == SQLITE_ROW {
            return UInt32(sqlite3_column_int(statement, 0))
        }

        return nil
    }

    // MARK: - Statistics

    /// Get total email count for an account
    func getEmailCount(accountId: String) throws -> Int {
        let query = "SELECT COUNT(*) FROM emails WHERE account_id = ? AND download_complete = 1"

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)

        if sqlite3_step(statement) == SQLITE_ROW {
            return Int(sqlite3_column_int(statement, 0))
        }

        return 0
    }

    /// Get total email count across all accounts
    func getTotalEmailCount() throws -> Int {
        let query = "SELECT COUNT(*) FROM emails WHERE download_complete = 1"

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw DatabaseError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        if sqlite3_step(statement) == SQLITE_ROW {
            return Int(sqlite3_column_int(statement, 0))
        }

        return 0
    }

    // MARK: - Helpers

    private func execute(_ sql: String) throws {
        var errorMessage: UnsafeMutablePointer<CChar>?
        if sqlite3_exec(db, sql, nil, nil, &errorMessage) != SQLITE_OK {
            let message = errorMessage != nil ? String(cString: errorMessage!) : "Unknown error"
            sqlite3_free(errorMessage)
            throw DatabaseError.executeFailed(message)
        }
    }
}

// MARK: - SQLite Transient

private let SQLITE_TRANSIENT = unsafeBitCast(-1, to: sqlite3_destructor_type.self)

// MARK: - Errors

enum DatabaseError: LocalizedError {
    case failedToOpen(String)
    case queryFailed(String)
    case insertFailed(String)
    case executeFailed(String)

    var errorDescription: String? {
        switch self {
        case .failedToOpen(let msg): return "Failed to open database: \(msg)"
        case .queryFailed(let msg): return "Query failed: \(msg)"
        case .insertFailed(let msg): return "Insert failed: \(msg)"
        case .executeFailed(let msg): return "Execute failed: \(msg)"
        }
    }
}
