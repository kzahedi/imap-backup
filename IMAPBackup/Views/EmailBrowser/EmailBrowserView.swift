import SwiftUI
import AppKit

struct EmailBrowserView: View {
    @EnvironmentObject var backupManager: BackupManager
    @StateObject private var browserService = EmailBrowserService()

    @State private var selectedAccount: String?
    @State private var selectedFolder: String?
    @State private var selectedEmail: EmailFileInfo?
    @State private var searchText = ""

    var body: some View {
        NavigationSplitView {
            // Sidebar - Accounts and Folders
            List(selection: $selectedFolder) {
                ForEach(browserService.accounts, id: \.self) { account in
                    Section(header: Text(account)) {
                        ForEach(browserService.folders(for: account), id: \.self) { folder in
                            Label(folder, systemImage: folderIcon(for: folder))
                                .tag("\(account)/\(folder)")
                        }
                    }
                }
            }
            .listStyle(.sidebar)
            .frame(minWidth: 200)
        } content: {
            // Email list
            if let selection = selectedFolder {
                emailListView(for: selection)
            } else {
                ContentUnavailableView(
                    "Select a Folder",
                    systemImage: "folder",
                    description: Text("Choose a folder from the sidebar to view emails")
                )
            }
        } detail: {
            // Email preview
            if let email = selectedEmail {
                EmailPreviewView(email: email)
            } else {
                ContentUnavailableView(
                    "Select an Email",
                    systemImage: "envelope",
                    description: Text("Choose an email to preview its contents")
                )
            }
        }
        .searchable(text: $searchText, prompt: "Search emails...")
        .task {
            await browserService.loadAccounts(from: backupManager.backupLocation)
        }
        .onChange(of: selectedFolder) { _, newValue in
            selectedEmail = nil
            if let selection = newValue {
                let parts = selection.split(separator: "/", maxSplits: 1)
                if parts.count == 2 {
                    let account = String(parts[0])
                    let folder = String(parts[1])
                    Task {
                        await browserService.loadEmails(account: account, folder: folder, from: backupManager.backupLocation)
                    }
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button(action: { refreshEmails() }) {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh")
            }

            ToolbarItem(placement: .primaryAction) {
                Button(action: { openInFinder() }) {
                    Image(systemName: "folder")
                }
                .help("Open in Finder")
                .disabled(selectedEmail == nil)
            }
        }
    }

    @ViewBuilder
    private func emailListView(for selection: String) -> some View {
        let filteredEmails = browserService.emails.filter { email in
            searchText.isEmpty ||
            email.subject.localizedCaseInsensitiveContains(searchText) ||
            email.sender.localizedCaseInsensitiveContains(searchText)
        }

        if browserService.isLoading {
            ProgressView("Loading emails...")
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if filteredEmails.isEmpty {
            ContentUnavailableView(
                searchText.isEmpty ? "No Emails" : "No Results",
                systemImage: "envelope",
                description: Text(searchText.isEmpty ? "This folder is empty" : "No emails match your search")
            )
        } else {
            List(filteredEmails, selection: $selectedEmail) { email in
                EmailRowView(email: email)
                    .tag(email)
            }
            .listStyle(.inset)
        }
    }

    private func folderIcon(for folder: String) -> String {
        let lower = folder.lowercased()
        if lower.contains("inbox") { return "tray.fill" }
        if lower.contains("sent") { return "paperplane.fill" }
        if lower.contains("draft") { return "doc.fill" }
        if lower.contains("trash") || lower.contains("deleted") { return "trash.fill" }
        if lower.contains("spam") || lower.contains("junk") { return "xmark.shield.fill" }
        if lower.contains("archive") { return "archivebox.fill" }
        return "folder.fill"
    }

    private func refreshEmails() {
        Task {
            await browserService.loadAccounts(from: backupManager.backupLocation)
            if let selection = selectedFolder {
                let parts = selection.split(separator: "/", maxSplits: 1)
                if parts.count == 2 {
                    await browserService.loadEmails(
                        account: String(parts[0]),
                        folder: String(parts[1]),
                        from: backupManager.backupLocation
                    )
                }
            }
        }
    }

    private func openInFinder() {
        guard let email = selectedEmail else { return }
        NSWorkspace.shared.selectFile(email.filePath, inFileViewerRootedAtPath: "")
    }
}

// MARK: - Email Row View

struct EmailRowView: View {
    let email: EmailFileInfo

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(email.subject)
                    .font(.headline)
                    .lineLimit(1)
                Spacer()
                Text(email.date, style: .date)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack {
                Text(email.sender)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                Spacer()
                Text(email.formattedSize)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Email Preview View

struct EmailPreviewView: View {
    let email: EmailFileInfo
    @State private var emailContent: String = ""
    @State private var isLoading = true
    @State private var headers: EmailHeaders?

    var body: some View {
        VStack(spacing: 0) {
            // Header
            VStack(alignment: .leading, spacing: 8) {
                Text(email.subject)
                    .font(.title2)
                    .fontWeight(.semibold)

                if let headers = headers {
                    HStack {
                        Text("From:")
                            .foregroundStyle(.secondary)
                        Text(headers.from)
                    }
                    .font(.subheadline)

                    if !headers.to.isEmpty {
                        HStack {
                            Text("To:")
                                .foregroundStyle(.secondary)
                            Text(headers.to)
                                .lineLimit(2)
                        }
                        .font(.subheadline)
                    }

                    HStack {
                        Text("Date:")
                            .foregroundStyle(.secondary)
                        Text(email.date, format: .dateTime)
                    }
                    .font(.subheadline)
                }

                Divider()
            }
            .padding()
            .background(Color(nsColor: .controlBackgroundColor))

            // Content
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ScrollView {
                    Text(emailContent)
                        .font(.body)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                }
            }

            Divider()

            // Actions
            HStack {
                Button(action: openEmail) {
                    Label("Open in Mail", systemImage: "envelope")
                }

                Button(action: revealInFinder) {
                    Label("Show in Finder", systemImage: "folder")
                }

                Spacer()

                Text(email.filePath)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
            .padding()
            .background(Color(nsColor: .controlBackgroundColor))
        }
        .task {
            await loadEmailContent()
        }
    }

    private func loadEmailContent() async {
        isLoading = true

        // Read email file
        guard let data = FileManager.default.contents(atPath: email.filePath),
              let content = String(data: data, encoding: .utf8) ?? String(data: data, encoding: .ascii) else {
            emailContent = "Unable to read email content"
            isLoading = false
            return
        }

        // Parse headers
        headers = parseHeaders(from: content)

        // Extract body (simplified - just show plain text portion)
        emailContent = extractBody(from: content)
        isLoading = false
    }

    private func parseHeaders(from content: String) -> EmailHeaders {
        var from = ""
        var to = ""
        var subject = ""

        let lines = content.components(separatedBy: .newlines)
        var currentHeader = ""

        for line in lines {
            if line.isEmpty { break } // End of headers

            if line.hasPrefix(" ") || line.hasPrefix("\t") {
                // Continuation of previous header
                currentHeader += " " + line.trimmingCharacters(in: .whitespaces)
            } else if let colonIndex = line.firstIndex(of: ":") {
                let headerName = String(line[..<colonIndex]).lowercased()
                let headerValue = String(line[line.index(after: colonIndex)...]).trimmingCharacters(in: .whitespaces)

                switch headerName {
                case "from": from = headerValue
                case "to": to = headerValue
                case "subject": subject = headerValue
                default: break
                }
                currentHeader = headerValue
            }
        }

        return EmailHeaders(from: from, to: to, subject: subject)
    }

    private func extractBody(from content: String) -> String {
        // Find the blank line that separates headers from body
        if let headerEnd = content.range(of: "\r\n\r\n") ?? content.range(of: "\n\n") {
            var body = String(content[headerEnd.upperBound...])

            // If it's multipart, try to find plain text part
            if body.contains("Content-Type: text/plain") {
                if let plainStart = body.range(of: "Content-Type: text/plain"),
                   let bodyStart = body[plainStart.upperBound...].range(of: "\n\n") ?? body[plainStart.upperBound...].range(of: "\r\n\r\n") {
                    let remainingContent = body[bodyStart.upperBound...]
                    // Find next boundary or end
                    if let boundaryEnd = remainingContent.range(of: "\n--") {
                        body = String(remainingContent[..<boundaryEnd.lowerBound])
                    } else {
                        body = String(remainingContent)
                    }
                }
            }

            // Basic cleanup
            body = body.trimmingCharacters(in: .whitespacesAndNewlines)

            // Limit preview size
            if body.count > 10000 {
                body = String(body.prefix(10000)) + "\n\n[Content truncated...]"
            }

            return body
        }

        return "Unable to parse email body"
    }

    private func openEmail() {
        let url = URL(fileURLWithPath: email.filePath)
        NSWorkspace.shared.open(url)
    }

    private func revealInFinder() {
        NSWorkspace.shared.selectFile(email.filePath, inFileViewerRootedAtPath: "")
    }
}

struct EmailHeaders {
    let from: String
    let to: String
    let subject: String
}

// MARK: - Email File Info

struct EmailFileInfo: Identifiable, Hashable {
    let id: String
    let filePath: String
    let subject: String
    let sender: String
    let date: Date
    let size: Int64

    var formattedSize: String {
        ByteCountFormatter.string(fromByteCount: size, countStyle: .file)
    }
}

// MARK: - Email Browser Service

@MainActor
class EmailBrowserService: ObservableObject {
    @Published var accounts: [String] = []
    @Published var foldersByAccount: [String: [String]] = [:]
    @Published var emails: [EmailFileInfo] = []
    @Published var isLoading = false

    private let fileManager = FileManager.default

    func folders(for account: String) -> [String] {
        foldersByAccount[account] ?? []
    }

    func loadAccounts(from backupLocation: URL) async {
        let contents = (try? fileManager.contentsOfDirectory(at: backupLocation, includingPropertiesForKeys: [.isDirectoryKey])) ?? []

        var loadedAccounts: [String] = []
        var loadedFolders: [String: [String]] = [:]

        for url in contents {
            let isDir = (try? url.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) ?? false
            if isDir {
                let accountName = url.lastPathComponent
                loadedAccounts.append(accountName)
                loadedFolders[accountName] = scanFolders(at: url)
            }
        }

        accounts = loadedAccounts.sorted()
        foldersByAccount = loadedFolders
    }

    private func scanFolders(at accountURL: URL, prefix: String = "") -> [String] {
        var folders: [String] = []

        guard let contents = try? fileManager.contentsOfDirectory(at: accountURL, includingPropertiesForKeys: [.isDirectoryKey]) else {
            return folders
        }

        for url in contents {
            let isDir = (try? url.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) ?? false
            if isDir && !url.lastPathComponent.hasPrefix(".") {
                let folderName = prefix.isEmpty ? url.lastPathComponent : "\(prefix)/\(url.lastPathComponent)"

                // Check if this folder has .eml files
                let hasEmails = (try? fileManager.contentsOfDirectory(at: url, includingPropertiesForKeys: nil))?
                    .contains { $0.pathExtension == "eml" } ?? false

                if hasEmails {
                    folders.append(folderName)
                }

                // Recursively scan subfolders
                folders.append(contentsOf: scanFolders(at: url, prefix: folderName))
            }
        }

        return folders.sorted()
    }

    func loadEmails(account: String, folder: String, from backupLocation: URL) async {
        isLoading = true
        emails = []

        let folderURL = backupLocation
            .appendingPathComponent(account)
            .appendingPathComponent(folder)

        guard let contents = try? fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: [.fileSizeKey, .contentModificationDateKey]) else {
            isLoading = false
            return
        }

        var loadedEmails: [EmailFileInfo] = []

        for url in contents where url.pathExtension == "eml" {
            let attrs = try? url.resourceValues(forKeys: [.fileSizeKey, .contentModificationDateKey])
            let size = Int64(attrs?.fileSize ?? 0)
            let modDate = attrs?.contentModificationDate ?? Date()

            // Parse filename for metadata: <UID>_<timestamp>_<sender>.eml
            let filename = url.deletingPathExtension().lastPathComponent
            let parts = filename.components(separatedBy: "_")

            var subject = "(No Subject)"
            var sender = "Unknown"
            var emailDate = modDate

            if parts.count >= 3 {
                // Try to parse date from filename
                let dateStr = "\(parts[1])_\(parts[2])"
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyyMMdd_HHmmss"
                if let parsedDate = formatter.date(from: dateStr) {
                    emailDate = parsedDate
                }

                // Sender is everything after the second underscore
                sender = parts.dropFirst(3).joined(separator: "_").replacingOccurrences(of: "_", with: " ")
                if sender.isEmpty { sender = "Unknown" }
            }

            // Try to read subject from file headers (first few KB)
            if let handle = FileHandle(forReadingAtPath: url.path) {
                let headerData = handle.readData(ofLength: 4096)
                try? handle.close()

                if let headerStr = String(data: headerData, encoding: .utf8) ?? String(data: headerData, encoding: .ascii) {
                    // Extract subject
                    if let subjectRange = headerStr.range(of: "Subject: ", options: .caseInsensitive) {
                        let afterSubject = headerStr[subjectRange.upperBound...]
                        if let endOfLine = afterSubject.firstIndex(of: "\r") ?? afterSubject.firstIndex(of: "\n") {
                            subject = String(afterSubject[..<endOfLine])
                        }
                    }

                    // Extract from
                    if let fromRange = headerStr.range(of: "From: ", options: .caseInsensitive) {
                        let afterFrom = headerStr[fromRange.upperBound...]
                        if let endOfLine = afterFrom.firstIndex(of: "\r") ?? afterFrom.firstIndex(of: "\n") {
                            sender = String(afterFrom[..<endOfLine])
                        }
                    }
                }
            }

            loadedEmails.append(EmailFileInfo(
                id: url.path,
                filePath: url.path,
                subject: subject,
                sender: sender,
                date: emailDate,
                size: size
            ))
        }

        // Sort by date, newest first
        emails = loadedEmails.sorted { $0.date > $1.date }
        isLoading = false
    }
}

#Preview {
    EmailBrowserView()
        .environmentObject(BackupManager())
        .frame(width: 1000, height: 600)
}
