import Foundation
import SwiftUI

/// Backup schedule options
enum BackupSchedule: String, Codable, CaseIterable {
    case manual = "Manual"
    case hourly = "Every Hour"
    case daily = "Daily"
    case weekly = "Weekly"

    var interval: TimeInterval? {
        switch self {
        case .manual: return nil
        case .hourly: return 3600
        case .daily: return 86400
        case .weekly: return 604800
        }
    }

    var needsTimeSelection: Bool {
        switch self {
        case .daily, .weekly: return true
        default: return false
        }
    }
}

/// Main backup manager that coordinates backup operations
@MainActor
class BackupManager: ObservableObject {
    @Published var accounts: [EmailAccount] = []
    @Published var progress: [UUID: BackupProgress] = [:]
    @Published var isBackingUp = false
    @Published var backupLocation: URL
    @Published var schedule: BackupSchedule = .manual
    @Published var scheduledTime: Date = Calendar.current.date(bySettingHour: 2, minute: 0, second: 0, of: Date()) ?? Date()
    @Published var nextScheduledBackup: Date?

    private var activeTasks: [UUID: Task<Void, Never>] = [:]
    private var scheduleTimer: Timer?
    private let accountsKey = "EmailAccounts"
    private let scheduleKey = "BackupSchedule"
    private let scheduleTimeKey = "BackupScheduleTime"
    private let backupLocationKey = "BackupLocation"

    init() {
        // Load backup location or set default
        if let savedPath = UserDefaults.standard.string(forKey: backupLocationKey) {
            self.backupLocation = URL(fileURLWithPath: savedPath)
        } else {
            let documentsURL = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
            self.backupLocation = documentsURL.appendingPathComponent("IMAPBackup")
        }

        // Load saved accounts and schedule
        loadAccounts()
        loadSchedule()

        // Create backup directory
        try? FileManager.default.createDirectory(at: backupLocation, withIntermediateDirectories: true)

        // Start scheduler if needed
        updateScheduler()
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

        // Add test account for development if no accounts exist
        #if DEBUG
        if accounts.isEmpty {
            let testAccount = EmailAccount.gmail(
                email: "wuce.brain.twitter@gmail.com",
                appPassword: "jpjx twes mhax ijft"
            )
            accounts.append(testAccount)
        }
        #endif
    }

    private func saveAccounts() {
        if let encoded = try? JSONEncoder().encode(accounts) {
            UserDefaults.standard.set(encoded, forKey: accountsKey)
        }
    }

    // MARK: - Scheduling

    private func loadSchedule() {
        if let savedSchedule = UserDefaults.standard.string(forKey: scheduleKey),
           let schedule = BackupSchedule(rawValue: savedSchedule) {
            self.schedule = schedule
        }

        if let savedTimeInterval = UserDefaults.standard.object(forKey: scheduleTimeKey) as? TimeInterval {
            self.scheduledTime = Date(timeIntervalSince1970: savedTimeInterval)
        }
    }

    func setSchedule(_ newSchedule: BackupSchedule) {
        schedule = newSchedule
        UserDefaults.standard.set(newSchedule.rawValue, forKey: scheduleKey)
        updateScheduler()
    }

    func setScheduledTime(_ time: Date) {
        scheduledTime = time
        UserDefaults.standard.set(time.timeIntervalSince1970, forKey: scheduleTimeKey)
        updateScheduler()
    }

    var scheduledTimeFormatted: String {
        let formatter = DateFormatter()
        formatter.timeStyle = .short
        return formatter.string(from: scheduledTime)
    }

    private func updateScheduler() {
        // Cancel existing timer
        scheduleTimer?.invalidate()
        scheduleTimer = nil
        nextScheduledBackup = nil

        guard schedule != .manual else { return }

        // Calculate next backup time
        nextScheduledBackup = calculateNextBackupTime()

        // Set up timer to check every minute if it's time to backup
        scheduleTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.checkScheduledBackup()
            }
        }
    }

    private func calculateNextBackupTime() -> Date? {
        let calendar = Calendar.current
        let now = Date()

        switch schedule {
        case .manual:
            return nil

        case .hourly:
            // Next hour
            return calendar.date(byAdding: .hour, value: 1, to: now)

        case .daily:
            // Today or tomorrow at scheduled time
            let hour = calendar.component(.hour, from: scheduledTime)
            let minute = calendar.component(.minute, from: scheduledTime)

            var components = calendar.dateComponents([.year, .month, .day], from: now)
            components.hour = hour
            components.minute = minute
            components.second = 0

            if let todayBackup = calendar.date(from: components), todayBackup > now {
                return todayBackup
            } else {
                // Tomorrow
                components.day! += 1
                return calendar.date(from: components)
            }

        case .weekly:
            // Next week at scheduled time (same weekday as now, or next week)
            let hour = calendar.component(.hour, from: scheduledTime)
            let minute = calendar.component(.minute, from: scheduledTime)

            var components = calendar.dateComponents([.year, .month, .day], from: now)
            components.hour = hour
            components.minute = minute
            components.second = 0

            if let thisWeek = calendar.date(from: components), thisWeek > now {
                return thisWeek
            } else {
                return calendar.date(byAdding: .day, value: 7, to: calendar.date(from: components) ?? now)
            }
        }
    }

    private func checkScheduledBackup() {
        guard !isBackingUp,
              let nextBackup = nextScheduledBackup,
              Date() >= nextBackup else { return }

        startBackupAll()

        // Calculate next backup time
        nextScheduledBackup = calculateNextBackupTime()
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

                // Parse email headers to get metadata
                let parsed = EmailParser.parseMetadata(from: emailData)

                let email = Email(
                    messageId: parsed?.messageId ?? UUID().uuidString,
                    uid: uid,
                    folder: folder.path,
                    subject: parsed?.subject ?? "(No Subject)",
                    sender: parsed?.senderName ?? "Unknown",
                    senderEmail: parsed?.senderEmail ?? "",
                    date: parsed?.date ?? Date()
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

    var isUsingICloud: Bool {
        backupLocation.path.contains("Mobile Documents") ||
        backupLocation.path.contains("iCloud")
    }

    var iCloudAvailable: Bool {
        FileManager.default.ubiquityIdentityToken != nil
    }

    var iCloudDriveURL: URL? {
        // iCloud Drive location for documents
        if let iCloudURL = FileManager.default.url(forUbiquityContainerIdentifier: nil) {
            return iCloudURL.appendingPathComponent("Documents").appendingPathComponent("IMAPBackup")
        }
        // Fallback to ~/Library/Mobile Documents/com~apple~CloudDocs/
        let homeDir = FileManager.default.homeDirectoryForCurrentUser
        let iCloudDocs = homeDir.appendingPathComponent("Library/Mobile Documents/com~apple~CloudDocs/IMAPBackup")
        return iCloudDocs
    }

    func setBackupLocation(_ url: URL) {
        backupLocation = url
        UserDefaults.standard.set(url.path, forKey: backupLocationKey)
        try? FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
    }

    func useICloudDrive() {
        guard let iCloudURL = iCloudDriveURL else { return }
        setBackupLocation(iCloudURL)
    }

    func useLocalStorage() {
        let documentsURL = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        let localURL = documentsURL.appendingPathComponent("IMAPBackup")
        setBackupLocation(localURL)
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

    // MARK: - Statistics

    struct AccountStats {
        var totalEmails: Int = 0
        var totalSize: Int64 = 0
        var folderCount: Int = 0
        var oldestEmail: Date?
        var newestEmail: Date?
    }

    struct GlobalStats {
        var totalEmails: Int = 0
        var totalSize: Int64 = 0
        var accountCount: Int = 0
    }

    func getStats(for account: EmailAccount) -> AccountStats {
        let accountDir = backupLocation.appendingPathComponent(account.email.sanitizedForFilename())
        return calculateStats(at: accountDir)
    }

    func getGlobalStats() -> GlobalStats {
        var global = GlobalStats()
        global.accountCount = accounts.count

        for account in accounts {
            let stats = getStats(for: account)
            global.totalEmails += stats.totalEmails
            global.totalSize += stats.totalSize
        }

        return global
    }

    private func calculateStats(at directory: URL) -> AccountStats {
        var stats = AccountStats()
        let fileManager = FileManager.default

        guard let enumerator = fileManager.enumerator(
            at: directory,
            includingPropertiesForKeys: [.fileSizeKey, .creationDateKey, .isRegularFileKey],
            options: [.skipsHiddenFiles]
        ) else {
            return stats
        }

        var folders = Set<String>()

        for case let fileURL as URL in enumerator {
            guard let resourceValues = try? fileURL.resourceValues(forKeys: [.fileSizeKey, .creationDateKey, .isRegularFileKey]),
                  resourceValues.isRegularFile == true,
                  fileURL.pathExtension == "eml" else {
                continue
            }

            stats.totalEmails += 1
            stats.totalSize += Int64(resourceValues.fileSize ?? 0)

            // Track folder
            let folderPath = fileURL.deletingLastPathComponent().path
            folders.insert(folderPath)

            // Track dates from filename (format: YYYYMMDD_HHMMSS_sender.eml)
            let filename = fileURL.deletingPathExtension().lastPathComponent
            if let date = parseDateFromFilename(filename) {
                if stats.oldestEmail == nil || date < stats.oldestEmail! {
                    stats.oldestEmail = date
                }
                if stats.newestEmail == nil || date > stats.newestEmail! {
                    stats.newestEmail = date
                }
            }
        }

        stats.folderCount = folders.count
        return stats
    }

    private func parseDateFromFilename(_ filename: String) -> Date? {
        // Format: YYYYMMDD_HHMMSS_sender
        let parts = filename.components(separatedBy: "_")
        guard parts.count >= 2,
              parts[0].count == 8,
              parts[1].count == 6 else {
            return nil
        }

        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyyMMdd_HHmmss"
        return dateFormatter.date(from: "\(parts[0])_\(parts[1])")
    }
}
