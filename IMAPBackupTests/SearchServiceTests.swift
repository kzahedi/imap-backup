import XCTest
@testable import IMAPBackup

final class SearchServiceTests: XCTestCase {

    var tempDirectory: URL!
    var searchService: SearchService!

    override func setUp() async throws {
        try await super.setUp()

        // Create a temporary directory for each test
        tempDirectory = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: tempDirectory, withIntermediateDirectories: true)

        searchService = SearchService(backupLocation: tempDirectory)
        try await searchService.open()
    }

    override func tearDown() async throws {
        await searchService.close()

        // Clean up temporary directory
        try? FileManager.default.removeItem(at: tempDirectory)

        try await super.tearDown()
    }

    // MARK: - Database Setup Tests

    func testDatabaseOpensSuccessfully() async throws {
        // Database should already be open from setUp
        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 0)
        XCTAssertEqual(stats.attachmentCount, 0)
    }

    func testDatabaseCreatesFile() async throws {
        let dbPath = tempDirectory.appendingPathComponent(".imap_search.db")
        XCTAssertTrue(FileManager.default.fileExists(atPath: dbPath.path))
    }

    // MARK: - Email Indexing Tests

    func testIndexSimpleEmail() async throws {
        let emailData = createSimpleEmail(
            from: "John Doe <john@example.com>",
            subject: "Test Subject",
            body: "This is the body of the email."
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "John Doe",
            senderEmail: "john@example.com",
            subject: "Test Subject",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)
    }

    func testIndexEmailWithNilValues() async throws {
        let emailData = createSimpleEmail(
            from: "",
            subject: "",
            body: ""
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg-nil@example.com>",
            sender: nil,
            senderEmail: nil,
            subject: nil,
            date: nil,
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)
    }

    func testIndexMultipleEmails() async throws {
        for i in 1...10 {
            let emailData = createSimpleEmail(
                from: "sender\(i)@example.com",
                subject: "Subject \(i)",
                body: "Body \(i)"
            )

            try await searchService.indexEmail(
                accountId: "test@example.com",
                mailbox: "INBOX",
                messageId: "<msg\(i)@example.com>",
                sender: "Sender \(i)",
                senderEmail: "sender\(i)@example.com",
                subject: "Subject \(i)",
                date: Date(),
                filePath: "/path/to/email\(i).eml",
                emlData: emailData
            )
        }

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 10)
    }

    func testIndexEmailReplacesOnDuplicate() async throws {
        let emailData1 = createSimpleEmail(from: "test@example.com", subject: "Original", body: "Original body")
        let emailData2 = createSimpleEmail(from: "test@example.com", subject: "Updated", body: "Updated body")

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Original",
            date: Date(),
            filePath: "/path/to/email1.eml",
            emlData: emailData1
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Updated",
            date: Date(),
            filePath: "/path/to/email1.eml",
            emlData: emailData2
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1) // Should still be 1
    }

    // MARK: - Search Tests

    func testSearchBySender() async throws {
        let emailData = createSimpleEmail(
            from: "John Doe <john@example.com>",
            subject: "Test",
            body: "Body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "John Doe",
            senderEmail: "john@example.com",
            subject: "Test Subject",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "John")
        XCTAssertEqual(results.count, 1)
        XCTAssertEqual(results.first?.matchType, .sender)
    }

    func testSearchBySubject() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Important Meeting Tomorrow",
            body: "Body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Important Meeting Tomorrow",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "Meeting")
        XCTAssertEqual(results.count, 1)
        XCTAssertEqual(results.first?.matchType, .subject)
    }

    func testSearchByBody() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Test",
            body: "The quarterly report is attached."
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Test Subject",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "quarterly")
        XCTAssertEqual(results.count, 1)
        XCTAssertEqual(results.first?.matchType, .body)
    }

    func testSearchNoResults() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Test Subject",
            body: "Test body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Test Subject",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "nonexistent")
        XCTAssertEqual(results.count, 0)
    }

    func testSearchWithMultipleWords() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Project Alpha Update",
            body: "The project is on track."
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Project Alpha Update",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "Project Alpha")
        XCTAssertEqual(results.count, 1)
    }

    func testSearchPrefixMatching() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Development Update",
            body: "Body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Development Update",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "Dev")
        XCTAssertEqual(results.count, 1)
    }

    func testSearchLimit() async throws {
        // Index more emails than the limit
        for i in 1...20 {
            let emailData = createSimpleEmail(
                from: "test@example.com",
                subject: "Searchable Subject \(i)",
                body: "Body"
            )

            try await searchService.indexEmail(
                accountId: "test@example.com",
                mailbox: "INBOX",
                messageId: "<msg\(i)@example.com>",
                sender: "Test",
                senderEmail: "test@example.com",
                subject: "Searchable Subject \(i)",
                date: Date().addingTimeInterval(Double(i) * -3600),
                filePath: "/path/to/email\(i).eml",
                emlData: emailData
            )
        }

        let results = try await searchService.search(query: "Searchable", limit: 5)
        XCTAssertEqual(results.count, 5)
    }

    // MARK: - isIndexed Tests

    func testIsIndexed() async throws {
        let emailData = createSimpleEmail(from: "test@example.com", subject: "Test", body: "Body")

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let isIndexed = try await searchService.isIndexed(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>"
        )
        XCTAssertTrue(isIndexed)
    }

    func testIsNotIndexed() async throws {
        let isIndexed = try await searchService.isIndexed(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<nonexistent@example.com>"
        )
        XCTAssertFalse(isIndexed)
    }

    // MARK: - Multipart Email Tests

    func testIndexMultipartEmail() async throws {
        let emailData = createMultipartEmail(
            from: "test@example.com",
            subject: "Multipart Test",
            textBody: "Plain text content",
            htmlBody: "<html><body>HTML content with special formatting</body></html>"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<multipart@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Multipart Test",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)

        // Search for content from plain text
        let results = try await searchService.search(query: "Plain text")
        XCTAssertEqual(results.count, 1)
    }

    // MARK: - Attachment Indexing Tests

    func testIndexEmailWithAttachment() async throws {
        let emailData = createEmailWithAttachment(
            from: "test@example.com",
            subject: "With Attachment",
            body: "See attachment",
            attachmentName: "report.txt",
            attachmentContent: "This is the attachment content"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<attachment@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "With Attachment",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)
        XCTAssertGreaterThanOrEqual(stats.attachmentCount, 1)
    }

    func testSearchByAttachmentFilename() async throws {
        let emailData = createEmailWithAttachment(
            from: "test@example.com",
            subject: "Invoice Email",
            body: "Please see the attached invoice",
            attachmentName: "invoice-2024-001.pdf",
            attachmentContent: ""
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<invoice@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Invoice Email",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "invoice-2024")
        XCTAssertGreaterThanOrEqual(results.count, 1)
    }

    // MARK: - Edge Cases

    func testSearchWithEmptyQuery() async throws {
        // Empty query should be handled gracefully
        // Note: This might throw or return empty results depending on implementation
        do {
            let results = try await searchService.search(query: "")
            // If it doesn't throw, it should return empty results
            XCTAssertEqual(results.count, 0)
        } catch {
            // It's acceptable to throw for empty query
            XCTAssertTrue(true)
        }
    }

    func testSearchWithSpecialCharacters() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Test: Special (Characters) [Here]",
            body: "Body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<special@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Test: Special (Characters) [Here]",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        // Search with special characters should not crash
        let results = try await searchService.search(query: "Special")
        XCTAssertEqual(results.count, 1)
    }

    func testSearchWithQuotes() async throws {
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "He said \"hello world\"",
            body: "Body"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<quotes@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "He said \"hello world\"",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let results = try await searchService.search(query: "hello")
        XCTAssertEqual(results.count, 1)
    }

    func testIndexEmailWithVeryLongBody() async throws {
        let longBody = String(repeating: "This is a test sentence. ", count: 10000)
        let emailData = createSimpleEmail(
            from: "test@example.com",
            subject: "Long Email",
            body: longBody
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<long@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Long Email",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)
    }

    func testIndexEmailWithUnicode() async throws {
        let emailData = createSimpleEmail(
            from: "日本語@example.com",
            subject: "Über die Änderung 中文主题",
            body: "Body with emoji 🎉 and special chars"
        )

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<unicode@example.com>",
            sender: "日本語名前",
            senderEmail: "japanese@example.com",
            subject: "Über die Änderung 中文主题",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 1)

        // Search for unicode content
        let results = try await searchService.search(query: "Änderung")
        XCTAssertEqual(results.count, 1)
    }

    // MARK: - Concurrent Access Tests

    func testConcurrentIndexing() async throws {
        await withTaskGroup(of: Void.self) { group in
            for i in 1...50 {
                group.addTask {
                    let emailData = self.createSimpleEmail(
                        from: "sender\(i)@example.com",
                        subject: "Concurrent Subject \(i)",
                        body: "Body \(i)"
                    )

                    try? await self.searchService.indexEmail(
                        accountId: "test@example.com",
                        mailbox: "INBOX",
                        messageId: "<concurrent\(i)@example.com>",
                        sender: "Sender \(i)",
                        senderEmail: "sender\(i)@example.com",
                        subject: "Concurrent Subject \(i)",
                        date: Date(),
                        filePath: "/path/to/email\(i).eml",
                        emlData: emailData
                    )
                }
            }
        }

        let stats = try await searchService.getStats()
        XCTAssertEqual(stats.emailCount, 50)
    }

    func testConcurrentSearches() async throws {
        // First, index some emails
        for i in 1...10 {
            let emailData = createSimpleEmail(
                from: "test@example.com",
                subject: "Searchable Item \(i)",
                body: "Content \(i)"
            )

            try await searchService.indexEmail(
                accountId: "test@example.com",
                mailbox: "INBOX",
                messageId: "<search\(i)@example.com>",
                sender: "Test",
                senderEmail: "test@example.com",
                subject: "Searchable Item \(i)",
                date: Date(),
                filePath: "/path/to/email\(i).eml",
                emlData: emailData
            )
        }

        // Perform concurrent searches
        await withTaskGroup(of: Int.self) { group in
            for _ in 1...20 {
                group.addTask {
                    let results = try? await self.searchService.search(query: "Searchable")
                    return results?.count ?? 0
                }
            }

            for await count in group {
                XCTAssertEqual(count, 10)
            }
        }
    }

    // MARK: - Reindex Tests

    func testReindexClearsExistingData() async throws {
        // Index some emails
        let emailData = createSimpleEmail(from: "test@example.com", subject: "Test", body: "Body")

        try await searchService.indexEmail(
            accountId: "test@example.com",
            mailbox: "INBOX",
            messageId: "<msg1@example.com>",
            sender: "Test",
            senderEmail: "test@example.com",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email.eml",
            emlData: emailData
        )

        var initialStats = try await searchService.getStats()
        XCTAssertEqual(initialStats.emailCount, 1)

        // Reindex (with no actual files)
        try await searchService.reindexAll { _, _ in }

        let finalStats = try await searchService.getStats()
        XCTAssertEqual(finalStats.emailCount, 0) // Should be cleared
    }

    // MARK: - Helper Methods

    private func createSimpleEmail(from: String, subject: String, body: String) -> Data {
        let email = """
        From: \(from)
        To: recipient@example.com
        Subject: \(subject)
        Date: Mon, 15 Jan 2024 10:30:00 +0000
        Message-ID: <\(UUID().uuidString)@example.com>
        Content-Type: text/plain; charset=utf-8

        \(body)
        """
        return email.data(using: .utf8)!
    }

    private func createMultipartEmail(from: String, subject: String, textBody: String, htmlBody: String) -> Data {
        let boundary = "----=_Part_\(UUID().uuidString)"
        let email = """
        From: \(from)
        To: recipient@example.com
        Subject: \(subject)
        Date: Mon, 15 Jan 2024 10:30:00 +0000
        Message-ID: <\(UUID().uuidString)@example.com>
        Content-Type: multipart/alternative; boundary="\(boundary)"

        --\(boundary)
        Content-Type: text/plain; charset=utf-8

        \(textBody)

        --\(boundary)
        Content-Type: text/html; charset=utf-8

        \(htmlBody)

        --\(boundary)--
        """
        return email.data(using: .utf8)!
    }

    private func createEmailWithAttachment(from: String, subject: String, body: String, attachmentName: String, attachmentContent: String) -> Data {
        let boundary = "----=_Part_\(UUID().uuidString)"
        let encodedContent = Data(attachmentContent.utf8).base64EncodedString()
        let email = """
        From: \(from)
        To: recipient@example.com
        Subject: \(subject)
        Date: Mon, 15 Jan 2024 10:30:00 +0000
        Message-ID: <\(UUID().uuidString)@example.com>
        Content-Type: multipart/mixed; boundary="\(boundary)"

        --\(boundary)
        Content-Type: text/plain; charset=utf-8

        \(body)

        --\(boundary)
        Content-Type: application/octet-stream
        Content-Disposition: attachment; filename="\(attachmentName)"
        Content-Transfer-Encoding: base64

        \(encodedContent)

        --\(boundary)--
        """
        return email.data(using: .utf8)!
    }
}
