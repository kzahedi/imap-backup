import Foundation
import SQLite3
import PDFKit

/// Search result with highlighted snippets
struct SearchResult: Identifiable, Hashable {
    let id: Int64
    let accountId: String
    let mailbox: String
    let messageId: String
    let sender: String
    let senderEmail: String
    let subject: String
    let date: Date
    let filePath: String
    let snippet: String
    let matchType: MatchType

    enum MatchType: String {
        case sender = "Sender"
        case subject = "Subject"
        case body = "Body"
        case attachment = "Attachment"
        case attachmentContent = "Attachment Content"
    }
}

/// Service for full-text search across backed up emails
actor SearchService {
    private var db: OpaquePointer?
    private let dbPath: String
    private let backupLocation: URL

    init(backupLocation: URL) {
        self.backupLocation = backupLocation
        self.dbPath = backupLocation.appendingPathComponent(".imap_search.db").path
    }

    // MARK: - Database Setup

    func open() throws {
        if sqlite3_open(dbPath, &db) != SQLITE_OK {
            throw SearchError.failedToOpen(String(cString: sqlite3_errmsg(db)))
        }

        // Enable WAL mode for better concurrent access
        try execute("PRAGMA journal_mode=WAL")

        // Set busy timeout to 30 seconds - wait if database is locked instead of failing
        try execute("PRAGMA busy_timeout=30000")

        // NORMAL synchronous is safe with WAL and faster
        try execute("PRAGMA synchronous=NORMAL")

        try createTables()
    }

    func close() {
        if db != nil {
            sqlite3_close(db)
            db = nil
        }
    }

    private func createTables() throws {
        // Main email index table
        let createEmailIndexTable = """
            CREATE TABLE IF NOT EXISTS email_index (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                account_id TEXT NOT NULL,
                mailbox TEXT NOT NULL,
                message_id TEXT NOT NULL,
                sender TEXT,
                sender_email TEXT,
                subject TEXT,
                date REAL,
                file_path TEXT NOT NULL,
                body_text TEXT,
                indexed_at REAL,
                UNIQUE(account_id, mailbox, message_id)
            );
            """

        // Attachments table for indexing attachment names and content
        let createAttachmentIndexTable = """
            CREATE TABLE IF NOT EXISTS attachment_index (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                email_id INTEGER REFERENCES email_index(id),
                filename TEXT,
                content_text TEXT
            );
            """

        // FTS5 virtual table for full-text search
        let createFTSTable = """
            CREATE VIRTUAL TABLE IF NOT EXISTS email_fts USING fts5(
                sender,
                sender_email,
                subject,
                body_text,
                content='email_index',
                content_rowid='id'
            );
            """

        // FTS5 for attachment search
        let createAttachmentFTSTable = """
            CREATE VIRTUAL TABLE IF NOT EXISTS attachment_fts USING fts5(
                filename,
                content_text,
                content='attachment_index',
                content_rowid='id'
            );
            """

        // Triggers to keep FTS in sync
        let createInsertTrigger = """
            CREATE TRIGGER IF NOT EXISTS email_fts_insert AFTER INSERT ON email_index BEGIN
                INSERT INTO email_fts(rowid, sender, sender_email, subject, body_text)
                VALUES (new.id, new.sender, new.sender_email, new.subject, new.body_text);
            END;
            """

        let createDeleteTrigger = """
            CREATE TRIGGER IF NOT EXISTS email_fts_delete AFTER DELETE ON email_index BEGIN
                INSERT INTO email_fts(email_fts, rowid, sender, sender_email, subject, body_text)
                VALUES ('delete', old.id, old.sender, old.sender_email, old.subject, old.body_text);
            END;
            """

        let createUpdateTrigger = """
            CREATE TRIGGER IF NOT EXISTS email_fts_update AFTER UPDATE ON email_index BEGIN
                INSERT INTO email_fts(email_fts, rowid, sender, sender_email, subject, body_text)
                VALUES ('delete', old.id, old.sender, old.sender_email, old.subject, old.body_text);
                INSERT INTO email_fts(rowid, sender, sender_email, subject, body_text)
                VALUES (new.id, new.sender, new.sender_email, new.subject, new.body_text);
            END;
            """

        let createAttachmentInsertTrigger = """
            CREATE TRIGGER IF NOT EXISTS attachment_fts_insert AFTER INSERT ON attachment_index BEGIN
                INSERT INTO attachment_fts(rowid, filename, content_text)
                VALUES (new.id, new.filename, new.content_text);
            END;
            """

        let createAttachmentDeleteTrigger = """
            CREATE TRIGGER IF NOT EXISTS attachment_fts_delete AFTER DELETE ON attachment_index BEGIN
                INSERT INTO attachment_fts(attachment_fts, rowid, filename, content_text)
                VALUES ('delete', old.id, old.filename, old.content_text);
            END;
            """

        try execute(createEmailIndexTable)
        try execute(createAttachmentIndexTable)
        try execute(createFTSTable)
        try execute(createAttachmentFTSTable)
        try execute(createInsertTrigger)
        try execute(createDeleteTrigger)
        try execute(createUpdateTrigger)
        try execute(createAttachmentInsertTrigger)
        try execute(createAttachmentDeleteTrigger)

        // Create indexes for faster lookup
        try execute("CREATE INDEX IF NOT EXISTS idx_email_account ON email_index(account_id);")
        try execute("CREATE INDEX IF NOT EXISTS idx_email_path ON email_index(file_path);")
        try execute("CREATE INDEX IF NOT EXISTS idx_attachment_email ON attachment_index(email_id);")
    }

    // MARK: - Indexing

    /// Index an email file
    func indexEmail(
        accountId: String,
        mailbox: String,
        messageId: String,
        sender: String?,
        senderEmail: String?,
        subject: String?,
        date: Date?,
        filePath: String,
        emlData: Data
    ) throws {
        // Extract body text from email
        let bodyText = extractBodyText(from: emlData)

        // Insert into email_index
        let query = """
            INSERT OR REPLACE INTO email_index
            (account_id, mailbox, message_id, sender, sender_email, subject, date, file_path, body_text, indexed_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw SearchError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 3, messageId, -1, SQLITE_TRANSIENT)
        bindTextOrNull(statement, 4, sender)
        bindTextOrNull(statement, 5, senderEmail)
        bindTextOrNull(statement, 6, subject)

        if let date = date {
            sqlite3_bind_double(statement, 7, date.timeIntervalSince1970)
        } else {
            sqlite3_bind_null(statement, 7)
        }

        sqlite3_bind_text(statement, 8, filePath, -1, SQLITE_TRANSIENT)
        bindTextOrNull(statement, 9, bodyText)
        sqlite3_bind_double(statement, 10, Date().timeIntervalSince1970)

        if sqlite3_step(statement) != SQLITE_DONE {
            throw SearchError.indexFailed(String(cString: sqlite3_errmsg(db)))
        }

        // Get the email id for attachment indexing
        let emailId = sqlite3_last_insert_rowid(db)

        // Extract and index attachments
        let attachments = extractAttachments(from: emlData, emailId: emailId, basePath: filePath)
        for attachment in attachments {
            try indexAttachment(attachment)
        }
    }

    /// Index a single attachment
    private func indexAttachment(_ attachment: AttachmentInfo) throws {
        let query = """
            INSERT INTO attachment_index (email_id, filename, content_text)
            VALUES (?, ?, ?)
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw SearchError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_int64(statement, 1, attachment.emailId)
        sqlite3_bind_text(statement, 2, attachment.filename, -1, SQLITE_TRANSIENT)
        bindTextOrNull(statement, 3, attachment.contentText)

        if sqlite3_step(statement) != SQLITE_DONE {
            throw SearchError.indexFailed(String(cString: sqlite3_errmsg(db)))
        }
    }

    /// Check if an email is already indexed
    func isIndexed(accountId: String, mailbox: String, messageId: String) throws -> Bool {
        let query = "SELECT COUNT(*) FROM email_index WHERE account_id = ? AND mailbox = ? AND message_id = ?"

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw SearchError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        sqlite3_bind_text(statement, 1, accountId, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 2, mailbox, -1, SQLITE_TRANSIENT)
        sqlite3_bind_text(statement, 3, messageId, -1, SQLITE_TRANSIENT)

        if sqlite3_step(statement) == SQLITE_ROW {
            return sqlite3_column_int(statement, 0) > 0
        }

        return false
    }

    // MARK: - Search

    /// Search emails by query
    func search(query searchQuery: String, limit: Int = 100) throws -> [SearchResult] {
        var results: [SearchResult] = []

        // Search in email FTS
        let emailResults = try searchEmails(query: searchQuery, limit: limit)
        results.append(contentsOf: emailResults)

        // Search in attachment FTS
        let attachmentResults = try searchAttachments(query: searchQuery, limit: limit)
        results.append(contentsOf: attachmentResults)

        // Sort by date descending and remove duplicates
        let uniqueResults = Dictionary(grouping: results, by: { $0.filePath })
            .mapValues { $0.first! }
            .values
            .sorted { $0.date > $1.date }

        return Array(uniqueResults.prefix(limit))
    }

    private func searchEmails(query searchQuery: String, limit: Int) throws -> [SearchResult] {
        // FTS5 search with snippet
        let query = """
            SELECT
                e.id, e.account_id, e.mailbox, e.message_id,
                e.sender, e.sender_email, e.subject, e.date, e.file_path,
                snippet(email_fts, 0, '<mark>', '</mark>', '...', 20) as sender_snippet,
                snippet(email_fts, 2, '<mark>', '</mark>', '...', 20) as subject_snippet,
                snippet(email_fts, 3, '<mark>', '</mark>', '...', 40) as body_snippet,
                email_fts.sender as fts_sender,
                email_fts.subject as fts_subject,
                email_fts.body_text as fts_body
            FROM email_fts
            JOIN email_index e ON email_fts.rowid = e.id
            WHERE email_fts MATCH ?
            ORDER BY e.date DESC
            LIMIT ?
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw SearchError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        // Escape special FTS5 characters and prepare query
        let ftsQuery = prepareFTSQuery(searchQuery)
        sqlite3_bind_text(statement, 1, ftsQuery, -1, SQLITE_TRANSIENT)
        sqlite3_bind_int(statement, 2, Int32(limit))

        var results: [SearchResult] = []

        while sqlite3_step(statement) == SQLITE_ROW {
            let id = sqlite3_column_int64(statement, 0)
            let accountId = String(cString: sqlite3_column_text(statement, 1))
            let mailbox = String(cString: sqlite3_column_text(statement, 2))
            let messageId = String(cString: sqlite3_column_text(statement, 3))
            let sender = columnTextOrEmpty(statement, 4)
            let senderEmail = columnTextOrEmpty(statement, 5)
            let subject = columnTextOrEmpty(statement, 6)
            let dateValue = sqlite3_column_double(statement, 7)
            let filePath = String(cString: sqlite3_column_text(statement, 8))

            let senderSnippet = columnTextOrEmpty(statement, 9)
            let subjectSnippet = columnTextOrEmpty(statement, 10)
            let bodySnippet = columnTextOrEmpty(statement, 11)

            // Determine match type based on which field has the highlight
            let (snippet, matchType) = determineMatchType(
                senderSnippet: senderSnippet,
                subjectSnippet: subjectSnippet,
                bodySnippet: bodySnippet
            )

            let result = SearchResult(
                id: id,
                accountId: accountId,
                mailbox: mailbox,
                messageId: messageId,
                sender: sender,
                senderEmail: senderEmail,
                subject: subject,
                date: Date(timeIntervalSince1970: dateValue),
                filePath: filePath,
                snippet: snippet,
                matchType: matchType
            )
            results.append(result)
        }

        return results
    }

    private func searchAttachments(query searchQuery: String, limit: Int) throws -> [SearchResult] {
        let query = """
            SELECT
                e.id, e.account_id, e.mailbox, e.message_id,
                e.sender, e.sender_email, e.subject, e.date, e.file_path,
                a.filename,
                snippet(attachment_fts, 0, '<mark>', '</mark>', '...', 20) as filename_snippet,
                snippet(attachment_fts, 1, '<mark>', '</mark>', '...', 40) as content_snippet
            FROM attachment_fts
            JOIN attachment_index a ON attachment_fts.rowid = a.id
            JOIN email_index e ON a.email_id = e.id
            WHERE attachment_fts MATCH ?
            ORDER BY e.date DESC
            LIMIT ?
            """

        var statement: OpaquePointer?
        defer { sqlite3_finalize(statement) }

        guard sqlite3_prepare_v2(db, query, -1, &statement, nil) == SQLITE_OK else {
            throw SearchError.queryFailed(String(cString: sqlite3_errmsg(db)))
        }

        let ftsQuery = prepareFTSQuery(searchQuery)
        sqlite3_bind_text(statement, 1, ftsQuery, -1, SQLITE_TRANSIENT)
        sqlite3_bind_int(statement, 2, Int32(limit))

        var results: [SearchResult] = []

        while sqlite3_step(statement) == SQLITE_ROW {
            let id = sqlite3_column_int64(statement, 0)
            let accountId = String(cString: sqlite3_column_text(statement, 1))
            let mailbox = String(cString: sqlite3_column_text(statement, 2))
            let messageId = String(cString: sqlite3_column_text(statement, 3))
            let sender = columnTextOrEmpty(statement, 4)
            let senderEmail = columnTextOrEmpty(statement, 5)
            let subject = columnTextOrEmpty(statement, 6)
            let dateValue = sqlite3_column_double(statement, 7)
            let filePath = String(cString: sqlite3_column_text(statement, 8))

            let filenameSnippet = columnTextOrEmpty(statement, 10)
            let contentSnippet = columnTextOrEmpty(statement, 11)

            // Determine if match is in filename or content
            let (snippet, matchType): (String, SearchResult.MatchType)
            if filenameSnippet.contains("<mark>") {
                snippet = filenameSnippet
                matchType = .attachment
            } else {
                snippet = contentSnippet
                matchType = .attachmentContent
            }

            let result = SearchResult(
                id: id,
                accountId: accountId,
                mailbox: mailbox,
                messageId: messageId,
                sender: sender,
                senderEmail: senderEmail,
                subject: subject,
                date: Date(timeIntervalSince1970: dateValue),
                filePath: filePath,
                snippet: snippet,
                matchType: matchType
            )
            results.append(result)
        }

        return results
    }

    // MARK: - Text Extraction

    /// Extract plain text body from email
    private func extractBodyText(from data: Data) -> String? {
        guard let content = String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1) else {
            return nil
        }

        // Find the body (after double newline)
        let bodyStart: String.Index
        if let range = content.range(of: "\r\n\r\n") {
            bodyStart = range.upperBound
        } else if let range = content.range(of: "\n\n") {
            bodyStart = range.upperBound
        } else {
            return nil
        }

        var body = String(content[bodyStart...])

        // Check Content-Type for multipart
        let headers = String(content[..<bodyStart])

        if let contentType = parseHeader("Content-Type", in: headers) {
            if contentType.contains("multipart/") {
                // Extract boundary
                if let boundary = extractBoundary(from: contentType) {
                    body = extractTextFromMultipart(body, boundary: boundary)
                }
            } else if contentType.contains("text/html") {
                body = stripHTML(body)
            }
        }

        // Check for quoted-printable or base64 encoding
        if let encoding = parseHeader("Content-Transfer-Encoding", in: headers) {
            if encoding.lowercased().contains("quoted-printable") {
                body = decodeQuotedPrintable(body)
            } else if encoding.lowercased().contains("base64") {
                if let decoded = decodeBase64Text(body) {
                    body = decoded
                }
            }
        }

        // Clean up the text
        body = body.trimmingCharacters(in: .whitespacesAndNewlines)

        // Limit body size to prevent massive index entries
        if body.count > 50000 {
            body = String(body.prefix(50000))
        }

        return body.isEmpty ? nil : body
    }

    /// Extract text from multipart email
    private func extractTextFromMultipart(_ body: String, boundary: String) -> String {
        let parts = body.components(separatedBy: "--\(boundary)")
        var textParts: [String] = []

        for part in parts {
            guard !part.isEmpty && !part.starts(with: "--") else { continue }

            // Find the header/body split in this part
            let partBody: String
            if let range = part.range(of: "\r\n\r\n") {
                let partHeaders = String(part[..<range.lowerBound])
                partBody = String(part[range.upperBound...])

                // Check content type
                if let contentType = parseHeader("Content-Type", in: partHeaders) {
                    if contentType.contains("text/plain") {
                        var text = partBody
                        if let encoding = parseHeader("Content-Transfer-Encoding", in: partHeaders) {
                            if encoding.lowercased().contains("quoted-printable") {
                                text = decodeQuotedPrintable(text)
                            } else if encoding.lowercased().contains("base64") {
                                text = decodeBase64Text(text) ?? text
                            }
                        }
                        textParts.append(text)
                    } else if contentType.contains("text/html") {
                        var text = partBody
                        if let encoding = parseHeader("Content-Transfer-Encoding", in: partHeaders) {
                            if encoding.lowercased().contains("quoted-printable") {
                                text = decodeQuotedPrintable(text)
                            } else if encoding.lowercased().contains("base64") {
                                text = decodeBase64Text(text) ?? text
                            }
                        }
                        textParts.append(stripHTML(text))
                    } else if contentType.contains("multipart/") {
                        // Nested multipart
                        if let nestedBoundary = extractBoundary(from: contentType) {
                            textParts.append(extractTextFromMultipart(partBody, boundary: nestedBoundary))
                        }
                    }
                }
            } else if let range = part.range(of: "\n\n") {
                partBody = String(part[range.upperBound...])
                textParts.append(partBody)
            }
        }

        return textParts.joined(separator: "\n")
    }

    /// Extract attachments info from email
    private func extractAttachments(from data: Data, emailId: Int64, basePath: String) -> [AttachmentInfo] {
        guard let content = String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1) else {
            return []
        }

        var attachments: [AttachmentInfo] = []

        // Find content-type and boundary
        let headerEnd: String.Index
        if let range = content.range(of: "\r\n\r\n") {
            headerEnd = range.lowerBound
        } else if let range = content.range(of: "\n\n") {
            headerEnd = range.lowerBound
        } else {
            return []
        }

        let headers = String(content[..<headerEnd])

        guard let contentType = parseHeader("Content-Type", in: headers),
              contentType.contains("multipart/"),
              let boundary = extractBoundary(from: contentType) else {
            return []
        }

        let body = String(content[headerEnd...])
        let parts = body.components(separatedBy: "--\(boundary)")

        for part in parts {
            guard !part.isEmpty && !part.starts(with: "--") else { continue }

            // Find headers in this part
            let partHeaderEnd: String.Index
            if let range = part.range(of: "\r\n\r\n") {
                partHeaderEnd = range.lowerBound
            } else if let range = part.range(of: "\n\n") {
                partHeaderEnd = range.lowerBound
            } else {
                continue
            }

            let partHeaders = String(part[..<partHeaderEnd])

            // Check for attachment
            if let disposition = parseHeader("Content-Disposition", in: partHeaders),
               disposition.contains("attachment") || disposition.contains("filename") {

                // Extract filename
                if let filename = extractFilename(from: disposition) ?? extractFilename(from: parseHeader("Content-Type", in: partHeaders) ?? "") {

                    // Try to extract text content from certain file types
                    var contentText: String? = nil

                    let lowercaseFilename = filename.lowercased()
                    if lowercaseFilename.hasSuffix(".pdf") {
                        // Try to find and extract PDF content
                        contentText = extractPDFText(from: part, partHeaders: partHeaders)
                    } else if lowercaseFilename.hasSuffix(".txt") ||
                              lowercaseFilename.hasSuffix(".md") ||
                              lowercaseFilename.hasSuffix(".csv") ||
                              lowercaseFilename.hasSuffix(".log") ||
                              lowercaseFilename.hasSuffix(".json") ||
                              lowercaseFilename.hasSuffix(".xml") {
                        contentText = extractTextAttachmentContent(from: part, partHeaders: partHeaders)
                    }

                    attachments.append(AttachmentInfo(
                        emailId: emailId,
                        filename: filename,
                        contentText: contentText
                    ))
                }
            }
        }

        return attachments
    }

    /// Extract PDF text content
    private func extractPDFText(from part: String, partHeaders: String) -> String? {
        // Get the body of the part
        let bodyStart: String.Index
        if let range = part.range(of: "\r\n\r\n") {
            bodyStart = range.upperBound
        } else if let range = part.range(of: "\n\n") {
            bodyStart = range.upperBound
        } else {
            return nil
        }

        var body = String(part[bodyStart...]).trimmingCharacters(in: .whitespacesAndNewlines)

        // Check encoding
        if let encoding = parseHeader("Content-Transfer-Encoding", in: partHeaders),
           encoding.lowercased().contains("base64") {
            // Decode base64
            body = body.replacingOccurrences(of: "\r\n", with: "")
            body = body.replacingOccurrences(of: "\n", with: "")

            guard let data = Data(base64Encoded: body, options: .ignoreUnknownCharacters) else {
                return nil
            }

            // Use PDFKit to extract text
            guard let pdfDocument = PDFDocument(data: data) else {
                return nil
            }

            var fullText = ""
            for pageIndex in 0..<pdfDocument.pageCount {
                if let page = pdfDocument.page(at: pageIndex),
                   let pageText = page.string {
                    fullText += pageText + "\n"
                }
            }

            // Limit text size
            if fullText.count > 50000 {
                fullText = String(fullText.prefix(50000))
            }

            return fullText.isEmpty ? nil : fullText
        }

        return nil
    }

    /// Extract text from plain text attachments
    private func extractTextAttachmentContent(from part: String, partHeaders: String) -> String? {
        let bodyStart: String.Index
        if let range = part.range(of: "\r\n\r\n") {
            bodyStart = range.upperBound
        } else if let range = part.range(of: "\n\n") {
            bodyStart = range.upperBound
        } else {
            return nil
        }

        var body = String(part[bodyStart...])

        // Check encoding
        if let encoding = parseHeader("Content-Transfer-Encoding", in: partHeaders) {
            if encoding.lowercased().contains("base64") {
                body = body.replacingOccurrences(of: "\r\n", with: "")
                body = body.replacingOccurrences(of: "\n", with: "")
                if let data = Data(base64Encoded: body, options: .ignoreUnknownCharacters),
                   let text = String(data: data, encoding: .utf8) {
                    body = text
                }
            } else if encoding.lowercased().contains("quoted-printable") {
                body = decodeQuotedPrintable(body)
            }
        }

        body = body.trimmingCharacters(in: .whitespacesAndNewlines)

        // Limit size
        if body.count > 50000 {
            body = String(body.prefix(50000))
        }

        return body.isEmpty ? nil : body
    }

    // MARK: - Helpers

    private func parseHeader(_ name: String, in headers: String) -> String? {
        let pattern = "(?m)^\(name):\\s*(.+?)(?=\\r?\\n[^\\s\\t]|\\r?\\n\\r?\\n|$)"

        guard let regex = try? NSRegularExpression(pattern: pattern, options: [.caseInsensitive, .dotMatchesLineSeparators]),
              let match = regex.firstMatch(in: headers, range: NSRange(headers.startIndex..., in: headers)),
              let valueRange = Range(match.range(at: 1), in: headers) else {
            return nil
        }

        var value = String(headers[valueRange])
        value = value.replacingOccurrences(of: "\r\n ", with: " ")
        value = value.replacingOccurrences(of: "\r\n\t", with: " ")
        value = value.replacingOccurrences(of: "\n ", with: " ")
        value = value.replacingOccurrences(of: "\n\t", with: " ")

        return value.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func extractBoundary(from contentType: String) -> String? {
        let pattern = #"boundary\s*=\s*"?([^";]+)"?"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: .caseInsensitive),
              let match = regex.firstMatch(in: contentType, range: NSRange(contentType.startIndex..., in: contentType)),
              let range = Range(match.range(at: 1), in: contentType) else {
            return nil
        }
        return String(contentType[range])
    }

    private func extractFilename(from header: String) -> String? {
        // Try filename*= (RFC 5987 encoded)
        if let match = header.range(of: #"filename\*\s*=\s*[^']*'[^']*'([^;\s]+)"#, options: .regularExpression) {
            var filename = String(header[match])
            if let eqRange = filename.range(of: "'", options: .backwards) {
                filename = String(filename[filename.index(after: eqRange.lowerBound)...])
                filename = filename.removingPercentEncoding ?? filename
                return filename
            }
        }

        // Try filename=
        let pattern = #"filename\s*=\s*"?([^";]+)"?"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: .caseInsensitive),
              let match = regex.firstMatch(in: header, range: NSRange(header.startIndex..., in: header)),
              let range = Range(match.range(at: 1), in: header) else {
            return nil
        }
        return String(header[range]).trimmingCharacters(in: .whitespaces)
    }

    private func stripHTML(_ html: String) -> String {
        // Remove HTML tags
        var text = html.replacingOccurrences(of: "<[^>]+>", with: " ", options: .regularExpression)
        // Decode common HTML entities
        text = text.replacingOccurrences(of: "&nbsp;", with: " ")
        text = text.replacingOccurrences(of: "&amp;", with: "&")
        text = text.replacingOccurrences(of: "&lt;", with: "<")
        text = text.replacingOccurrences(of: "&gt;", with: ">")
        text = text.replacingOccurrences(of: "&quot;", with: "\"")
        text = text.replacingOccurrences(of: "&#39;", with: "'")
        // Clean up whitespace
        text = text.replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
        return text.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func decodeQuotedPrintable(_ input: String) -> String {
        var result = ""
        var index = input.startIndex

        while index < input.endIndex {
            let char = input[index]

            if char == "=" {
                let nextIndex = input.index(after: index)

                // Check we have at least one more character
                if nextIndex < input.endIndex {
                    let nextChar = input[nextIndex]

                    // Check for soft line break
                    if nextChar == "\r" || nextChar == "\n" {
                        // Soft line break, skip
                        index = input.index(after: nextIndex)
                        if index < input.endIndex && input[index] == "\n" {
                            index = input.index(after: index)
                        }
                        continue
                    }

                    // Try to decode hex - need 2 characters after =
                    let secondIndex = input.index(after: nextIndex)
                    if secondIndex < input.endIndex {
                        let hex = String(input[nextIndex]) + String(input[secondIndex])
                        if let byte = UInt8(hex, radix: 16) {
                            result.append(Character(UnicodeScalar(byte)))
                            index = input.index(after: secondIndex)
                            continue
                        }
                    }
                }
            }

            result.append(char)
            index = input.index(after: index)
        }

        return result
    }

    private func decodeBase64Text(_ input: String) -> String? {
        let cleaned = input
            .replacingOccurrences(of: "\r\n", with: "")
            .replacingOccurrences(of: "\n", with: "")
            .trimmingCharacters(in: .whitespacesAndNewlines)

        guard let data = Data(base64Encoded: cleaned, options: .ignoreUnknownCharacters) else {
            return nil
        }

        return String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1)
    }

    private func prepareFTSQuery(_ query: String) -> String {
        // Escape special FTS5 characters and use prefix matching
        let escaped = query
            .replacingOccurrences(of: "\"", with: "\"\"")
            .trimmingCharacters(in: .whitespacesAndNewlines)

        // Split into words and use prefix matching for each
        let words = escaped.components(separatedBy: .whitespaces).filter { !$0.isEmpty }

        // Handle empty query
        guard !words.isEmpty else {
            return "\"\""  // Return empty quoted string for empty query
        }

        if words.count == 1 {
            // Single word: use prefix match
            return "\"\(words[0])\"*"
        } else {
            // Multiple words: AND them together with prefix on last word
            let allButLast = words.dropLast().map { "\"\($0)\"" }
            let last = "\"\(words.last!)\"*"
            return (allButLast + [last]).joined(separator: " ")
        }
    }

    private func determineMatchType(senderSnippet: String, subjectSnippet: String, bodySnippet: String) -> (String, SearchResult.MatchType) {
        if senderSnippet.contains("<mark>") {
            return (senderSnippet, .sender)
        } else if subjectSnippet.contains("<mark>") {
            return (subjectSnippet, .subject)
        } else {
            return (bodySnippet, .body)
        }
    }

    private func execute(_ sql: String) throws {
        var errorMessage: UnsafeMutablePointer<CChar>?
        if sqlite3_exec(db, sql, nil, nil, &errorMessage) != SQLITE_OK {
            let message = errorMessage != nil ? String(cString: errorMessage!) : "Unknown error"
            sqlite3_free(errorMessage)
            throw SearchError.executeFailed(message)
        }
    }

    private func bindTextOrNull(_ statement: OpaquePointer?, _ index: Int32, _ value: String?) {
        if let value = value {
            sqlite3_bind_text(statement, index, value, -1, SQLITE_TRANSIENT)
        } else {
            sqlite3_bind_null(statement, index)
        }
    }

    private func columnTextOrEmpty(_ statement: OpaquePointer?, _ index: Int32) -> String {
        if let text = sqlite3_column_text(statement, index) {
            return String(cString: text)
        }
        return ""
    }

    // MARK: - Reindexing

    /// Reindex all emails from backup folder
    func reindexAll(progressHandler: @escaping (Int, Int) -> Void) async throws {
        // Clear existing index
        try execute("DELETE FROM attachment_index;")
        try execute("DELETE FROM email_index;")

        // Find all .eml files
        let fileManager = FileManager.default
        let enumerator = fileManager.enumerator(
            at: backupLocation,
            includingPropertiesForKeys: [.isRegularFileKey],
            options: [.skipsHiddenFiles]
        )

        var emlFiles: [URL] = []
        while let url = enumerator?.nextObject() as? URL {
            if url.pathExtension.lowercased() == "eml" {
                emlFiles.append(url)
            }
        }

        let total = emlFiles.count
        var indexed = 0

        for url in emlFiles {
            // Extract account and mailbox from path
            let relativePath = url.path.replacingOccurrences(of: backupLocation.path + "/", with: "")
            let pathComponents = relativePath.components(separatedBy: "/")

            guard pathComponents.count >= 2 else { continue }

            let accountId = pathComponents[0]
            let mailbox = pathComponents.dropFirst().dropLast().joined(separator: "/")

            // Read and parse the email
            guard let data = try? Data(contentsOf: url) else { continue }

            if let metadata = EmailParser.parseMetadata(from: data) {
                try indexEmail(
                    accountId: accountId,
                    mailbox: mailbox,
                    messageId: metadata.messageId,
                    sender: metadata.senderName,
                    senderEmail: metadata.senderEmail,
                    subject: metadata.subject,
                    date: metadata.date,
                    filePath: url.path,
                    emlData: data
                )
            }

            indexed += 1
            progressHandler(indexed, total)
        }
    }

    /// Get index statistics
    func getStats() throws -> (emailCount: Int, attachmentCount: Int) {
        var emailCount = 0
        var attachmentCount = 0

        var statement: OpaquePointer?

        if sqlite3_prepare_v2(db, "SELECT COUNT(*) FROM email_index", -1, &statement, nil) == SQLITE_OK {
            if sqlite3_step(statement) == SQLITE_ROW {
                emailCount = Int(sqlite3_column_int(statement, 0))
            }
        }
        sqlite3_finalize(statement)

        if sqlite3_prepare_v2(db, "SELECT COUNT(*) FROM attachment_index", -1, &statement, nil) == SQLITE_OK {
            if sqlite3_step(statement) == SQLITE_ROW {
                attachmentCount = Int(sqlite3_column_int(statement, 0))
            }
        }
        sqlite3_finalize(statement)

        return (emailCount, attachmentCount)
    }
}

// MARK: - Supporting Types

private struct AttachmentInfo {
    let emailId: Int64
    let filename: String
    let contentText: String?
}

private let SQLITE_TRANSIENT = unsafeBitCast(-1, to: sqlite3_destructor_type.self)

// MARK: - Errors

enum SearchError: LocalizedError {
    case failedToOpen(String)
    case queryFailed(String)
    case indexFailed(String)
    case executeFailed(String)

    var errorDescription: String? {
        switch self {
        case .failedToOpen(let msg): return "Failed to open search database: \(msg)"
        case .queryFailed(let msg): return "Search query failed: \(msg)"
        case .indexFailed(let msg): return "Indexing failed: \(msg)"
        case .executeFailed(let msg): return "Execute failed: \(msg)"
        }
    }
}
