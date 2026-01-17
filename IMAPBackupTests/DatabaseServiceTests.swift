import XCTest
@testable import IMAPBackup

final class DatabaseServiceTests: XCTestCase {

    var tempDirectory: URL!
    var databaseService: DatabaseService!

    override func setUp() async throws {
        try await super.setUp()

        // Create a temporary directory for each test
        tempDirectory = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: tempDirectory, withIntermediateDirectories: true)

        databaseService = DatabaseService(backupLocation: tempDirectory)
        try await databaseService.open()
    }

    override func tearDown() async throws {
        await databaseService.close()

        // Clean up temporary directory
        try? FileManager.default.removeItem(at: tempDirectory)

        try await super.tearDown()
    }

    // MARK: - Database Setup Tests

    func testDatabaseOpensSuccessfully() async throws {
        // Database should already be open from setUp
        // Try to get email count to verify it works
        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 0)
    }

    func testDatabaseCreatesFile() async throws {
        let dbPath = tempDirectory.appendingPathComponent(".imap_backup.db")
        XCTAssertTrue(FileManager.default.fileExists(atPath: dbPath.path))
    }

    // MARK: - Email Recording Tests

    func testRecordEmail() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<msg1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "John Doe",
            subject: "Test Subject",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1)
    }

    func testRecordMultipleEmails() async throws {
        for i in 1...10 {
            try await databaseService.recordEmail(
                accountId: "test@example.com",
                messageId: "<msg\(i)@example.com>",
                uid: UInt32(i),
                mailbox: "INBOX",
                sender: "Sender \(i)",
                subject: "Subject \(i)",
                date: Date(),
                filePath: "/path/to/email\(i).eml"
            )
        }

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 10)
    }

    func testRecordEmailWithNilValues() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<msg-nil@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: nil,
            subject: nil,
            date: nil,
            filePath: "/path/to/email.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1)
    }

    func testRecordEmailReplacesOnDuplicate() async throws {
        // Record first version
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<msg1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Original Sender",
            subject: "Original Subject",
            date: Date(),
            filePath: "/path/to/email1.eml"
        )

        // Record same email again (same account, mailbox, uid)
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<msg1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Updated Sender",
            subject: "Updated Subject",
            date: Date(),
            filePath: "/path/to/email1_updated.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1) // Should still be 1, not 2
    }

    // MARK: - Email Backup Check Tests

    func testIsEmailBackedUp() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<msg1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let isBackedUp = try await databaseService.isEmailBackedUp(
            accountId: "test@example.com",
            mailbox: "INBOX",
            uid: 1
        )
        XCTAssertTrue(isBackedUp)
    }

    func testIsEmailNotBackedUp() async throws {
        let isBackedUp = try await databaseService.isEmailBackedUp(
            accountId: "test@example.com",
            mailbox: "INBOX",
            uid: 999
        )
        XCTAssertFalse(isBackedUp)
    }

    func testIsMessageIdBackedUp() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<unique-msg@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let isBackedUp = try await databaseService.isMessageIdBackedUp(
            accountId: "test@example.com",
            messageId: "<unique-msg@example.com>"
        )
        XCTAssertTrue(isBackedUp)
    }

    // MARK: - UID Tracking Tests

    func testGetBackedUpUIDs() async throws {
        let uids: [UInt32] = [1, 5, 10, 15, 20]

        for uid in uids {
            try await databaseService.recordEmail(
                accountId: "test@example.com",
                messageId: "<msg\(uid)@example.com>",
                uid: uid,
                mailbox: "INBOX",
                sender: "Test",
                subject: "Test",
                date: Date(),
                filePath: "/path/to/email\(uid).eml"
            )
        }

        let backedUpUIDs = try await databaseService.getBackedUpUIDs(
            accountId: "test@example.com",
            mailbox: "INBOX"
        )

        XCTAssertEqual(backedUpUIDs.count, 5)
        XCTAssertTrue(backedUpUIDs.contains(1))
        XCTAssertTrue(backedUpUIDs.contains(5))
        XCTAssertTrue(backedUpUIDs.contains(10))
        XCTAssertTrue(backedUpUIDs.contains(15))
        XCTAssertTrue(backedUpUIDs.contains(20))
        XCTAssertFalse(backedUpUIDs.contains(999))
    }

    func testGetBackedUpUIDsForDifferentMailboxes() async throws {
        // Add emails to INBOX
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<inbox1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/inbox1.eml"
        )

        // Add emails to Sent
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<sent1@example.com>",
            uid: 1, // Same UID but different mailbox
            mailbox: "Sent",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/sent1.eml"
        )

        let inboxUIDs = try await databaseService.getBackedUpUIDs(
            accountId: "test@example.com",
            mailbox: "INBOX"
        )
        let sentUIDs = try await databaseService.getBackedUpUIDs(
            accountId: "test@example.com",
            mailbox: "Sent"
        )

        XCTAssertEqual(inboxUIDs.count, 1)
        XCTAssertEqual(sentUIDs.count, 1)
    }

    // MARK: - Sync State Tests

    func testUpdateAndGetSyncState() async throws {
        try await databaseService.updateSyncState(
            accountId: "test@example.com",
            mailbox: "INBOX",
            lastUID: 100
        )

        let lastUID = try await databaseService.getLastUID(
            accountId: "test@example.com",
            mailbox: "INBOX"
        )

        XCTAssertEqual(lastUID, 100)
    }

    func testGetLastUIDForNonexistentMailbox() async throws {
        let lastUID = try await databaseService.getLastUID(
            accountId: "test@example.com",
            mailbox: "NonExistent"
        )

        XCTAssertNil(lastUID)
    }

    func testUpdateSyncStateReplaces() async throws {
        try await databaseService.updateSyncState(
            accountId: "test@example.com",
            mailbox: "INBOX",
            lastUID: 50
        )

        try await databaseService.updateSyncState(
            accountId: "test@example.com",
            mailbox: "INBOX",
            lastUID: 100
        )

        let lastUID = try await databaseService.getLastUID(
            accountId: "test@example.com",
            mailbox: "INBOX"
        )

        XCTAssertEqual(lastUID, 100) // Should be updated, not 50
    }

    // MARK: - Statistics Tests

    func testGetTotalEmailCount() async throws {
        // Add emails for different accounts
        try await databaseService.recordEmail(
            accountId: "account1@example.com",
            messageId: "<msg1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email1.eml"
        )

        try await databaseService.recordEmail(
            accountId: "account2@example.com",
            messageId: "<msg2@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email2.eml"
        )

        let total = try await databaseService.getTotalEmailCount()
        XCTAssertEqual(total, 2)
    }

    // MARK: - Multi-Account Tests

    func testEmailsAreSeparatedByAccount() async throws {
        try await databaseService.recordEmail(
            accountId: "alice@example.com",
            messageId: "<alice1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/alice1.eml"
        )

        try await databaseService.recordEmail(
            accountId: "bob@example.com",
            messageId: "<bob1@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/bob1.eml"
        )

        let aliceCount = try await databaseService.getEmailCount(accountId: "alice@example.com")
        let bobCount = try await databaseService.getEmailCount(accountId: "bob@example.com")

        XCTAssertEqual(aliceCount, 1)
        XCTAssertEqual(bobCount, 1)
    }

    // MARK: - Edge Cases

    func testRecordEmailWithVeryLongSubject() async throws {
        let longSubject = String(repeating: "A", count: 10000)

        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<long@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "Test",
            subject: longSubject,
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1)
    }

    func testRecordEmailWithSpecialCharacters() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<special@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "O'Brien \"Bob\" <test>",
            subject: "Subject with 'quotes' and \"double quotes\"",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1)
    }

    func testRecordEmailWithUnicodeCharacters() async throws {
        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<unicode@example.com>",
            uid: 1,
            mailbox: "INBOX",
            sender: "日本語名前",
            subject: "Über die Änderung 中文主题 🎉",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 1)
    }

    func testRecordEmailWithLargeUID() async throws {
        // Note: Database uses Int32 for UID binding, so max safe UID is Int32.max (2,147,483,647)
        // UIDs above this may overflow when cast to Int32
        let largeUID: UInt32 = 2_000_000_000

        try await databaseService.recordEmail(
            accountId: "test@example.com",
            messageId: "<largeuid@example.com>",
            uid: largeUID,
            mailbox: "INBOX",
            sender: "Test",
            subject: "Test",
            date: Date(),
            filePath: "/path/to/email.eml"
        )

        let isBackedUp = try await databaseService.isEmailBackedUp(
            accountId: "test@example.com",
            mailbox: "INBOX",
            uid: largeUID
        )
        XCTAssertTrue(isBackedUp)
    }

    // MARK: - Concurrent Access Tests

    func testConcurrentRecords() async throws {
        await withTaskGroup(of: Void.self) { group in
            for i in 1...100 {
                group.addTask {
                    try? await self.databaseService.recordEmail(
                        accountId: "test@example.com",
                        messageId: "<concurrent\(i)@example.com>",
                        uid: UInt32(i),
                        mailbox: "INBOX",
                        sender: "Sender \(i)",
                        subject: "Subject \(i)",
                        date: Date(),
                        filePath: "/path/to/email\(i).eml"
                    )
                }
            }
        }

        let count = try await databaseService.getEmailCount(accountId: "test@example.com")
        XCTAssertEqual(count, 100)
    }
}
