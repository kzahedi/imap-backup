import Foundation
import Network

/// IMAP Service for connecting to mail servers and fetching emails
actor IMAPService {
    private var connection: NWConnection?
    private var isConnected = false
    private var responseBuffer = ""
    private var tagCounter = 0

    private let account: EmailAccount

    init(account: EmailAccount) {
        self.account = account
    }

    // MARK: - Connection Management

    func connect() async throws {
        let host = NWEndpoint.Host(account.imapServer)
        let port = NWEndpoint.Port(integerLiteral: UInt16(account.port))

        let tlsOptions = NWProtocolTLS.Options()
        let tcpOptions = NWProtocolTCP.Options()
        let params = NWParameters(tls: account.useSSL ? tlsOptions : nil, tcp: tcpOptions)

        connection = NWConnection(host: host, port: port, using: params)

        // Use a class to track if continuation has been resumed (reference type for closure capture)
        class ContinuationState {
            var hasResumed = false
        }
        let state = ContinuationState()

        return try await withCheckedThrowingContinuation { continuation in
            connection?.stateUpdateHandler = { [weak self] connectionState in
                // Only resume once
                guard !state.hasResumed else { return }

                switch connectionState {
                case .ready:
                    state.hasResumed = true
                    Task {
                        await self?.setConnected(true)
                        continuation.resume()
                    }
                case .failed(let error):
                    state.hasResumed = true
                    continuation.resume(throwing: IMAPError.connectionFailed(error.localizedDescription))
                case .cancelled:
                    state.hasResumed = true
                    continuation.resume(throwing: IMAPError.connectionCancelled)
                default:
                    break
                }
            }
            connection?.start(queue: .global(qos: .userInitiated))
        }
    }

    private func setConnected(_ value: Bool) {
        isConnected = value
    }

    func disconnect() async {
        connection?.cancel()
        connection = nil
        isConnected = false
    }

    // MARK: - IMAP Commands

    func login() async throws {
        // Read greeting
        _ = try await readResponse()

        // Send login command
        let response = try await sendCommand("LOGIN \"\(account.username)\" \"\(account.password)\"")
        guard response.contains("OK") else {
            throw IMAPError.authenticationFailed
        }
    }

    func logout() async throws {
        _ = try await sendCommand("LOGOUT")
        await disconnect()
    }

    func listFolders() async throws -> [IMAPFolder] {
        let response = try await sendCommand("LIST \"\" \"*\"")
        return parseListResponse(response)
    }

    func selectFolder(_ folder: String) async throws -> FolderStatus {
        let escapedFolder = folder.replacingOccurrences(of: "\"", with: "\\\"")
        let response = try await sendCommand("SELECT \"\(escapedFolder)\"")
        return parseFolderStatus(response)
    }

    func fetchEmailHeaders(uids: ClosedRange<UInt32>) async throws -> [EmailHeader] {
        let response = try await sendCommand(
            "UID FETCH \(uids.lowerBound):\(uids.upperBound) (UID FLAGS BODY.PEEK[HEADER.FIELDS (FROM SUBJECT DATE MESSAGE-ID)] BODYSTRUCTURE)"
        )
        return parseEmailHeaders(response)
    }

    func fetchEmail(uid: UInt32) async throws -> Data {
        let response = try await sendCommand("UID FETCH \(uid) BODY.PEEK[]")
        return extractEmailData(from: response)
    }

    func searchUnseen() async throws -> [UInt32] {
        let response = try await sendCommand("UID SEARCH ALL")
        return parseSearchResponse(response)
    }

    // MARK: - Low-level Communication

    private func sendCommand(_ command: String) async throws -> String {
        guard let connection = connection else {
            throw IMAPError.notConnected
        }

        tagCounter += 1
        let tag = "A\(String(format: "%04d", tagCounter))"
        let fullCommand = "\(tag) \(command)\r\n"

        // Send command
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            connection.send(
                content: fullCommand.data(using: .utf8),
                completion: .contentProcessed { error in
                    if let error = error {
                        continuation.resume(throwing: IMAPError.sendFailed(error.localizedDescription))
                    } else {
                        continuation.resume()
                    }
                }
            )
        }

        // Read response until we get the tagged response
        var fullResponse = ""
        while true {
            let chunk = try await readResponse()
            fullResponse += chunk

            // Check if we have the complete tagged response
            if chunk.contains("\(tag) OK") || chunk.contains("\(tag) NO") || chunk.contains("\(tag) BAD") {
                break
            }
        }

        return fullResponse
    }

    private func readResponse() async throws -> String {
        guard let connection = connection else {
            throw IMAPError.notConnected
        }

        return try await withCheckedThrowingContinuation { continuation in
            connection.receive(minimumIncompleteLength: 1, maximumLength: 65536) { data, _, _, error in
                if let error = error {
                    continuation.resume(throwing: IMAPError.receiveFailed(error.localizedDescription))
                    return
                }

                if let data = data, let response = String(data: data, encoding: .utf8) {
                    continuation.resume(returning: response)
                } else {
                    continuation.resume(returning: "")
                }
            }
        }
    }

    // MARK: - Response Parsing

    private func parseListResponse(_ response: String) -> [IMAPFolder] {
        var folders: [IMAPFolder] = []
        let lines = response.components(separatedBy: "\r\n")

        for line in lines {
            // Parse lines like: * LIST (\HasNoChildren) "/" "INBOX"
            if line.hasPrefix("* LIST") || line.hasPrefix("* LSUB") {
                if let folder = parseListLine(line) {
                    folders.append(folder)
                }
            }
        }

        return folders
    }

    private func parseListLine(_ line: String) -> IMAPFolder? {
        // Match pattern: * LIST (flags) "delimiter" "name"
        let pattern = #"\* (?:LIST|LSUB) \(([^)]*)\) "(.)" "?([^"]+)"?"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: []),
              let match = regex.firstMatch(in: line, range: NSRange(line.startIndex..., in: line)) else {
            return nil
        }

        let flagsRange = Range(match.range(at: 1), in: line)!
        let delimiterRange = Range(match.range(at: 2), in: line)!
        let nameRange = Range(match.range(at: 3), in: line)!

        let flags = String(line[flagsRange])
        let delimiter = String(line[delimiterRange])
        let name = String(line[nameRange])

        return IMAPFolder(
            name: name,
            delimiter: delimiter,
            flags: flags.components(separatedBy: " "),
            path: name.replacingOccurrences(of: delimiter, with: "/")
        )
    }

    private func parseFolderStatus(_ response: String) -> FolderStatus {
        var exists = 0
        var recent = 0
        var uidNext: UInt32 = 0
        var uidValidity: UInt32 = 0

        let lines = response.components(separatedBy: "\r\n")
        for line in lines {
            if line.contains("EXISTS") {
                exists = Int(line.components(separatedBy: " ").first(where: { Int($0) != nil }) ?? "0") ?? 0
            }
            if line.contains("RECENT") {
                recent = Int(line.components(separatedBy: " ").first(where: { Int($0) != nil }) ?? "0") ?? 0
            }
            if line.contains("UIDNEXT") {
                if let match = line.range(of: #"UIDNEXT (\d+)"#, options: .regularExpression) {
                    let numStr = line[match].replacingOccurrences(of: "UIDNEXT ", with: "")
                    uidNext = UInt32(numStr) ?? 0
                }
            }
            if line.contains("UIDVALIDITY") {
                if let match = line.range(of: #"UIDVALIDITY (\d+)"#, options: .regularExpression) {
                    let numStr = line[match].replacingOccurrences(of: "UIDVALIDITY ", with: "")
                    uidValidity = UInt32(numStr) ?? 0
                }
            }
        }

        return FolderStatus(exists: exists, recent: recent, uidNext: uidNext, uidValidity: uidValidity)
    }

    private func parseEmailHeaders(_ response: String) -> [EmailHeader] {
        // Simplified parsing - in production, use a proper MIME parser
        var headers: [EmailHeader] = []
        // TODO: Implement proper FETCH response parsing
        return headers
    }

    private func parseSearchResponse(_ response: String) -> [UInt32] {
        var uids: [UInt32] = []
        let lines = response.components(separatedBy: "\r\n")

        for line in lines {
            if line.hasPrefix("* SEARCH") {
                let parts = line.replacingOccurrences(of: "* SEARCH", with: "").trimmingCharacters(in: .whitespaces)
                for part in parts.components(separatedBy: " ") {
                    if let uid = UInt32(part) {
                        uids.append(uid)
                    }
                }
            }
        }

        return uids
    }

    private func extractEmailData(from response: String) -> Data {
        // Extract the literal email data from FETCH response
        // IMAP literal format: {size}\r\n<data>

        // Find the literal marker: BODY[] {size}
        guard let literalStart = response.range(of: "BODY[]") ?? response.range(of: "BODY.PEEK[]") else {
            return Data()
        }

        // Find the opening brace with size
        guard let braceStart = response.range(of: "{", range: literalStart.upperBound..<response.endIndex),
              let braceEnd = response.range(of: "}", range: braceStart.upperBound..<response.endIndex) else {
            return Data()
        }

        // Parse the size
        let sizeString = String(response[braceStart.upperBound..<braceEnd.lowerBound])
        guard let size = Int(sizeString) else {
            return Data()
        }

        // Find the start of actual data (after }\r\n)
        let afterBrace = braceEnd.upperBound

        // The data starts after the CRLF following the }
        // We need to find \r\n and skip it
        var dataStartIndex = afterBrace
        if dataStartIndex < response.endIndex {
            let remaining = response[dataStartIndex...]
            if remaining.hasPrefix("\r\n") {
                dataStartIndex = response.index(dataStartIndex, offsetBy: 2)
            } else if remaining.hasPrefix("\n") {
                dataStartIndex = response.index(dataStartIndex, offsetBy: 1)
            }
        }

        // Extract exactly 'size' bytes
        // Convert to UTF8 data first for accurate byte counting
        let dataSection = response[dataStartIndex...]
        let utf8Data = Data(dataSection.utf8)

        if utf8Data.count >= size {
            return utf8Data.prefix(size)
        } else {
            return utf8Data
        }
    }
}

// MARK: - Supporting Types

struct IMAPFolder: Identifiable, Hashable {
    let id = UUID()
    let name: String
    let delimiter: String
    let flags: [String]
    let path: String

    var isSelectable: Bool {
        !flags.contains("\\Noselect")
    }
}

struct FolderStatus {
    let exists: Int
    let recent: Int
    let uidNext: UInt32
    let uidValidity: UInt32
}

struct EmailHeader {
    let uid: UInt32
    let messageId: String
    let from: String
    let subject: String
    let date: Date
    let hasAttachments: Bool
    let size: Int
}

// MARK: - Errors

enum IMAPError: LocalizedError {
    case notConnected
    case connectionFailed(String)
    case connectionCancelled
    case authenticationFailed
    case sendFailed(String)
    case receiveFailed(String)
    case folderNotFound(String)
    case fetchFailed(String)

    var errorDescription: String? {
        switch self {
        case .notConnected:
            return "Not connected to server"
        case .connectionFailed(let reason):
            return "Connection failed: \(reason)"
        case .connectionCancelled:
            return "Connection was cancelled"
        case .authenticationFailed:
            return "Authentication failed - check username and password"
        case .sendFailed(let reason):
            return "Failed to send command: \(reason)"
        case .receiveFailed(let reason):
            return "Failed to receive response: \(reason)"
        case .folderNotFound(let name):
            return "Folder not found: \(name)"
        case .fetchFailed(let reason):
            return "Failed to fetch email: \(reason)"
        }
    }
}
