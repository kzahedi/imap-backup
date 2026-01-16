import SwiftUI

struct MenubarView: View {
    @EnvironmentObject var backupManager: BackupManager
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Text("IMAP Backup")
                    .font(.headline)
                Spacer()
                if backupManager.isBackingUp {
                    ProgressView()
                        .scaleEffect(0.7)
                }
            }

            Divider()

            // Account statuses
            if backupManager.accounts.isEmpty {
                Text("No accounts configured")
                    .foregroundStyle(.secondary)
                    .font(.caption)
            } else {
                ForEach(backupManager.accounts) { account in
                    MenubarAccountRow(account: account)
                }
            }

            Divider()

            // Actions
            Button(action: {
                backupManager.startBackupAll()
            }) {
                Label("Backup Now", systemImage: "arrow.clockwise")
            }
            .disabled(backupManager.accounts.isEmpty || backupManager.isBackingUp)
            .buttonStyle(.plain)

            if backupManager.isBackingUp {
                Button(action: {
                    backupManager.cancelAllBackups()
                }) {
                    Label("Cancel Backup", systemImage: "xmark.circle")
                }
                .buttonStyle(.plain)
            }

            Divider()

            Button(action: {
                if let url = URL(string: "imapbackup://main") {
                    NSWorkspace.shared.open(url)
                }
                NSApp.activate(ignoringOtherApps: true)
                // Open main window
                for window in NSApp.windows {
                    if window.title.isEmpty || window.title == "IMAP Backup" {
                        window.makeKeyAndOrderFront(nil)
                        break
                    }
                }
            }) {
                Label("Open Main Window", systemImage: "macwindow")
            }
            .buttonStyle(.plain)

            SettingsLink {
                Label("Settings...", systemImage: "gear")
            }
            .buttonStyle(.plain)

            Divider()

            Button(action: {
                NSApplication.shared.terminate(nil)
            }) {
                Label("Quit", systemImage: "power")
            }
            .buttonStyle(.plain)
        }
        .padding()
        .frame(width: 280)
    }
}

struct MenubarAccountRow: View {
    @EnvironmentObject var backupManager: BackupManager
    let account: EmailAccount

    var progress: BackupProgress? {
        backupManager.progress[account.id]
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Circle()
                    .fill(statusColor)
                    .frame(width: 6, height: 6)

                Text(account.email)
                    .font(.caption)
                    .lineLimit(1)

                Spacer()

                if let progress = progress, progress.status.isActive {
                    Text("\(Int(progress.progress * 100))%")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            if let progress = progress, progress.status.isActive {
                ProgressView(value: progress.progress)
                    .progressViewStyle(.linear)
                    .scaleEffect(y: 0.5)

                if !progress.currentFolder.isEmpty {
                    Text(progress.currentFolder)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            } else if let lastBackup = account.lastBackupDate {
                Text("Last: \(lastBackup, style: .relative) ago")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 2)
    }

    var statusColor: Color {
        guard account.isEnabled else { return .gray }

        if let status = progress?.status {
            switch status {
            case .completed: return .green
            case .failed: return .red
            case .cancelled: return .orange
            case .idle: return .gray
            default: return .blue
            }
        }

        return .gray
    }
}

#Preview {
    MenubarView()
        .environmentObject(BackupManager())
}
