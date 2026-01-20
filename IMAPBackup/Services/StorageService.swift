import Foundation

/// Service for storing emails and attachments to disk
actor StorageService {
    private let baseURL: URL
    private let fileManager = FileManager.default

    init(baseURL: URL) {
        self.baseURL = baseURL
    }

    // MARK: - Directory Management

    func createAccountDirectory(email: String) throws -> URL {
        let sanitizedEmail = email.sanitizedForFilename()
        let accountURL = baseURL.appendingPathComponent(sanitizedEmail)

        if !fileManager.fileExists(atPath: accountURL.path) {
            try fileManager.createDirectory(at: accountURL, withIntermediateDirectories: true)
        }

        return accountURL
    }

    func createFolderDirectory(accountEmail: String, folderPath: String) throws -> URL {
        let accountURL = try createAccountDirectory(email: accountEmail)

        // Convert IMAP folder path to filesystem path
        // e.g., "Work/Projects/Alpha" -> "Work/Projects/Alpha"
        let sanitizedPath = folderPath
            .components(separatedBy: "/")
            .map { $0.sanitizedForFilename() }
            .joined(separator: "/")

        let folderURL = accountURL.appendingPathComponent(sanitizedPath)

        if !fileManager.fileExists(atPath: folderURL.path) {
            try fileManager.createDirectory(at: folderURL, withIntermediateDirectories: true)
        }

        return folderURL
    }

    // MARK: - Email Storage

    /// Save email with atomic write to prevent partial files from interrupted downloads
    func saveEmail(_ emailData: Data, email: Email, accountEmail: String, folderPath: String) throws -> URL {
        let folderURL = try createFolderDirectory(accountEmail: accountEmail, folderPath: folderPath)
        let filename = email.filename()
        let fileURL = folderURL.appendingPathComponent(filename)

        // Check for duplicate filename and increment if needed
        let finalURL = uniqueFileURL(for: fileURL)

        // Write to temp file first, then atomically move to final location
        // This prevents partial files from interrupted downloads
        let tempURL = finalURL.appendingPathExtension("tmp")
        try emailData.write(to: tempURL)
        try fileManager.moveItem(at: tempURL, to: finalURL)

        return finalURL
    }

    /// Prepare a destination URL for streaming large emails directly to disk
    func prepareStreamingDestination(email: Email, accountEmail: String, folderPath: String) throws -> (tempURL: URL, finalURL: URL) {
        let folderURL = try createFolderDirectory(accountEmail: accountEmail, folderPath: folderPath)
        let filename = email.filename()
        let fileURL = folderURL.appendingPathComponent(filename)
        let finalURL = uniqueFileURL(for: fileURL)
        let tempURL = finalURL.appendingPathExtension("tmp")
        return (tempURL, finalURL)
    }

    /// Finalize a streamed file by moving from temp to final location
    func finalizeStreamedFile(tempURL: URL, finalURL: URL) throws {
        if fileManager.fileExists(atPath: finalURL.path) {
            try fileManager.removeItem(at: finalURL)
        }
        try fileManager.moveItem(at: tempURL, to: finalURL)
    }

    /// Read headers from a saved .eml file for metadata extraction
    func readEmailHeaders(at url: URL, maxBytes: Int = 32768) -> String? {
        guard let handle = FileHandle(forReadingAtPath: url.path) else { return nil }
        defer { try? handle.close() }

        let data = handle.readData(ofLength: maxBytes)
        return String(data: data, encoding: .utf8) ?? String(data: data, encoding: .ascii)
    }

    /// Clean up any orphaned temp files from interrupted downloads
    func cleanupIncompleteDownloads() throws -> Int {
        var cleanedCount = 0
        let enumerator = fileManager.enumerator(at: baseURL, includingPropertiesForKeys: nil)

        while let fileURL = enumerator?.nextObject() as? URL {
            if fileURL.pathExtension == "tmp" {
                try? fileManager.removeItem(at: fileURL)
                cleanedCount += 1
            }
        }

        return cleanedCount
    }

    func saveAttachment(_ data: Data, filename: String, email: Email, accountEmail: String, folderPath: String) throws -> URL {
        let folderURL = try createFolderDirectory(accountEmail: accountEmail, folderPath: folderPath)
        let attachmentFolderName = email.attachmentFolderName()
        let attachmentFolderURL = folderURL.appendingPathComponent(attachmentFolderName)

        if !fileManager.fileExists(atPath: attachmentFolderURL.path) {
            try fileManager.createDirectory(at: attachmentFolderURL, withIntermediateDirectories: true)
        }

        let sanitizedFilename = filename.sanitizedForFilename()
        let fileURL = attachmentFolderURL.appendingPathComponent(sanitizedFilename)
        let finalURL = uniqueFileURL(for: fileURL)

        // Write to temp file first, then atomically move to final location
        let tempURL = finalURL.appendingPathExtension("tmp")
        try data.write(to: tempURL)
        try fileManager.moveItem(at: tempURL, to: finalURL)

        return finalURL
    }

    // MARK: - Query Methods

    /// Get UIDs of already downloaded emails by scanning filenames
    /// Filename format: <UID>_<timestamp>_<sender>.eml
    func getExistingUIDs(accountEmail: String, folderPath: String) throws -> Set<UInt32> {
        let sanitizedEmail = accountEmail.sanitizedForFilename()
        let sanitizedPath = folderPath
            .components(separatedBy: "/")
            .map { $0.sanitizedForFilename() }
            .joined(separator: "/")

        let folderURL = baseURL
            .appendingPathComponent(sanitizedEmail)
            .appendingPathComponent(sanitizedPath)

        guard fileManager.fileExists(atPath: folderURL.path) else {
            return []
        }

        let contents = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
        var uids = Set<UInt32>()

        for fileURL in contents where fileURL.pathExtension == "eml" {
            let filename = fileURL.deletingPathExtension().lastPathComponent
            // Extract UID from start of filename (before first underscore)
            if let firstUnderscore = filename.firstIndex(of: "_"),
               let uid = UInt32(filename[..<firstUnderscore]) {
                uids.insert(uid)
            }
        }

        return uids
    }

    func emailExists(messageId: String, accountEmail: String, folderPath: String) throws -> Bool {
        // This is a simple check - in production, use the database
        let folderURL = try createFolderDirectory(accountEmail: accountEmail, folderPath: folderPath)
        let contents = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
        return contents.contains { $0.pathExtension == "eml" }
    }

    func getBackupSize(for accountEmail: String) throws -> Int64 {
        let accountURL = try createAccountDirectory(email: accountEmail)
        return try directorySize(at: accountURL)
    }

    func getEmailCount(for accountEmail: String) throws -> Int {
        let accountURL = try createAccountDirectory(email: accountEmail)
        return try countFiles(at: accountURL, withExtension: "eml")
    }

    // MARK: - Helpers

    private func uniqueFileURL(for url: URL) -> URL {
        var finalURL = url
        var counter = 1

        while fileManager.fileExists(atPath: finalURL.path) {
            let filename = url.deletingPathExtension().lastPathComponent
            let ext = url.pathExtension
            let newFilename = "\(filename)_\(counter).\(ext)"
            finalURL = url.deletingLastPathComponent().appendingPathComponent(newFilename)
            counter += 1
        }

        return finalURL
    }

    private func directorySize(at url: URL) throws -> Int64 {
        var totalSize: Int64 = 0
        let enumerator = fileManager.enumerator(at: url, includingPropertiesForKeys: [.fileSizeKey])

        while let fileURL = enumerator?.nextObject() as? URL {
            let attributes = try fileURL.resourceValues(forKeys: [.fileSizeKey])
            totalSize += Int64(attributes.fileSize ?? 0)
        }

        return totalSize
    }

    private func countFiles(at url: URL, withExtension ext: String) throws -> Int {
        var count = 0
        let enumerator = fileManager.enumerator(at: url, includingPropertiesForKeys: nil)

        while let fileURL = enumerator?.nextObject() as? URL {
            if fileURL.pathExtension == ext {
                count += 1
            }
        }

        return count
    }
}

// MARK: - Backup Location Manager

class BackupLocationManager: ObservableObject {
    @Published var backupURL: URL

    private let defaultsKey = "BackupLocation"

    init() {
        if let savedPath = UserDefaults.standard.string(forKey: defaultsKey),
           let url = URL(string: savedPath) {
            self.backupURL = url
        } else {
            // Default to Documents/IMAPBackup
            let documentsURL = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
            self.backupURL = documentsURL.appendingPathComponent("IMAPBackup")
        }

        // Ensure directory exists
        try? FileManager.default.createDirectory(at: backupURL, withIntermediateDirectories: true)
    }

    func setBackupLocation(_ url: URL) {
        backupURL = url
        UserDefaults.standard.set(url.absoluteString, forKey: defaultsKey)
        try? FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
    }
}
