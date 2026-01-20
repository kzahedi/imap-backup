import Foundation
import Security

/// Service for securely storing credentials in macOS Keychain
actor KeychainService {
    static let shared = KeychainService()

    private let service = "com.kzahedi.MailKeep"

    private init() {}

    // MARK: - Password Management

    /// Save password to Keychain
    func savePassword(_ password: String, for accountId: UUID) throws {
        let account = accountId.uuidString
        guard let passwordData = password.data(using: .utf8) else {
            throw KeychainError.encodingFailed
        }

        // Delete any existing password first
        try? deletePassword(for: accountId)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: passwordData,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else {
            throw KeychainError.saveFailed(status)
        }
    }

    /// Retrieve password from Keychain
    func getPassword(for accountId: UUID) throws -> String {
        let account = accountId.uuidString

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let passwordData = result as? Data,
              let password = String(data: passwordData, encoding: .utf8) else {
            throw KeychainError.notFound
        }

        return password
    }

    /// Delete password from Keychain
    func deletePassword(for accountId: UUID) throws {
        let account = accountId.uuidString

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]

        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }

    /// Check if password exists in Keychain
    func hasPassword(for accountId: UUID) -> Bool {
        do {
            _ = try getPassword(for: accountId)
            return true
        } catch {
            return false
        }
    }

    /// Migrate password from plaintext to Keychain
    func migratePassword(_ password: String, for accountId: UUID) throws {
        // Only migrate if not already in Keychain
        guard !hasPassword(for: accountId) else { return }
        try savePassword(password, for: accountId)
    }

    // MARK: - OAuth Token Management (with custom service)

    /// Save password/token to Keychain with custom service
    func savePassword(_ password: String, for accountId: UUID, service customService: String) throws {
        let account = accountId.uuidString
        guard let passwordData = password.data(using: .utf8) else {
            throw KeychainError.encodingFailed
        }

        // Delete any existing password first
        try? deletePassword(for: accountId, service: customService)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: customService,
            kSecAttrAccount as String: account,
            kSecValueData as String: passwordData,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else {
            throw KeychainError.saveFailed(status)
        }
    }

    /// Retrieve password/token from Keychain with custom service
    func getPassword(for accountId: UUID, service customService: String) throws -> String {
        let account = accountId.uuidString

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: customService,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let passwordData = result as? Data,
              let password = String(data: passwordData, encoding: .utf8) else {
            throw KeychainError.notFound
        }

        return password
    }

    /// Delete password/token from Keychain with custom service
    func deletePassword(for accountId: UUID, service customService: String) throws {
        let account = accountId.uuidString

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: customService,
            kSecAttrAccount as String: account
        ]

        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }
}

// MARK: - Errors

enum KeychainError: LocalizedError {
    case encodingFailed
    case saveFailed(OSStatus)
    case notFound
    case deleteFailed(OSStatus)

    var errorDescription: String? {
        switch self {
        case .encodingFailed:
            return "Failed to encode password"
        case .saveFailed(let status):
            return "Failed to save to Keychain (status: \(status))"
        case .notFound:
            return "Password not found in Keychain"
        case .deleteFailed(let status):
            return "Failed to delete from Keychain (status: \(status))"
        }
    }
}
