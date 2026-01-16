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
        }
        .frame(width: 500, height: 350)
    }
}

struct GeneralSettingsView: View {
    @EnvironmentObject var backupManager: BackupManager

    var body: some View {
        Form {
            Section {
                HStack {
                    Text("Backup Location:")
                    Spacer()
                    Text(backupManager.backupLocation.path)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Button("Choose...") {
                        backupManager.selectBackupLocation()
                    }
                }

                HStack {
                    Text("Open backup folder:")
                    Spacer()
                    Button("Open in Finder") {
                        NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: backupManager.backupLocation.path)
                    }
                }
            }

            Section {
                Toggle("Start at login", isOn: .constant(false))
                Toggle("Show in Dock", isOn: .constant(true))
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct ScheduleSettingsView: View {
    @State private var scheduleEnabled = false
    @State private var frequency: ScheduleFrequency = .daily
    @State private var scheduleTime = Date()

    var body: some View {
        Form {
            Section {
                Toggle("Enable automatic backup", isOn: $scheduleEnabled)

                if scheduleEnabled {
                    Picker("Frequency", selection: $frequency) {
                        ForEach(ScheduleFrequency.allCases, id: \.self) { freq in
                            Text(freq.description).tag(freq)
                        }
                    }

                    if frequency != .hourly {
                        DatePicker("Time", selection: $scheduleTime, displayedComponents: .hourAndMinute)
                    }
                }
            }

            Section {
                Text("Next scheduled backup:")
                    .foregroundStyle(.secondary)
                Text(scheduleEnabled ? "Tomorrow at 2:00 AM" : "Not scheduled")
                    .font(.headline)
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
