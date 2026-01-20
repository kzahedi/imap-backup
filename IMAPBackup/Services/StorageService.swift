import Foundation

/// Service for storing emails and attachments to disk
actor StorageService {
    private let baseURL: URL
    private let fileManager = FileManager.default

    /// Cache file name for storing UIDs (hidden file)
    private let uidCacheFilename = ".uid_cache"

    init(baseURL: URL) {
        self.baseURL = baseURL
    }

    // MARK: - UID Cache Management

    /// Get the UID cache file URL for a folder
    private func uidCacheURL(for folderURL: URL) -> URL {
        folderURL.appendingPathComponent(uidCacheFilename)
    }

    /// Append a UID to the cache file
    private func appendUIDToCache(_ uid: UInt32, folderURL: URL) {
        let cacheURL = uidCacheURL(for: folderURL)
        let line = "\(uid)\n"

        if let data = line.data(using: .utf8) {
            if fileManager.fileExists(atPath: cacheURL.path) {
                // Append to existing file
                if let handle = try? FileHandle(forWritingTo: cacheURL) {
                    handle.seekToEndOfFile()
                    handle.write(data)
                    try? handle.close()
                }
            } else {
                // Create new file
                try? data.write(to: cacheURL)
            }
        }
    }

    /// Read UIDs from cache file (O(1) file read instead of O(n) directory scan)
    private func readUIDsFromCache(folderURL: URL) -> Set<UInt32>? {
        let cacheURL = uidCacheURL(for: folderURL)

        guard let content = try? String(contentsOf: cacheURL, encoding: .utf8) else {
            return nil
        }

        var uids = Set<UInt32>()
        for line in content.components(separatedBy: .newlines) {
            if let uid = UInt32(line.trimmingCharacters(in: .whitespaces)) {
                uids.insert(uid)
            }
        }
        return uids
    }

    /// Rebuild UID cache from existing files (migration for existing backups)
    func rebuildUIDCache(accountEmail: String, folderPath: String) throws {
        let sanitizedEmail = accountEmail.sanitizedForFilename()
        let sanitizedPath = folderPath
            .components(separatedBy: "/")
            .map { $0.sanitizedForFilename() }
            .joined(separator: "/")

        let folderURL = baseURL
            .appendingPathComponent(sanitizedEmail)
            .appendingPathComponent(sanitizedPath)

        guard fileManager.fileExists(atPath: folderURL.path) else { return }

        // Scan files and build cache
        let contents = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
        var uids: [UInt32] = []

        for fileURL in contents where fileURL.pathExtension == "eml" {
            let filename = fileURL.deletingPathExtension().lastPathComponent
            if let firstUnderscore = filename.firstIndex(of: "_"),
               let uid = UInt32(filename[..<firstUnderscore]) {
                uids.append(uid)
            }
        }

        // Write cache file
        let cacheURL = uidCacheURL(for: folderURL)
        let content = uids.map { String($0) }.joined(separator: "\n") + (uids.isEmpty ? "" : "\n")
        try content.write(to: cacheURL, atomically: true, encoding: .utf8)
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

        // Append UID to cache for O(1) lookup on next backup
        appendUIDToCache(email.uid, folderURL: folderURL)

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
    func finalizeStreamedFile(tempURL: URL, finalURL: URL, uid: UInt32? = nil) throws {
        if fileManager.fileExists(atPath: finalURL.path) {
            try fileManager.removeItem(at: finalURL)
        }
        try fileManager.moveItem(at: tempURL, to: finalURL)

        // Append UID to cache for O(1) lookup on next backup
        if let uid = uid {
            let folderURL = finalURL.deletingLastPathComponent()
            appendUIDToCache(uid, folderURL: folderURL)
        }
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

    /// Get UIDs of already downloaded emails
    /// Uses cache file for O(1) lookup, falls back to O(n) file scan if cache missing
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

        // Try to read from cache first (fast path)
        if let cachedUIDs = readUIDsFromCache(folderURL: folderURL) {
            return cachedUIDs
        }

        // Cache miss - fall back to file scan (slow path, builds cache)
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

        // Build cache for next time
        let cacheURL = uidCacheURL(for: folderURL)
        let content = uids.map { String($0) }.joined(separator: "\n") + (uids.isEmpty ? "" : "\n")
        try? content.write(to: cacheURL, atomically: true, encoding: .utf8)

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
