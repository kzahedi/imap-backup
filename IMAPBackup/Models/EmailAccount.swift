import Foundation

/// Authentication type for email accounts
enum AuthenticationType: String, Codable {
    case password = "password"
    case oauth2 = "oauth2"
}

struct EmailAccount: Identifiable, Codable, Hashable {
    let id: UUID
    var email: String
    var imapServer: String
    var port: Int
    var username: String
    var useSSL: Bool
    var isEnabled: Bool
    var lastBackupDate: Date?
    var authType: AuthenticationType

    // Password is stored in Keychain, not in this struct
    // This property is only used during account creation/update
    private var _password: String?

    enum CodingKeys: String, CodingKey {
        case id, email, imapServer, port, username, useSSL, isEnabled, lastBackupDate, authType
        // Note: password is excluded from Codable
    }

    // Custom decoder to handle older accounts without authType
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(UUID.self, forKey: .id)
        email = try container.decode(String.self, forKey: .email)
        imapServer = try container.decode(String.self, forKey: .imapServer)
        port = try container.decode(Int.self, forKey: .port)
        username = try container.decode(String.self, forKey: .username)
        useSSL = try container.decode(Bool.self, forKey: .useSSL)
        isEnabled = try container.decode(Bool.self, forKey: .isEnabled)
        lastBackupDate = try container.decodeIfPresent(Date.self, forKey: .lastBackupDate)
        // Default to password auth for older accounts
        authType = try container.decodeIfPresent(AuthenticationType.self, forKey: .authType) ?? .password
    }

    init(
        id: UUID = UUID(),
        email: String,
        imapServer: String,
        port: Int = 993,
        username: String? = nil,
        password: String? = nil,
        useSSL: Bool = true,
        isEnabled: Bool = true,
        lastBackupDate: Date? = nil,
        authType: AuthenticationType = .password
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
        self.authType = authType
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

    // MARK: - OAuth Token Management

    /// Keychain key for OAuth tokens
    private var oauthTokenKey: String {
        "oauth_\(id.uuidString)"
    }

    /// Save OAuth tokens to Keychain
    func saveOAuthTokens(_ tokens: GoogleOAuthTokens) async throws {
        let encoder = JSONEncoder()
        let data = try encoder.encode(tokens)
        guard let tokenString = String(data: data, encoding: .utf8) else {
            throw NSError(domain: "EmailAccount", code: 1, userInfo: [NSLocalizedDescriptionKey: "Failed to encode OAuth tokens"])
        }
        try await KeychainService.shared.savePassword(tokenString, for: id, service: "com.kzahedi.MailKeep.oauth")
    }

    /// Get OAuth tokens from Keychain
    func getOAuthTokens() async -> GoogleOAuthTokens? {
        guard let tokenString = try? await KeychainService.shared.getPassword(for: id, service: "com.kzahedi.MailKeep.oauth"),
              let data = tokenString.data(using: .utf8) else {
            return nil
        }
        return try? JSONDecoder().decode(GoogleOAuthTokens.self, from: data)
    }

    /// Delete OAuth tokens from Keychain
    func deleteOAuthTokens() async throws {
        try await KeychainService.shared.deletePassword(for: id, service: "com.kzahedi.MailKeep.oauth")
    }

    /// Get a valid access token, refreshing if necessary
    func getValidAccessToken() async throws -> String {
        guard authType == .oauth2 else {
            throw NSError(domain: "EmailAccount", code: 2, userInfo: [NSLocalizedDescriptionKey: "Account is not using OAuth"])
        }

        guard var tokens = await getOAuthTokens() else {
            throw NSError(domain: "EmailAccount", code: 3, userInfo: [NSLocalizedDescriptionKey: "No OAuth tokens found"])
        }

        // Refresh if expired
        if tokens.isExpired {
            logInfo("Access token expired, refreshing...")
            tokens = try await GoogleOAuthService.shared.refreshAccessToken(refreshToken: tokens.refreshToken)
            try await saveOAuthTokens(tokens)
            logInfo("Access token refreshed successfully")
        }

        return tokens.accessToken
    }

    // MARK: - Convenience Initializers

    // Convenience initializer for Gmail with App Password
    static func gmail(email: String, appPassword: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.gmail.com",
            port: 993,
            password: appPassword,
            useSSL: true,
            authType: .password
        )
    }

    // Convenience initializer for Gmail with OAuth
    static func gmailOAuth(email: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.gmail.com",
            port: 993,
            useSSL: true,
            authType: .oauth2
        )
    }

    // Convenience initializer for IONOS
    static func ionos(email: String, password: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.ionos.de",
            port: 993,
            password: password,
            useSSL: true,
            authType: .password
        )
    }
}
