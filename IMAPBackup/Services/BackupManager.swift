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

    /// Threshold above which emails are streamed directly to disk (in bytes)
    /// Default: 10 MB
    @Published var streamingThresholdBytes: Int = 10 * 1024 * 1024

    private var activeTasks: [UUID: Task<Void, Never>] = [:]
    private var activeHistoryIds: [UUID: UUID] = [:]  // Account ID -> History Entry ID
    private var scheduleTimer: Timer?
    private let accountsKey = "EmailAccounts"
    private let scheduleKey = "BackupSchedule"
    private let scheduleTimeKey = "BackupScheduleTime"
    private let backupLocationKey = "BackupLocation"
    private let streamingThresholdKey = "StreamingThresholdBytes"

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

        // Load streaming threshold
        if UserDefaults.standard.object(forKey: streamingThresholdKey) != nil {
            streamingThresholdBytes = UserDefaults.standard.integer(forKey: streamingThresholdKey)
        }

        // Create backup directory
        try? FileManager.default.createDirectory(at: backupLocation, withIntermediateDirectories: true)

        // Clean up any incomplete downloads from previous sessions
        Task {
            let storageService = StorageService(baseURL: backupLocation)
            if let cleaned = try? await storageService.cleanupIncompleteDownloads(), cleaned > 0 {
                print("Cleaned up \(cleaned) incomplete download(s)")
            }
        }

        // Initialize notification service
        NotificationService.shared.setupNotificationCategories()

        // Start scheduler if needed
        updateScheduler()
    }

    // MARK: - Account Management

    func addAccount(_ account: EmailAccount, password: String) {
        accounts.append(account)
        saveAccounts()
        // Save password to Keychain
        Task {
            try? await KeychainService.shared.savePassword(password, for: account.id)
        }
    }

    func removeAccount(_ account: EmailAccount) {
        accounts.removeAll { $0.id == account.id }
        saveAccounts()
        // Remove password from Keychain
        Task {
            try? await KeychainService.shared.deletePassword(for: account.id)
        }
    }

    func updateAccount(_ account: EmailAccount, password: String? = nil) {
        if let index = accounts.firstIndex(where: { $0.id == account.id }) {
            accounts[index] = account
            saveAccounts()
            // Update password in Keychain if provided
            if let password = password {
                Task {
                    try? await KeychainService.shared.savePassword(password, for: account.id)
                }
            }
        }
    }

    private func loadAccounts() {
        if let data = UserDefaults.standard.data(forKey: accountsKey),
           let decoded = try? JSONDecoder().decode([EmailAccount].self, from: data) {
            accounts = decoded
        }

        // Uncomment to add a test account for development
        // #if DEBUG
        // if accounts.isEmpty {
        //     let testAccount = EmailAccount.gmail(
        //         email: "your-email@gmail.com",
        //         appPassword: "your-app-password"
        //     )
        //     accounts.append(testAccount)
        // }
        // #endif
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

        // Mark history entry as cancelled
        if let historyId = activeHistoryIds[accountId] {
            BackupHistoryService.shared.completeEntry(id: historyId, status: .cancelled)
            activeHistoryIds.removeValue(forKey: accountId)
        }

        updateIsBackingUp()
    }

    func cancelAllBackups() {
        for (id, task) in activeTasks {
            task.cancel()
            progress[id]?.status = .cancelled

            // Mark history entry as cancelled
            if let historyId = activeHistoryIds[id] {
                BackupHistoryService.shared.completeEntry(id: historyId, status: .cancelled)
            }
        }
        activeTasks.removeAll()
        activeHistoryIds.removeAll()
        isBackingUp = false
    }

    private func updateIsBackingUp() {
        isBackingUp = !activeTasks.isEmpty
    }

    private func checkAllBackupsComplete() {
        // Only send summary if no more active tasks and we had multiple accounts
        guard activeTasks.isEmpty else { return }

        let completedCount = progress.values.filter {
            $0.status == .completed || $0.status == .failed
        }.count

        guard completedCount > 1 else { return }

        var totalDownloaded = 0
        var totalErrors = 0

        for (_, prog) in progress {
            totalDownloaded += prog.downloadedEmails
            totalErrors += prog.errors.count
        }

        NotificationService.shared.notifyAllBackupsCompleted(
            totalAccounts: completedCount,
            totalDownloaded: totalDownloaded,
            totalErrors: totalErrors
        )

        // Apply retention policies after all backups complete
        Task {
            let result = await RetentionService.shared.applyRetentionToAll(backupLocation: backupLocation)
            if result.filesDeleted > 0 {
                logInfo("Retention policy applied: deleted \(result.filesDeleted) files, freed \(result.bytesFreedFormatted)")
            }
        }
    }

    // MARK: - Backup Execution

    private func performBackup(for account: EmailAccount) async {
        let imapService = IMAPService(account: account)
        let storageService = StorageService(baseURL: backupLocation)

        // Configure rate limiting
        let rateLimitSettings = RateLimitService.shared.getSettings(for: account.id)
        await imapService.configureRateLimit(settings: rateLimitSettings)
        logDebug("Rate limiting configured: \(rateLimitSettings.requestDelayMs)ms delay, enabled: \(rateLimitSettings.isEnabled)")

        // Start history entry
        let historyId = BackupHistoryService.shared.startEntry(for: account.email)
        activeHistoryIds[account.id] = historyId

        logInfo("Starting backup for account: \(account.email)")

        do {
            // Connect
            updateProgress(for: account.id) { $0.status = .connecting }
            logDebug("Connecting to \(account.imapServer):\(account.port)")
            try await imapService.connect()
            try await imapService.login()
            logInfo("Connected and authenticated to \(account.imapServer)")

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

            // Update and complete history entry
            if let finalProgress = progress[account.id] {
                logInfo("Backup completed for \(account.email): \(finalProgress.downloadedEmails) emails downloaded, \(finalProgress.errors.count) errors")

                BackupHistoryService.shared.updateEntry(
                    id: historyId,
                    emailsDownloaded: finalProgress.downloadedEmails,
                    totalEmails: finalProgress.totalEmails,
                    bytesDownloaded: finalProgress.bytesDownloaded,
                    foldersProcessed: finalProgress.processedFolders
                )

                let historyStatus: BackupHistoryStatus = finalProgress.errors.isEmpty ? .completed : .completedWithErrors
                for error in finalProgress.errors {
                    logWarning("Backup error for \(account.email): \(error.message)")
                    BackupHistoryService.shared.updateEntry(id: historyId, error: error.message)
                }
                BackupHistoryService.shared.completeEntry(id: historyId, status: historyStatus)

                // Send completion notification
                NotificationService.shared.notifyBackupCompleted(
                    account: account.email,
                    emailsDownloaded: finalProgress.downloadedEmails,
                    totalEmails: finalProgress.totalEmails,
                    errors: finalProgress.errors.count
                )
            }

        } catch {
            logError("Backup failed for \(account.email): \(error.localizedDescription)")

            updateProgress(for: account.id) {
                $0.status = .failed
                $0.errors.append(BackupError(message: error.localizedDescription))
            }

            // Complete history entry with failure
            BackupHistoryService.shared.updateEntry(id: historyId, error: error.localizedDescription)
            BackupHistoryService.shared.completeEntry(id: historyId, status: .failed)

            // Send failure notification
            NotificationService.shared.notifyBackupFailed(
                account: account.email,
                error: error.localizedDescription
            )
        }

        activeTasks.removeValue(forKey: account.id)
        activeHistoryIds.removeValue(forKey: account.id)
        updateIsBackingUp()

        // Check if all backups are complete for summary notification
        checkAllBackupsComplete()
    }

    private func backupFolder(
        folder: IMAPFolder,
        account: EmailAccount,
        imapService: IMAPService,
        storageService: StorageService
    ) async throws {
        // Select folder
        let status = try await imapService.selectFolder(folder.name)

        guard status.exists > 0 else { return }

        // Search for all emails
        updateProgress(for: account.id) { $0.status = .scanning }
        let allUIDs = try await imapService.searchAll()

        // Get already backed up UIDs by scanning existing files (no database needed)
        let backedUpUIDs = (try? await storageService.getExistingUIDs(
            accountEmail: account.email,
            folderPath: folder.path
        )) ?? []

        // Filter to only new emails
        let newUIDs = allUIDs.filter { !backedUpUIDs.contains($0) }

        updateProgress(for: account.id) {
            $0.totalEmails += newUIDs.count
        }

        guard !newUIDs.isEmpty else { return }

        // Download each new email with retry logic
        updateProgress(for: account.id) { $0.status = .downloading }

        for uid in newUIDs {
            guard !Task.isCancelled else { break }

            // Retry with exponential backoff (max 3 attempts)
            var lastError: Error?
            for attempt in 1...3 {
                do {
                    // Check email size first to decide whether to stream
                    let emailSize = try await imapService.fetchEmailSize(uid: uid)
                    let useStreaming = emailSize > streamingThresholdBytes

                    var bytesDownloaded: Int64 = 0
                    var email: Email
                    var parsed: ParsedEmail?

                    if useStreaming {
                        // Stream large email directly to disk
                        logInfo("Streaming large email (UID: \(uid), size: \(ByteCountFormatter.string(fromByteCount: Int64(emailSize), countStyle: .file)))")

                        // Create placeholder email for filename
                        email = Email(
                            messageId: UUID().uuidString,
                            uid: uid,
                            folder: folder.path,
                            subject: "(Streaming)",
                            sender: "Unknown",
                            senderEmail: "",
                            date: Date()
                        )

                        let (tempURL, finalURL) = try await storageService.prepareStreamingDestination(
                            email: email,
                            accountEmail: account.email,
                            folderPath: folder.path
                        )

                        // Stream directly to disk
                        bytesDownloaded = try await imapService.streamEmailToFile(uid: uid, destinationURL: tempURL)

                        // Move to final location
                        try await storageService.finalizeStreamedFile(tempURL: tempURL, finalURL: finalURL)

                        // Read headers from saved file for metadata
                        if let headerContent = await storageService.readEmailHeaders(at: finalURL) {
                            if let headerData = headerContent.data(using: .utf8) {
                                parsed = EmailParser.parseMetadata(from: headerData)
                            }
                        }

                        // Update email with parsed metadata (file is already saved with placeholder name)
                        // In streaming mode, we keep the placeholder filename but log the actual subject
                        if let p = parsed {
                            logDebug("Streamed email: \(p.subject ?? "(No Subject)") from \(p.senderEmail ?? "unknown")")
                        }

                    } else {
                        // Normal in-memory download for smaller emails
                        let emailData = try await imapService.fetchEmail(uid: uid)
                        bytesDownloaded = Int64(emailData.count)

                        // Verify download - check for valid email structure
                        guard emailData.count > 0,
                              let content = String(data: emailData, encoding: .utf8) ?? String(data: emailData, encoding: .ascii),
                              content.contains("From:") || content.contains("Date:") || content.contains("Subject:") else {
                            throw BackupManagerError.invalidEmailData
                        }

                        // Parse email headers to get metadata
                        parsed = EmailParser.parseMetadata(from: emailData)

                        let messageId = parsed?.messageId ?? UUID().uuidString
                        email = Email(
                            messageId: messageId,
                            uid: uid,
                            folder: folder.path,
                            subject: parsed?.subject ?? "(No Subject)",
                            sender: parsed?.senderName ?? "Unknown",
                            senderEmail: parsed?.senderEmail ?? "",
                            date: parsed?.date ?? Date()
                        )

                        // Save to disk (file existence = backup record, no database needed)
                        let savedURL = try await storageService.saveEmail(
                            emailData,
                            email: email,
                            accountEmail: account.email,
                            folderPath: folder.path
                        )

                        // Extract attachments if enabled
                        if AttachmentExtractionManager.shared.settings.isEnabled {
                            await extractAttachments(
                                from: emailData,
                                emailURL: savedURL,
                                accountEmail: account.email,
                                folderPath: folder.path,
                                storageService: storageService
                            )
                        }
                    }

                    updateProgress(for: account.id) {
                        $0.downloadedEmails += 1
                        $0.bytesDownloaded += bytesDownloaded
                        $0.currentEmailSubject = parsed?.subject ?? "(No Subject)"
                    }

                    lastError = nil
                    break // Success, exit retry loop

                } catch {
                    lastError = error
                    if attempt < 3 {
                        // Exponential backoff: 1s, 2s, 4s
                        let delay = UInt64(pow(2.0, Double(attempt - 1))) * 1_000_000_000
                        try? await Task.sleep(nanoseconds: delay)
                    }
                }
            }

            // Record error after all retries failed
            if let error = lastError {
                updateProgress(for: account.id) {
                    $0.errors.append(BackupError(
                        message: "Failed after 3 attempts: \(error.localizedDescription)",
                        folder: folder.name,
                        email: "UID: \(uid)"
                    ))
                }
            }
        }
    }

    // MARK: - Attachment Extraction

    private func extractAttachments(
        from emailData: Data,
        emailURL: URL,
        accountEmail: String,
        folderPath: String,
        storageService: StorageService
    ) async {
        let attachmentService = AttachmentService()
        let attachments = await attachmentService.extractAttachments(from: emailData)

        guard !attachments.isEmpty else { return }

        // Create attachment folder (same name as email file without extension)
        let emailFilename = emailURL.deletingPathExtension().lastPathComponent
        let attachmentFolderURL = emailURL.deletingLastPathComponent().appendingPathComponent("\(emailFilename)_attachments")

        do {
            let savedURLs = try await attachmentService.saveAttachments(attachments, to: attachmentFolderURL)
            if !savedURLs.isEmpty {
                logDebug("Extracted \(savedURLs.count) attachment(s) from \(emailFilename)")
            }
        } catch {
            logWarning("Failed to extract attachments from \(emailFilename): \(error.localizedDescription)")
        }
    }

    // MARK: - Errors

    enum BackupManagerError: LocalizedError {
        case invalidEmailData

        var errorDescription: String? {
            switch self {
            case .invalidEmailData:
                return "Downloaded data does not appear to be a valid email"
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

    /// Set the streaming threshold for large attachments
    func setStreamingThreshold(_ bytes: Int) {
        streamingThresholdBytes = bytes
        UserDefaults.standard.set(bytes, forKey: streamingThresholdKey)
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
