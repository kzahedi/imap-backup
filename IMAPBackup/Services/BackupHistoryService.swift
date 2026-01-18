import Foundation

/// Service for managing backup history
@MainActor
class BackupHistoryService: ObservableObject {
    static let shared = BackupHistoryService()

    @Published private(set) var entries: [BackupHistoryEntry] = []

    private let maxEntries = 100
    private let historyKey = "BackupHistory"

    private init() {
        loadHistory()
    }

    // MARK: - History Management

    func startEntry(for accountEmail: String) -> UUID {
        let entry = BackupHistoryEntry(accountEmail: accountEmail)
        entries.insert(entry, at: 0)
        trimOldEntries()
        saveHistory()
        return entry.id
    }

    func updateEntry(
        id: UUID,
        emailsDownloaded: Int? = nil,
        totalEmails: Int? = nil,
        bytesDownloaded: Int64? = nil,
        foldersProcessed: Int? = nil,
        error: String? = nil
    ) {
        guard let index = entries.firstIndex(where: { $0.id == id }) else { return }

        if let emails = emailsDownloaded {
            entries[index].emailsDownloaded = emails
        }
        if let total = totalEmails {
            entries[index].totalEmails = total
        }
        if let bytes = bytesDownloaded {
            entries[index].bytesDownloaded = bytes
        }
        if let folders = foldersProcessed {
            entries[index].foldersProcessed = folders
        }
        if let err = error {
            entries[index].errors.append(err)
        }
    }

    func completeEntry(id: UUID, status: BackupHistoryStatus) {
        guard let index = entries.firstIndex(where: { $0.id == id }) else { return }

        entries[index].endTime = Date()
        entries[index].status = status
        saveHistory()
    }

    func clearHistory() {
        entries.removeAll()
        saveHistory()
    }

    func entriesForAccount(_ email: String) -> [BackupHistoryEntry] {
        entries.filter { $0.accountEmail == email }
    }

    // MARK: - Persistence

    private func loadHistory() {
        if let data = UserDefaults.standard.data(forKey: historyKey),
           let decoded = try? JSONDecoder().decode([BackupHistoryEntry].self, from: data) {
            entries = decoded
        }
    }

    private func saveHistory() {
        if let encoded = try? JSONEncoder().encode(entries) {
            UserDefaults.standard.set(encoded, forKey: historyKey)
        }
    }

    private func trimOldEntries() {
        if entries.count > maxEntries {
            entries = Array(entries.prefix(maxEntries))
        }
    }
}
