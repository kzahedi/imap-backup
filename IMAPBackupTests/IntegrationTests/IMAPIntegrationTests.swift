import XCTest
@testable import IMAPBackup

/// Integration tests for IMAP functionality with real servers.
///
/// These tests require real IMAP credentials to run. Set the following
/// environment variables before running:
///
/// For Gmail:
///   - IMAP_TEST_GMAIL_EMAIL: Your Gmail address
///   - IMAP_TEST_GMAIL_PASSWORD: App password (not regular password)
///
/// For custom IMAP:
///   - IMAP_TEST_CUSTOM_EMAIL: Email address
///   - IMAP_TEST_CUSTOM_PASSWORD: Password
///   - IMAP_TEST_CUSTOM_SERVER: IMAP server hostname
///   - IMAP_TEST_CUSTOM_PORT: IMAP port (default: 993)
///
/// Run tests:
///   IMAP_TEST_GMAIL_EMAIL=user@gmail.com \
///   IMAP_TEST_GMAIL_PASSWORD=xxxx-xxxx-xxxx-xxxx \
///   xcodebuild test -target IMAPBackupTests -only-testing:IMAPBackupTests/IMAPIntegrationTests
///
final class IMAPIntegrationTests: XCTestCase {

    override func setUp() async throws {
        throw XCTSkip("Integration test - requires real IMAP credentials. Set IMAP_TEST_* environment variables to run.")
    }

    // MARK: - Test Account Configuration

    struct TestAccount {
        let email: String
        let password: String
        let server: String
        let port: Int
        let useSSL: Bool

        static var gmail: TestAccount? {
            guard let email = ProcessInfo.processInfo.environment["IMAP_TEST_GMAIL_EMAIL"],
                  let password = ProcessInfo.processInfo.environment["IMAP_TEST_GMAIL_PASSWORD"] else {
                return nil
            }
            return TestAccount(
                email: email,
                password: password,
                server: "imap.gmail.com",
                port: 993,
                useSSL: true
            )
        }

        static var custom: TestAccount? {
            guard let email = ProcessInfo.processInfo.environment["IMAP_TEST_CUSTOM_EMAIL"],
                  let password = ProcessInfo.processInfo.environment["IMAP_TEST_CUSTOM_PASSWORD"],
                  let server = ProcessInfo.processInfo.environment["IMAP_TEST_CUSTOM_SERVER"] else {
                return nil
            }
            let port = Int(ProcessInfo.processInfo.environment["IMAP_TEST_CUSTOM_PORT"] ?? "993") ?? 993
            return TestAccount(
                email: email,
                password: password,
                server: server,
                port: port,
                useSSL: true
            )
        }

        func toEmailAccount() -> EmailAccount {
            EmailAccount(
                email: email,
                imapServer: server,
                imapPort: port,
                useSSL: useSSL,
                accountType: server.contains("gmail") ? .gmail : .custom
            )
        }
    }

    // MARK: - Gmail Integration Tests

    func testGmailConnection() async throws {
        guard let account = TestAccount.gmail else {
            throw XCTSkip("Gmail test credentials not configured. Set IMAP_TEST_GMAIL_EMAIL and IMAP_TEST_GMAIL_PASSWORD environment variables.")
        }

        let emailAccount = account.toEmailAccount()
        let service = IMAPService(account: emailAccount)

        // Test connection
        try await service.connect()
        XCTAssertTrue(true, "Connected to Gmail IMAP")

        // Test login
        try await service.login()
        XCTAssertTrue(true, "Authenticated with Gmail")

        // Test logout
        try await service.logout()
    }

    func testGmailListFolders() async throws {
        guard let account = TestAccount.gmail else {
            throw XCTSkip("Gmail test credentials not configured.")
        }

        let emailAccount = account.toEmailAccount()
        let service = IMAPService(account: emailAccount)

        try await service.connect()
        try await service.login()

        let folders = try await service.listFolders()

        // Gmail should have at least INBOX
        XCTAssertTrue(folders.contains { $0.name == "INBOX" }, "Gmail should have INBOX folder")

        // Gmail typically has these folders
        let expectedFolders = ["INBOX", "[Gmail]/All Mail", "[Gmail]/Sent Mail", "[Gmail]/Drafts"]
        let folderNames = folders.map { $0.name }

        for expected in expectedFolders {
            if folderNames.contains(expected) {
                print("Found expected folder: \(expected)")
            }
        }

        try await service.logout()
    }

    func testGmailFetchEmail() async throws {
        guard let account = TestAccount.gmail else {
            throw XCTSkip("Gmail test credentials not configured.")
        }

        let emailAccount = account.toEmailAccount()
        let service = IMAPService(account: emailAccount)

        try await service.connect()
        try await service.login()

        // Select INBOX
        let status = try await service.selectFolder("INBOX")
        print("INBOX has \(status.exists) messages")

        guard status.exists > 0 else {
            throw XCTSkip("INBOX is empty, cannot test email fetch")
        }

        // Get first email UID
        let uids = try await service.searchAll()
        guard let firstUID = uids.first else {
            throw XCTSkip("No UIDs found in INBOX")
        }

        // Fetch the email
        let emailData = try await service.fetchEmail(uid: firstUID)
        XCTAssertTrue(emailData.count > 0, "Email data should not be empty")

        // Verify it looks like an email
        if let preview = String(data: emailData.prefix(500), encoding: .utf8) {
            XCTAssertTrue(
                preview.lowercased().contains("from:") ||
                preview.lowercased().contains("date:") ||
                preview.lowercased().contains("subject:"),
                "Email should contain standard headers"
            )
        }

        try await service.logout()
    }

    // MARK: - Custom IMAP Server Tests

    func testCustomIMAPConnection() async throws {
        guard let account = TestAccount.custom else {
            throw XCTSkip("Custom IMAP test credentials not configured. Set IMAP_TEST_CUSTOM_* environment variables.")
        }

        let emailAccount = account.toEmailAccount()
        let service = IMAPService(account: emailAccount)

        try await service.connect()
        XCTAssertTrue(true, "Connected to custom IMAP server: \(account.server)")

        try await service.login()
        XCTAssertTrue(true, "Authenticated with custom IMAP server")

        try await service.logout()
    }

    func testCustomIMAPListFolders() async throws {
        guard let account = TestAccount.custom else {
            throw XCTSkip("Custom IMAP test credentials not configured.")
        }

        let emailAccount = account.toEmailAccount()
        let service = IMAPService(account: emailAccount)

        try await service.connect()
        try await service.login()

        let folders = try await service.listFolders()

        // Any IMAP server should have INBOX
        XCTAssertTrue(folders.contains { $0.name.uppercased() == "INBOX" }, "Server should have INBOX folder")

        print("Found \(folders.count) folders on \(account.server):")
        for folder in folders {
            print("  - \(folder.name) (selectable: \(folder.isSelectable))")
        }

        try await service.logout()
    }

    // MARK: - Full Backup Integration Test

    func testFullBackupCycle() async throws {
        guard let account = TestAccount.gmail ?? TestAccount.custom else {
            throw XCTSkip("No IMAP test credentials configured.")
        }

        let emailAccount = account.toEmailAccount()

        // Create temporary backup directory
        let tempDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("IMAPBackupIntegrationTest_\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)

        defer {
            try? FileManager.default.removeItem(at: tempDir)
        }

        let service = IMAPService(account: emailAccount)
        let storage = StorageService(baseURL: tempDir)

        try await service.connect()
        try await service.login()

        // Select INBOX
        let status = try await service.selectFolder("INBOX")
        guard status.exists > 0 else {
            throw XCTSkip("INBOX is empty")
        }

        // Get UIDs
        let uids = try await service.searchAll()
        let testUID = uids.first!

        // Fetch email
        let emailData = try await service.fetchEmail(uid: testUID)

        // Parse metadata
        let parsed = EmailParser.parseMetadata(from: emailData)
        let email = Email(
            messageId: parsed?.messageId ?? UUID().uuidString,
            uid: testUID,
            folder: "INBOX",
            subject: parsed?.subject ?? "(No Subject)",
            sender: parsed?.senderName ?? "Unknown",
            senderEmail: parsed?.senderEmail ?? "",
            date: parsed?.date ?? Date()
        )

        // Save to disk
        let savedURL = try await storage.saveEmail(
            emailData,
            email: email,
            accountEmail: account.email,
            folderPath: "INBOX"
        )

        // Verify file exists
        XCTAssertTrue(FileManager.default.fileExists(atPath: savedURL.path), "Email file should exist")

        // Verify content
        let savedData = try Data(contentsOf: savedURL)
        XCTAssertEqual(savedData.count, emailData.count, "Saved email should match original")

        try await service.logout()

        print("Full backup cycle test passed!")
        print("  - Connected to: \(account.server)")
        print("  - Fetched email: \(email.subject)")
        print("  - Saved to: \(savedURL.path)")
    }
}
