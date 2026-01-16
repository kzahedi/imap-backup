import Foundation
import SwiftUI

/// Main backup manager that coordinates backup operations
@MainActor
class BackupManager: ObservableObject {
    @Published var accounts: [EmailAccount] = []
    @Published var progress: [UUID: BackupProgress] = [:]
    @Published var isBackingUp = false
    @Published var backupLocation: URL

    private var activeTasks: [UUID: Task<Void, Never>] = [:]
    private let accountsKey = "EmailAccounts"

    init() {
        // Set default backup location
        let documentsURL = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        self.backupLocation = documentsURL.appendingPathComponent("IMAPBackup")

        // Load saved accounts
        loadAccounts()

        // Create backup directory
        try? FileManager.default.createDirectory(at: backupLocation, withIntermediateDirectories: true)
    }

    // MARK: - Account Management

    func addAccount(_ account: EmailAccount) {
        accounts.append(account)
        saveAccounts()
    }

    func removeAccount(_ account: EmailAccount) {
        accounts.removeAll { $0.id == account.id }
        saveAccounts()
    }

    func updateAccount(_ account: EmailAccount) {
        if let index = accounts.firstIndex(where: { $0.id == account.id }) {
            accounts[index] = account
            saveAccounts()
        }
    }

    private func loadAccounts() {
        if let data = UserDefaults.standard.data(forKey: accountsKey),
           let decoded = try? JSONDecoder().decode([EmailAccount].self, from: data) {
            accounts = decoded
        }
    }

    private func saveAccounts() {
        if let encoded = try? JSONEncoder().encode(accounts) {
            UserDefaults.standard.set(encoded, forKey: accountsKey)
        }
    }

    // MARK: - Backup Operations

    func startBackup(for account: EmailAccount) {
        guard activeTasks[account.id] == nil else { return }

        isBackingUp = true
        progress[account.id] = BackupProgress(accountId: account.id)

        activeTasks[account.id] = Task {
            await performBackup(for: account)
        }
    }

    func startBackupAll() {
        for account in accounts where account.isEnabled {
            startBackup(for: account)
        }
    }

    func cancelBackup(for accountId: UUID) {
        activeTasks[accountId]?.cancel()
        activeTasks.removeValue(forKey: accountId)
        progress[accountId]?.status = .cancelled
        updateIsBackingUp()
    }

    func cancelAllBackups() {
        for (id, task) in activeTasks {
            task.cancel()
            progress[id]?.status = .cancelled
        }
        activeTasks.removeAll()
        isBackingUp = false
    }

    private func updateIsBackingUp() {
        isBackingUp = !activeTasks.isEmpty
    }

    // MARK: - Backup Execution

    private func performBackup(for account: EmailAccount) async {
        let imapService = IMAPService(account: account)
        let storageService = StorageService(baseURL: backupLocation)

        do {
            // Connect
            updateProgress(for: account.id) { $0.status = .connecting }
            try await imapService.connect()
            try await imapService.login()

            // Fetch folders
            updateProgress(for: account.id) { $0.status = .fetchingFolders }
            let folders = try await imapService.listFolders()
            let selectableFolders = folders.filter { $0.isSelectable }

            updateProgress(for: account.id) {
                $0.totalFolders = selectableFolders.count
            }

            // Process each folder
            for (index, folder) in selectableFolders.enumerated() {
                guard !Task.isCancelled else { break }

                updateProgress(for: account.id) {
                    $0.currentFolder = folder.name
                    $0.processedFolders = index
                }

                try await backupFolder(
                    folder: folder,
                    account: account,
                    imapService: imapService,
                    storageService: storageService
                )
            }

            // Complete
            updateProgress(for: account.id) {
                $0.status = .completed
                $0.processedFolders = selectableFolders.count
            }

            // Update last backup date
            var updatedAccount = account
            updatedAccount.lastBackupDate = Date()
            updateAccount(updatedAccount)

            try await imapService.logout()

        } catch {
            updateProgress(for: account.id) {
                $0.status = .failed
                $0.errors.append(BackupError(message: error.localizedDescription))
            }
        }

        activeTasks.removeValue(forKey: account.id)
        updateIsBackingUp()
    }

    private func backupFolder(
        folder: IMAPFolder,
        account: EmailAccount,
        imapService: IMAPService,
        storageService: StorageService
    ) async throws {
        // Select folder
        let status = try await imapService.selectFolder(folder.name)

        updateProgress(for: account.id) {
            $0.totalEmails += status.exists
        }

        guard status.exists > 0 else { return }

        // Search for all emails
        updateProgress(for: account.id) { $0.status = .scanning }
        let uids = try await imapService.searchUnseen()

        // Download each email
        updateProgress(for: account.id) { $0.status = .downloading }

        for uid in uids {
            guard !Task.isCancelled else { break }

            do {
                let emailData = try await imapService.fetchEmail(uid: uid)

                // Parse email for metadata (simplified)
                let email = Email(
                    messageId: UUID().uuidString, // Would parse from headers
                    uid: uid,
                    folder: folder.path,
                    subject: "Email \(uid)", // Would parse from headers
                    sender: "Unknown", // Would parse from headers
                    senderEmail: "",
                    date: Date()
                )

                // Save to disk
                _ = try await storageService.saveEmail(
                    emailData,
                    email: email,
                    accountEmail: account.email,
                    folderPath: folder.path
                )

                updateProgress(for: account.id) {
                    $0.downloadedEmails += 1
                    $0.bytesDownloaded += Int64(emailData.count)
                    $0.currentEmailSubject = email.subject
                }

            } catch {
                updateProgress(for: account.id) {
                    $0.errors.append(BackupError(
                        message: error.localizedDescription,
                        folder: folder.name,
                        email: "UID: \(uid)"
                    ))
                }
            }
        }
    }

    private func updateProgress(for accountId: UUID, update: (inout BackupProgress) -> Void) {
        if var current = progress[accountId] {
            update(&current)
            progress[accountId] = current
        }
    }

    // MARK: - Backup Location

    func setBackupLocation(_ url: URL) {
        backupLocation = url
        UserDefaults.standard.set(url.path, forKey: "BackupLocation")
        try? FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
    }

    func selectBackupLocation() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.canCreateDirectories = true
        panel.allowsMultipleSelection = false
        panel.message = "Choose backup location"

        if panel.runModal() == .OK, let url = panel.url {
            setBackupLocation(url)
        }
    }
}
