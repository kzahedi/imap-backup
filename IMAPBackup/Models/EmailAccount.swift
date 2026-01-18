import Foundation

struct EmailAccount: Identifiable, Codable, Hashable {
    let id: UUID
    var email: String
    var imapServer: String
    var port: Int
    var username: String
    var useSSL: Bool
    var isEnabled: Bool
    var lastBackupDate: Date?

    // Password is stored in Keychain, not in this struct
    // This property is only used during account creation/update
    private var _password: String?

    enum CodingKeys: String, CodingKey {
        case id, email, imapServer, port, username, useSSL, isEnabled, lastBackupDate
        // Note: password is excluded from Codable
    }

    init(
        id: UUID = UUID(),
        email: String,
        imapServer: String,
        port: Int = 993,
        username: String? = nil,
        password: String,
        useSSL: Bool = true,
        isEnabled: Bool = true,
        lastBackupDate: Date? = nil
    ) {
        self.id = id
        self.email = email
        self.imapServer = imapServer
        self.port = port
        self.username = username ?? email
        self._password = password
        self.useSSL = useSSL
        self.isEnabled = isEnabled
        self.lastBackupDate = lastBackupDate
    }

    /// Get password from Keychain
    func getPassword() async -> String? {
        // First check if we have a temporary password (during account creation)
        if let tempPassword = _password, !tempPassword.isEmpty {
            return tempPassword
        }
        // Otherwise fetch from Keychain
        return try? await KeychainService.shared.getPassword(for: id)
    }

    /// Save password to Keychain
    func savePassword(_ password: String) async throws {
        try await KeychainService.shared.savePassword(password, for: id)
    }

    /// Delete password from Keychain
    func deletePassword() async throws {
        try await KeychainService.shared.deletePassword(for: id)
    }

    /// Check if password exists
    func hasPassword() async -> Bool {
        if _password != nil { return true }
        return await KeychainService.shared.hasPassword(for: id)
    }

    // Convenience initializer for Gmail
    static func gmail(email: String, appPassword: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.gmail.com",
            port: 993,
            password: appPassword,
            useSSL: true
        )
    }

    // Convenience initializer for IONOS
    static func ionos(email: String, password: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.ionos.de",
            port: 993,
            password: password,
            useSSL: true
        )
    }
}
