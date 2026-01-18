import Foundation

/// Result of verifying a single folder
struct FolderVerificationResult {
    let folderName: String
    let serverUIDs: Set<UInt32>
    let localUIDs: Set<UInt32>

    /// UIDs on server but not backed up locally
    var missingLocally: Set<UInt32> {
        serverUIDs.subtracting(localUIDs)
    }

    /// UIDs backed up locally but no longer on server (deleted or moved)
    var deletedOnServer: Set<UInt32> {
        localUIDs.subtracting(serverUIDs)
    }

    /// UIDs that exist both locally and on server
    var synced: Set<UInt32> {
        serverUIDs.intersection(localUIDs)
    }

    var isFullySynced: Bool {
        missingLocally.isEmpty && deletedOnServer.isEmpty
    }

    var summary: String {
        if isFullySynced {
            return "✓ Fully synced (\(synced.count) emails)"
        } else {
            var parts: [String] = []
            if !missingLocally.isEmpty {
                parts.append("\(missingLocally.count) missing locally")
            }
            if !deletedOnServer.isEmpty {
                parts.append("\(deletedOnServer.count) deleted on server")
            }
            return "⚠ " + parts.joined(separator: ", ")
        }
    }
}

/// Result of verifying an entire account
struct AccountVerificationResult: Identifiable {
    let id = UUID()
    let accountEmail: String
    let folderResults: [FolderVerificationResult]
    let verifiedAt: Date

    var totalServerEmails: Int {
        folderResults.reduce(0) { $0 + $1.serverUIDs.count }
    }

    var totalLocalEmails: Int {
        folderResults.reduce(0) { $0 + $1.localUIDs.count }
    }

    var totalMissingLocally: Int {
        folderResults.reduce(0) { $0 + $1.missingLocally.count }
    }

    var totalDeletedOnServer: Int {
        folderResults.reduce(0) { $0 + $1.deletedOnServer.count }
    }

    var isFullySynced: Bool {
        folderResults.allSatisfy { $0.isFullySynced }
    }

    var summary: String {
        if isFullySynced {
            return "✓ All \(folderResults.count) folders fully synced"
        } else {
            var parts: [String] = []
            if totalMissingLocally > 0 {
                parts.append("\(totalMissingLocally) emails missing locally")
            }
            if totalDeletedOnServer > 0 {
                parts.append("\(totalDeletedOnServer) emails deleted on server")
            }
            return "⚠ " + parts.joined(separator: ", ")
        }
    }
}

/// Service for verifying backup integrity against server state
@MainActor
class VerificationService: ObservableObject {
    static let shared = VerificationService()

    @Published var isVerifying = false
    @Published var currentAccount: String?
    @Published var currentFolder: String?
    @Published var lastResults: [AccountVerificationResult] = []

    private init() {}

    /// Verify all accounts
    func verifyAll(accounts: [EmailAccount], backupLocation: URL) async -> [AccountVerificationResult] {
        isVerifying = true
        var results: [AccountVerificationResult] = []

        for account in accounts where account.isEnabled {
            if let result = await verifyAccount(account, backupLocation: backupLocation) {
                results.append(result)
            }
        }

        lastResults = results
        isVerifying = false
        currentAccount = nil
        currentFolder = nil

        return results
    }

    /// Verify a single account
    func verifyAccount(_ account: EmailAccount, backupLocation: URL) async -> AccountVerificationResult? {
        currentAccount = account.email
        logInfo("Starting verification for account: \(account.email)")

        let imapService = IMAPService(account: account)
        let storageService = StorageService(baseURL: backupLocation)

        do {
            // Connect to server
            try await imapService.connect()
            try await imapService.login()

            // Get folder list
            let folders = try await imapService.listFolders()
            let selectableFolders = folders.filter { $0.isSelectable }

            var folderResults: [FolderVerificationResult] = []

            for folder in selectableFolders {
                currentFolder = folder.name

                // Get server UIDs
                _ = try await imapService.selectFolder(folder.name)
                let serverUIDs = try await imapService.searchAll()

                // Get local UIDs
                let localUIDs = (try? await storageService.getExistingUIDs(
                    accountEmail: account.email,
                    folderPath: folder.path
                )) ?? []

                let result = FolderVerificationResult(
                    folderName: folder.name,
                    serverUIDs: Set(serverUIDs),
                    localUIDs: localUIDs
                )

                folderResults.append(result)

                if !result.isFullySynced {
                    logDebug("Folder \(folder.name): \(result.summary)")
                }
            }

            try await imapService.logout()

            let accountResult = AccountVerificationResult(
                accountEmail: account.email,
                folderResults: folderResults,
                verifiedAt: Date()
            )

            logInfo("Verification complete for \(account.email): \(accountResult.summary)")

            return accountResult

        } catch {
            logError("Verification failed for \(account.email): \(error.localizedDescription)")
            return nil
        }
    }

    /// Clear last results
    func clearResults() {
        lastResults = []
    }
}
