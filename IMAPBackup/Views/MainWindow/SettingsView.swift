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
        }
        .frame(width: 500, height: 400)
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
                    .help("Automatically launch IMAP Backup when you log in")

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

    var body: some View {
        VStack {
            List {
                ForEach(backupManager.accounts) { account in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(account.email)
                            Text(account.imapServer)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        Spacer()

                        Toggle("", isOn: Binding(
                            get: { account.isEnabled },
                            set: { newValue in
                                var updated = account
                                updated.isEnabled = newValue
                                backupManager.updateAccount(updated)
                            }
                        ))
                        .labelsHidden()
                    }
                }
                .onDelete { indexSet in
                    for index in indexSet {
                        backupManager.removeAccount(backupManager.accounts[index])
                    }
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
    }
}

#Preview {
    SettingsView()
        .environmentObject(BackupManager())
}
