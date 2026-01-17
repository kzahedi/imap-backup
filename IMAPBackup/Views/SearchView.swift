import SwiftUI
import AppKit

struct SearchView: View {
    @EnvironmentObject var backupManager: BackupManager
    @State private var searchText = ""
    @State private var searchResults: [SearchResult] = []
    @State private var isSearching = false
    @State private var isIndexing = false
    @State private var indexProgress: (current: Int, total: Int) = (0, 0)
    @State private var indexStats: (emailCount: Int, attachmentCount: Int) = (0, 0)
    @State private var errorMessage: String?
    @State private var searchService: SearchService?

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header with search field
            searchHeader

            Divider()

            // Content area
            if isIndexing {
                indexingView
            } else if isSearching {
                ProgressView("Searching...")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if searchResults.isEmpty {
                emptyStateView
            } else {
                resultsList
            }

            Divider()

            // Footer with stats
            footerView
        }
        .frame(minWidth: 600, minHeight: 400)
        .task {
            await initializeSearchService()
        }
    }

    // MARK: - Search Header

    var searchHeader: some View {
        HStack(spacing: 12) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(.secondary)

            TextField("Search emails, subjects, attachments...", text: $searchText)
                .textFieldStyle(.plain)
                .font(.title3)
                .onSubmit {
                    Task { await performSearch() }
                }

            if !searchText.isEmpty {
                Button(action: { searchText = "" }) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
            }

            Button("Search") {
                Task { await performSearch() }
            }
            .buttonStyle(.borderedProminent)
            .disabled(searchText.isEmpty || isSearching)
        }
        .padding()
        .background(Color(nsColor: .controlBackgroundColor))
    }

    // MARK: - Empty State

    var emptyStateView: some View {
        VStack(spacing: 16) {
            if searchText.isEmpty {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 48))
                    .foregroundStyle(.secondary)
                Text("Search Your Emails")
                    .font(.title2)
                Text("Search by sender, subject, email content, or attachment names and content.")
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            } else {
                Image(systemName: "doc.text.magnifyingglass")
                    .font(.system(size: 48))
                    .foregroundStyle(.secondary)
                Text("No Results")
                    .font(.title2)
                Text("No emails found matching \"\(searchText)\"")
                    .foregroundStyle(.secondary)
            }

            if indexStats.emailCount == 0 {
                Button("Build Search Index") {
                    Task { await rebuildIndex() }
                }
                .buttonStyle(.bordered)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding()
    }

    // MARK: - Indexing View

    var indexingView: some View {
        VStack(spacing: 16) {
            ProgressView(value: Double(indexProgress.current), total: Double(max(1, indexProgress.total))) {
                Text("Indexing emails...")
            }
            .progressViewStyle(.linear)

            Text("\(indexProgress.current) of \(indexProgress.total) emails indexed")
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(40)
    }

    // MARK: - Results List

    var resultsList: some View {
        List {
            ForEach(searchResults) { result in
                SearchResultRow(result: result)
                    .contentShape(Rectangle())
                    .onTapGesture(count: 2) {
                        openEmail(result)
                    }
                    .contextMenu {
                        Button("Open in Finder") {
                            revealInFinder(result)
                        }
                        Button("Open Email") {
                            openEmail(result)
                        }
                        Divider()
                        Button("Copy Path") {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(result.filePath, forType: .string)
                        }
                    }
            }
        }
        .listStyle(.inset)
    }

    // MARK: - Footer

    var footerView: some View {
        HStack {
            if !searchResults.isEmpty {
                Text("\(searchResults.count) results")
                    .foregroundStyle(.secondary)
            }

            Spacer()

            Text("\(indexStats.emailCount) emails, \(indexStats.attachmentCount) attachments indexed")
                .font(.caption)
                .foregroundStyle(.secondary)

            Button(action: {
                Task { await rebuildIndex() }
            }) {
                Image(systemName: "arrow.clockwise")
            }
            .buttonStyle(.borderless)
            .disabled(isIndexing)
            .help("Rebuild search index")
        }
        .padding(.horizontal)
        .padding(.vertical, 8)
        .background(Color(nsColor: .controlBackgroundColor))
    }

    // MARK: - Actions

    private func initializeSearchService() async {
        searchService = SearchService(backupLocation: backupManager.backupLocation)
        do {
            try await searchService?.open()
            let stats = try await searchService?.getStats() ?? (0, 0)
            await MainActor.run {
                indexStats = stats
            }
        } catch {
            await MainActor.run {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func performSearch() async {
        guard !searchText.isEmpty, let service = searchService else { return }

        await MainActor.run {
            isSearching = true
            errorMessage = nil
        }

        do {
            let results = try await service.search(query: searchText)
            await MainActor.run {
                searchResults = results
                isSearching = false
            }
        } catch {
            await MainActor.run {
                errorMessage = error.localizedDescription
                isSearching = false
            }
        }
    }

    private func rebuildIndex() async {
        guard let service = searchService else { return }

        await MainActor.run {
            isIndexing = true
            indexProgress = (0, 0)
        }

        do {
            try await service.reindexAll { current, total in
                Task { @MainActor in
                    indexProgress = (current, total)
                }
            }

            let stats = try await service.getStats()
            await MainActor.run {
                indexStats = stats
                isIndexing = false
            }
        } catch {
            await MainActor.run {
                errorMessage = error.localizedDescription
                isIndexing = false
            }
        }
    }

    private func openEmail(_ result: SearchResult) {
        let url = URL(fileURLWithPath: result.filePath)
        NSWorkspace.shared.open(url)
    }

    private func revealInFinder(_ result: SearchResult) {
        let url = URL(fileURLWithPath: result.filePath)
        NSWorkspace.shared.selectFile(url.path, inFileViewerRootedAtPath: url.deletingLastPathComponent().path)
    }
}

// MARK: - Search Result Row

struct SearchResultRow: View {
    let result: SearchResult

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Subject and date
            HStack {
                Text(result.subject)
                    .font(.headline)
                    .lineLimit(1)

                Spacer()

                Text(result.date, style: .date)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            // Sender
            HStack(spacing: 4) {
                Image(systemName: "person.circle.fill")
                    .foregroundStyle(.secondary)
                Text(result.sender)
                    .font(.subheadline)
                if !result.senderEmail.isEmpty {
                    Text("<\(result.senderEmail)>")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            // Match type badge and snippet
            HStack(alignment: .top, spacing: 8) {
                Text(result.matchType.rawValue)
                    .font(.caption)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(matchTypeColor.opacity(0.2))
                    .foregroundStyle(matchTypeColor)
                    .clipShape(Capsule())

                HighlightedText(text: result.snippet)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            // Account and mailbox
            HStack(spacing: 4) {
                Image(systemName: "folder")
                    .foregroundStyle(.tertiary)
                Text("\(result.accountId) / \(result.mailbox)")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 8)
    }

    var matchTypeColor: Color {
        switch result.matchType {
        case .sender: return .blue
        case .subject: return .green
        case .body: return .orange
        case .attachment: return .purple
        case .attachmentContent: return .pink
        }
    }
}

// MARK: - Highlighted Text

struct HighlightedText: View {
    let text: String

    var body: some View {
        highlightedAttributedString
    }

    private var highlightedAttributedString: Text {
        var result = Text("")

        // Split by <mark> tags
        let parts = text.components(separatedBy: "<mark>")

        for (index, part) in parts.enumerated() {
            if index == 0 {
                // First part is never highlighted
                result = result + Text(part)
            } else {
                // Check for closing tag
                let subparts = part.components(separatedBy: "</mark>")
                if subparts.count > 1 {
                    // Highlighted part - use bold and different color since background doesn't work with Text+
                    result = result + Text(subparts[0])
                        .foregroundColor(.orange)
                        .fontWeight(.bold)
                    // Rest after closing tag
                    result = result + Text(subparts.dropFirst().joined(separator: "</mark>"))
                } else {
                    result = result + Text(part)
                }
            }
        }

        return result
    }
}

// MARK: - Search Window

struct SearchWindow: View {
    @EnvironmentObject var backupManager: BackupManager

    var body: some View {
        SearchView()
            .environmentObject(backupManager)
    }
}

#Preview {
    SearchView()
        .environmentObject(BackupManager())
        .frame(width: 700, height: 500)
}
