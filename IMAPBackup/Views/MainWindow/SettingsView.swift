import SwiftUI

struct SettingsView: View {
    var body: some View {
        TabView {
            GeneralSettingsView()
                .tabItem {
                    Label("General", systemImage: "gear")
                }

            ScheduleSettingsView()
                .tabItem {
                    Label("Schedule", systemImage: "calendar")
                }

            AccountsSettingsView()
                .tabItem {
                    Label("Accounts", systemImage: "person.2")
                }

            BackupHistoryView()
                .tabItem {
                    Label("History", systemImage: "clock.arrow.circlepath")
                }

            RetentionSettingsView()
                .tabItem {
                    Label("Retention", systemImage: "trash.circle")
                }

            RateLimitSettingsView()
                .tabItem {
                    Label("Rate Limit", systemImage: "speedometer")
                }

            VerificationSettingsView()
                .tabItem {
                    Label("Verify", systemImage: "checkmark.shield")
                }

            AdvancedSettingsView()
                .tabItem {
                    Label("Advanced", systemImage: "gearshape.2")
                }
        }
        .frame(width: 650, height: 550)
    }
}

struct GeneralSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager
    @StateObject private var launchService = LaunchAtLoginService.shared
    @AppStorage("hideDockIcon") private var hideDockIcon = false
    @AppStorage("LogLevel") private var logLevel = 1  // Default: info

    var body: some View {
        Form {
            Section("Storage Location") {
                // Storage type picker
                Picker("Store backups in:", selection: Binding(
                    get: { backupManager.isUsingICloud ? "icloud" : "local" },
                    set: { newValue in
                        if newValue == "icloud" {
                            backupManager.useICloudDrive()
                        } else {
                            backupManager.useLocalStorage()
                        }
                    }
                )) {
                    HStack {
                        Image(systemName: "icloud.fill")
                        Text("iCloud Drive")
                    }
                    .tag("icloud")

                    HStack {
                        Image(systemName: "internaldrive.fill")
                        Text("Local Storage")
                    }
                    .tag("local")
                }
                .pickerStyle(.radioGroup)

                // Show current path
                HStack {
                    if backupManager.isUsingICloud {
                        Image(systemName: "icloud.fill")
                            .foregroundStyle(.blue)
                        Text("Syncing to iCloud Drive")
                            .foregroundStyle(.secondary)
                    } else {
                        Image(systemName: "folder.fill")
                            .foregroundStyle(.secondary)
                    }

                    Text(backupManager.backupLocation.path)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }

                HStack {
                    Button("Choose Custom Location...") {
                        backupManager.selectBackupLocation()
                    }

                    Spacer()

                    Button("Open in Finder") {
                        NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: backupManager.backupLocation.path)
                    }
                }
            }

            Section("Startup") {
                Toggle("Start at login", isOn: $launchService.isEnabled)
                    .help("Automatically launch MailKeep when you log in")

                Toggle("Hide dock icon", isOn: $hideDockIcon)
                    .help("Run as menubar-only app (requires restart)")
                    .onChange(of: hideDockIcon) { _, newValue in
                        setDockIconVisibility(hidden: newValue)
                    }
            }

            Section("Logging") {
                Picker("Log Level", selection: $logLevel) {
                    Text("Debug").tag(0)
                    Text("Info").tag(1)
                    Text("Warning").tag(2)
                    Text("Error").tag(3)
                }
                .pickerStyle(.menu)
                .help("Set the minimum log level for file logging")

                HStack {
                    Button("Open Log File") {
                        NSWorkspace.shared.selectFile(
                            LoggingService.shared.getLogFileURL().path,
                            inFileViewerRootedAtPath: LoggingService.shared.getLogDirectoryURL().path
                        )
                    }

                    Button("Clear Logs") {
                        Task {
                            await LoggingService.shared.clearLogs()
                        }
                    }
                }
            }

            Section("Large Attachments") {
                let thresholdMB = Binding(
                    get: { backupManager.streamingThresholdBytes / (1024 * 1024) },
                    set: { backupManager.setStreamingThreshold($0 * 1024 * 1024) }
                )

                Stepper(
                    "Stream emails larger than \(thresholdMB.wrappedValue) MB",
                    value: thresholdMB,
                    in: 1...100,
                    step: 5
                )
                .help("Emails larger than this threshold are streamed directly to disk to reduce memory usage")

                Text("Large emails with attachments are streamed directly to disk instead of loading into memory. This reduces memory usage when backing up emails with large attachments.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Section("Attachment Extraction") {
                Toggle("Extract attachments to separate folders", isOn: Binding(
                    get: { AttachmentExtractionManager.shared.settings.isEnabled },
                    set: { AttachmentExtractionManager.shared.settings.isEnabled = $0 }
                ))
                .help("When enabled, attachments are extracted from emails and saved to separate folders")

                Text("When enabled, attachments (PDFs, images, documents, etc.) are extracted from .eml files and saved to a subfolder next to each email. The original .eml file is preserved with embedded attachments.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
        .onAppear {
            // Apply saved dock icon preference on app start
            setDockIconVisibility(hidden: hideDockIcon)
        }
    }

    private func setDockIconVisibility(hidden: Bool) {
        if hidden {
            NSApp.setActivationPolicy(.accessory)
        } else {
            NSApp.setActivationPolicy(.regular)
        }
    }
}

struct ScheduleSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager

    var body: some View {
        Form {
            Section("Automatic Backup") {
                Picker("Schedule", selection: Binding(
                    get: { backupManager.schedule },
                    set: { backupManager.setSchedule($0) }
                )) {
                    ForEach(BackupSchedule.allCases, id: \.self) { schedule in
                        Text(schedule.rawValue).tag(schedule)
                    }
                }
                .pickerStyle(.radioGroup)

                if backupManager.schedule.needsTimeSelection {
                    DatePicker(
                        "Backup time",
                        selection: Binding(
                            get: { backupManager.scheduledTime },
                            set: { backupManager.setScheduledTime($0) }
                        ),
                        displayedComponents: .hourAndMinute
                    )
                    .datePickerStyle(.graphical)
                }
            }

            Section("Status") {
                if backupManager.schedule != .manual {
                    HStack {
                        Image(systemName: "clock.fill")
                            .foregroundStyle(.blue)
                        VStack(alignment: .leading) {
                            Text("Next scheduled backup")
                                .foregroundStyle(.secondary)
                            if let nextBackup = backupManager.nextScheduledBackup {
                                Text(nextBackup, style: .relative)
                                    .font(.headline)
                            }
                        }
                    }
                } else {
                    HStack {
                        Image(systemName: "clock")
                            .foregroundStyle(.secondary)
                        Text("Automatic backup is disabled")
                            .foregroundStyle(.secondary)
                    }
                }

                if let lastAccount = backupManager.accounts.first(where: { $0.lastBackupDate != nil }),
                   let lastBackup = lastAccount.lastBackupDate {
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(.green)
                        VStack(alignment: .leading) {
                            Text("Last backup")
                                .foregroundStyle(.secondary)
                            Text(lastBackup, style: .relative) + Text(" ago")
                        }
                    }
                }
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct AccountsSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager
    @State private var showingAddAccount = false
    @State private var accountToEdit: EmailAccount?
    @State private var accountToDelete: EmailAccount?
    @State private var showingDeleteConfirmation = false

    var body: some View {
        VStack {
            List {
                ForEach(backupManager.accounts) { account in
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(account.email)
                                if account.authType == .oauth2 {
                                    Text("OAuth")
                                        .font(.caption2)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 2)
                                        .background(Color.blue.opacity(0.2))
                                        .foregroundStyle(.blue)
                                        .cornerRadius(4)
                                }
                            }
                            Text(account.imapServer)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        Spacer()

                        // Edit button
                        Button(action: { accountToEdit = account }) {
                            Image(systemName: "pencil")
                        }
                        .buttonStyle(.borderless)
                        .help("Edit account")

                        // Delete button
                        Button(action: {
                            accountToDelete = account
                            showingDeleteConfirmation = true
                        }) {
                            Image(systemName: "trash")
                                .foregroundStyle(.red)
                        }
                        .buttonStyle(.borderless)
                        .help("Delete account")

                        Toggle("", isOn: Binding(
                            get: { account.isEnabled },
                            set: { newValue in
                                var updated = account
                                updated.isEnabled = newValue
                                backupManager.updateAccount(updated)
                            }
                        ))
                        .labelsHidden()
                        .help("Enable/disable backup")
                    }
                    .padding(.vertical, 4)
                }
            }

            HStack {
                Button(action: { showingAddAccount = true }) {
                    Label("Add Account", systemImage: "plus")
                }

                Spacer()
            }
            .padding()
        }
        .sheet(isPresented: $showingAddAccount) {
            AddAccountView()
        }
        .sheet(item: $accountToEdit) { account in
            EditAccountView(account: account)
        }
        .alert("Delete Account?", isPresented: $showingDeleteConfirmation) {
            Button("Cancel", role: .cancel) {
                accountToDelete = nil
            }
            Button("Delete", role: .destructive) {
                if let account = accountToDelete {
                    backupManager.removeAccount(account)
                }
                accountToDelete = nil
            }
        } message: {
            if let account = accountToDelete {
                Text("Are you sure you want to delete \(account.email)? This will remove the account from the app but will not delete any backed up emails.")
            }
        }
    }
}

struct EditAccountView: View {
    @EnvironmentObject var backupManager: BackupManager
    @Environment(\.dismiss) private var dismiss

    let account: EmailAccount

    @State private var email: String
    @State private var password = ""
    @State private var imapServer: String
    @State private var port: String
    @State private var useSSL: Bool

    @State private var isTesting = false
    @State private var testResult: TestResult?

    enum TestResult {
        case success
        case failure(String)
    }

    init(account: EmailAccount) {
        self.account = account
        _email = State(initialValue: account.email)
        _imapServer = State(initialValue: account.imapServer)
        _port = State(initialValue: String(account.port))
        _useSSL = State(initialValue: account.useSSL)
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Edit Account")
                    .font(.headline)
                Spacer()
                Button("Cancel") {
                    dismiss()
                }
                .buttonStyle(.plain)
            }
            .padding()

            Divider()

            // Form
            Form {
                if account.authType == .oauth2 {
                    // OAuth account - limited editing
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(.green)
                        Text("Signed in with Google")
                            .foregroundStyle(.primary)
                    }

                    LabeledContent("Email") {
                        Text(email)
                            .foregroundStyle(.secondary)
                    }

                    LabeledContent("Server") {
                        Text(imapServer)
                            .foregroundStyle(.secondary)
                    }

                    Text("To change the Google account, delete this account and add a new one.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    // Password-based account - full editing
                    TextField("Email Address", text: $email)
                        .textContentType(.emailAddress)

                    SecureField("Password", text: $password)

                    Text("Enter password and test connection to save it. Leave blank to use saved password.")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    TextField("IMAP Server", text: $imapServer)
                    TextField("Port", text: $port)
                    Toggle("Use SSL/TLS", isOn: $useSSL)
                }
            }
            .formStyle(.grouped)

            Divider()

            // Test result
            if let result = testResult {
                HStack {
                    switch result {
                    case .success:
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(.green)
                        Text("Connection successful!")
                            .foregroundStyle(.green)
                    case .failure(let message):
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.red)
                        Text(message)
                            .foregroundStyle(.red)
                            .lineLimit(2)
                    }
                    Spacer()
                }
                .padding(.horizontal)
                .padding(.vertical, 8)
            }

            // Actions
            HStack {
                if account.authType != .oauth2 {
                    Button("Test Connection") {
                        testConnection()
                    }
                    .disabled(isTesting || !isFormValid)

                    if isTesting {
                        ProgressView()
                            .scaleEffect(0.7)
                    }
                }

                Spacer()

                Button("Save Changes") {
                    saveChanges()
                }
                .buttonStyle(.borderedProminent)
                .disabled(account.authType != .oauth2 && !isFormValid)
            }
            .padding()
        }
        .frame(width: 450, height: account.authType == .oauth2 ? 300 : 380)
    }

    var isFormValid: Bool {
        !email.isEmpty && !imapServer.isEmpty && !port.isEmpty
    }

    func testConnection() {
        isTesting = true
        testResult = nil

        Task {
            do {
                // Get password: use typed password if available, otherwise try Keychain
                let testPassword: String
                if !password.isEmpty {
                    testPassword = password
                } else if let keychainPassword = try? await KeychainService.shared.getPassword(for: account.id) {
                    testPassword = keychainPassword
                } else {
                    await MainActor.run {
                        testResult = .failure("No password provided. Please enter the password.")
                        isTesting = false
                    }
                    return
                }

                let testAccount = EmailAccount(
                    id: account.id,
                    email: email,
                    imapServer: imapServer,
                    port: Int(port) ?? 993,
                    password: testPassword,
                    useSSL: useSSL,
                    authType: .password
                )

                let service = IMAPService(account: testAccount)
                try await service.connect()
                try await service.login()
                try await service.logout()

                // Save password to Keychain on successful test
                if !password.isEmpty {
                    try? await KeychainService.shared.savePassword(password, for: account.id)
                }

                await MainActor.run {
                    testResult = .success
                    isTesting = false
                }
            } catch {
                await MainActor.run {
                    testResult = .failure(error.localizedDescription)
                    isTesting = false
                }
            }
        }
    }

    func saveChanges() {
        var updatedAccount = account
        updatedAccount.email = email
        updatedAccount.username = email  // Username should match email for IMAP login
        updatedAccount.imapServer = imapServer
        updatedAccount.port = Int(port) ?? 993
        updatedAccount.useSSL = useSSL

        // Update password only if a new one was provided
        let newPassword = password.isEmpty ? nil : password

        backupManager.updateAccount(updatedAccount, password: newPassword)
        dismiss()
    }
}

struct RetentionSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager
    @StateObject private var retentionService = RetentionService.shared
    @State private var previewResult: RetentionResult?
    @State private var isApplying = false

    var body: some View {
        Form {
            Section("Retention Policy") {
                Picker("Policy", selection: $retentionService.globalSettings.policy) {
                    ForEach(RetentionPolicy.allCases, id: \.self) { policy in
                        Text(policy.rawValue).tag(policy)
                    }
                }
                .pickerStyle(.radioGroup)

                if retentionService.globalSettings.policy == .byAge {
                    Stepper(
                        "Delete backups older than \(retentionService.globalSettings.maxAgeDays) days",
                        value: $retentionService.globalSettings.maxAgeDays,
                        in: 7...3650,
                        step: 30
                    )

                    Text("Backups older than \(retentionService.globalSettings.maxAgeDays) days will be automatically deleted after each backup run.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                if retentionService.globalSettings.policy == .byCount {
                    Stepper(
                        "Keep only \(retentionService.globalSettings.maxCount) newest backups",
                        value: $retentionService.globalSettings.maxCount,
                        in: 100...100000,
                        step: 100
                    )

                    Text("Only the \(retentionService.globalSettings.maxCount) most recent email backups will be kept. Older emails will be deleted.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            Section("Manual Actions") {
                HStack {
                    Button("Preview") {
                        previewResult = retentionService.previewRetention(at: backupManager.backupLocation)
                    }
                    .disabled(retentionService.globalSettings.policy == .keepAll)

                    Button("Apply Now") {
                        isApplying = true
                        Task {
                            _ = await retentionService.applyRetentionToAll(backupLocation: backupManager.backupLocation)
                            await MainActor.run {
                                isApplying = false
                                previewResult = nil
                            }
                        }
                    }
                    .disabled(retentionService.globalSettings.policy == .keepAll || isApplying)

                    if isApplying {
                        ProgressView()
                            .scaleEffect(0.7)
                    }
                }

                if let preview = previewResult {
                    HStack {
                        Image(systemName: "info.circle.fill")
                            .foregroundStyle(.blue)
                        if preview.filesDeleted == 0 {
                            Text("No files would be deleted with current policy.")
                        } else {
                            Text("Would delete \(preview.filesDeleted) files, freeing \(preview.bytesFreedFormatted)")
                        }
                    }
                    .font(.callout)
                }
            }

            Section {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                    Text("Retention policies permanently delete email backups. Deleted emails cannot be recovered.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct VerificationResultsListView: View {
    let results: [AccountVerificationResult]

    var body: some View {
        Group {
            ForEach(results, id: \.id) { (result: AccountVerificationResult) in
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(result.accountEmail)
                        .font(.headline)
                    Spacer()
                    if result.isFullySynced {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(.green)
                    } else {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.orange)
                    }
                }

                Text(result.summary)
                    .font(.caption)
                    .foregroundColor(result.isFullySynced ? .secondary : .orange)

                HStack {
                    Text("Server: \(result.totalServerEmails) emails")
                    Text("•")
                    Text("Local: \(result.totalLocalEmails) emails")
                }
                .font(.caption2)
                .foregroundStyle(.secondary)

                Text("Verified \(result.verifiedAt, style: .relative) ago")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            .padding(.vertical, 4)
            }
        }
    }
}

struct VerificationSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager
    @StateObject private var verificationService = VerificationService.shared

    private var verificationResults: [AccountVerificationResult] {
        verificationService.lastResults
    }

    var body: some View {
        Form {
            Section("Backup Verification") {
                HStack {
                    Image(systemName: "info.circle.fill")
                        .foregroundStyle(.blue)
                    Text("Verification compares your local backups with the email server to detect missing emails or emails that have been deleted on the server.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Button(action: {
                    Task {
                        _ = await verificationService.verifyAll(
                            accounts: backupManager.accounts,
                            backupLocation: backupManager.backupLocation
                        )
                    }
                }) {
                    HStack {
                        if verificationService.isVerifying {
                            ProgressView()
                                .scaleEffect(0.7)
                            Text("Verifying...")
                        } else {
                            Image(systemName: "checkmark.shield")
                            Text("Verify All Accounts")
                        }
                    }
                }
                .disabled(verificationService.isVerifying || backupManager.accounts.isEmpty)

                if verificationService.isVerifying {
                    if let account = verificationService.currentAccount {
                        HStack {
                            Text("Account:")
                                .foregroundStyle(.secondary)
                            Text(account)
                        }
                        .font(.caption)
                    }
                    if let folder = verificationService.currentFolder {
                        HStack {
                            Text("Folder:")
                                .foregroundStyle(.secondary)
                            Text(folder)
                        }
                        .font(.caption)
                    }
                }
            }

            if !verificationResults.isEmpty {
                Section("Last Verification Results") {
                    VerificationResultsListView(results: verificationResults)

                    Button("Clear Results") {
                        verificationService.clearResults()
                    }
                    .buttonStyle(.borderless)
                }
            }

            Section {
                HStack {
                    Image(systemName: "lightbulb.fill")
                        .foregroundStyle(.yellow)
                    Text("Run verification periodically to ensure your backups are complete. Missing emails will be downloaded on the next backup.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct RateLimitSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager
    @StateObject private var rateLimitService = RateLimitService.shared
    @State private var selectedPreset: RateLimitPreset = .balanced

    var body: some View {
        Form {
            Section("Global Rate Limiting") {
                Toggle("Enable rate limiting", isOn: $rateLimitService.globalSettings.isEnabled)
                    .help("Add delays between requests to avoid server throttling")

                if rateLimitService.globalSettings.isEnabled {
                    Picker("Preset", selection: $selectedPreset) {
                        ForEach(RateLimitPreset.allCases, id: \.self) { preset in
                            Text(preset.rawValue).tag(preset)
                        }
                    }
                    .pickerStyle(.segmented)
                    .onChange(of: selectedPreset) { _, newValue in
                        if newValue != .custom {
                            rateLimitService.globalSettings = newValue.settings
                        }
                    }

                    Text(selectedPreset.description)
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    if selectedPreset == .custom {
                        Stepper(
                            "Request delay: \(rateLimitService.globalSettings.requestDelayMs)ms",
                            value: $rateLimitService.globalSettings.requestDelayMs,
                            in: 0...5000,
                            step: 50
                        )

                        Stepper(
                            "Max throttle delay: \(rateLimitService.globalSettings.maxThrottleDelayMs / 1000)s",
                            value: Binding(
                                get: { rateLimitService.globalSettings.maxThrottleDelayMs / 1000 },
                                set: { rateLimitService.globalSettings.maxThrottleDelayMs = $0 * 1000 }
                            ),
                            in: 5...120,
                            step: 5
                        )
                    }
                }
            }

            Section("Throttle Detection") {
                HStack {
                    Image(systemName: "info.circle.fill")
                        .foregroundStyle(.blue)
                    Text("The app automatically detects when servers send throttle warnings and backs off accordingly.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                HStack {
                    Text("Backoff multiplier")
                    Spacer()
                    Text("\(rateLimitService.globalSettings.throttleBackoffMultiplier, specifier: "%.1f")x")
                        .foregroundStyle(.secondary)
                }

                Slider(
                    value: $rateLimitService.globalSettings.throttleBackoffMultiplier,
                    in: 1.5...4.0,
                    step: 0.5
                )
                .disabled(!rateLimitService.globalSettings.isEnabled)

                Button("Reset Throttle State") {
                    Task {
                        await rateLimitService.resetAllThrottles()
                    }
                }
                .help("Clear any accumulated throttle delays")
            }

            Section("Per-Account Settings") {
                if backupManager.accounts.isEmpty {
                    Text("No accounts configured")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(backupManager.accounts) { account in
                        HStack {
                            VStack(alignment: .leading) {
                                Text(account.email)
                                    .font(.body)
                                if rateLimitService.hasCustomSettings(for: account.id) {
                                    let settings = rateLimitService.getSettings(for: account.id)
                                    Text("Custom: \(settings.requestDelayMs)ms delay")
                                        .font(.caption)
                                        .foregroundStyle(.blue)
                                } else {
                                    Text("Using global settings")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }

                            Spacer()

                            if rateLimitService.hasCustomSettings(for: account.id) {
                                Button("Reset") {
                                    rateLimitService.removeSettings(for: account.id)
                                }
                                .buttonStyle(.borderless)
                            }
                        }
                    }
                }

                Text("To customize per-account settings, click an account above.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
        .onAppear {
            // Detect current preset
            detectCurrentPreset()
        }
    }

    private func detectCurrentPreset() {
        let current = rateLimitService.globalSettings

        if current.requestDelayMs == RateLimitSettings.conservative.requestDelayMs &&
           current.throttleBackoffMultiplier == RateLimitSettings.conservative.throttleBackoffMultiplier {
            selectedPreset = .conservative
        } else if current.requestDelayMs == RateLimitSettings.aggressive.requestDelayMs &&
                  current.throttleBackoffMultiplier == RateLimitSettings.aggressive.throttleBackoffMultiplier {
            selectedPreset = .aggressive
        } else if current.requestDelayMs == RateLimitSettings.default.requestDelayMs &&
                  current.throttleBackoffMultiplier == RateLimitSettings.default.throttleBackoffMultiplier {
            selectedPreset = .balanced
        } else {
            selectedPreset = .custom
        }
    }
}

struct AdvancedSettingsView: View {
    @AppStorage("googleOAuthClientId") private var clientId = ""
    @State private var showingClientIdHelp = false

    var body: some View {
        Form {
            Section("Google OAuth Configuration") {
                HStack {
                    Image(systemName: "info.circle.fill")
                        .foregroundStyle(.blue)
                    Text("To use 'Sign in with Google' for Gmail accounts, you need to configure OAuth credentials from Google Cloud Console.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("Client ID")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("Enter your Google OAuth Client ID", text: $clientId)
                        .textFieldStyle(.roundedBorder)
                        .help("Your Google Cloud OAuth 2.0 Client ID (e.g., 123456789-abc.apps.googleusercontent.com)")
                }

                if !clientId.isEmpty {
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(.green)
                        Text("OAuth configured")
                            .foregroundStyle(.green)
                    }
                    .font(.caption)
                } else {
                    HStack {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.orange)
                        Text("OAuth not configured - Gmail accounts will require App Passwords")
                            .foregroundStyle(.orange)
                    }
                    .font(.caption)
                }

                Button("Setup Instructions") {
                    showingClientIdHelp = true
                }
                .buttonStyle(.link)
            }

            Section("How to Get a Client ID") {
                VStack(alignment: .leading, spacing: 8) {
                    Text("1. Go to Google Cloud Console")
                    Text("2. Create a new project or select existing")
                    Text("3. Enable the Gmail API")
                    Text("4. Go to Credentials → Create Credentials → OAuth Client ID")
                    Text("5. Select 'macOS' as application type")
                    Text("6. Copy the Client ID and paste it above")
                }
                .font(.caption)
                .foregroundStyle(.secondary)

                Link("Open Google Cloud Console",
                     destination: URL(string: "https://console.cloud.google.com/apis/credentials")!)
                    .font(.caption)
            }

            Section {
                HStack {
                    Image(systemName: "lock.shield.fill")
                        .foregroundStyle(.green)
                    Text("Your credentials are stored locally on this device. OAuth tokens are stored securely in the macOS Keychain.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .formStyle(.grouped)
        .padding()
        .sheet(isPresented: $showingClientIdHelp) {
            OAuthSetupHelpView()
        }
    }
}

struct OAuthSetupHelpView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                Text("Setting Up Google OAuth")
                    .font(.headline)
                Spacer()
                Button("Done") { dismiss() }
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    Group {
                        Text("Step 1: Create a Google Cloud Project")
                            .font(.subheadline.bold())
                        Text("Go to console.cloud.google.com and create a new project, or select an existing one.")

                        Text("Step 2: Enable the Gmail API")
                            .font(.subheadline.bold())
                        Text("In your project, go to 'APIs & Services' → 'Library' and search for 'Gmail API'. Enable it.")

                        Text("Step 3: Configure OAuth Consent Screen")
                            .font(.subheadline.bold())
                        Text("Go to 'OAuth consent screen' and configure it as 'External'. Add the required scopes: 'email', 'profile', and 'https://mail.google.com/'.")

                        Text("Step 4: Create OAuth Credentials")
                            .font(.subheadline.bold())
                        Text("Go to 'Credentials' → 'Create Credentials' → 'OAuth client ID'. Select 'macOS' as the application type. Use 'com.kzahedi.MailKeep' as the bundle ID.")

                        Text("Step 5: Copy the Client ID")
                            .font(.subheadline.bold())
                        Text("Copy the generated Client ID (looks like: 123456789-abc.apps.googleusercontent.com) and paste it in the Advanced settings.")
                    }
                    .font(.body)
                }
            }

            Link("Open Google Cloud Console",
                 destination: URL(string: "https://console.cloud.google.com/apis/credentials")!)
        }
        .padding()
        .frame(width: 500, height: 400)
    }
}

#Preview {
    SettingsView()
        .environmentObject(BackupManager())
}
