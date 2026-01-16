import SwiftUI

struct AccountRowView: View {
    @EnvironmentObject var backupManager: BackupManager
    let account: EmailAccount

    var progress: BackupProgress? {
        backupManager.progress[account.id]
    }

    var body: some View {
        HStack {
            // Status indicator
            Circle()
                .fill(statusColor)
                .frame(width: 8, height: 8)

            VStack(alignment: .leading) {
                Text(account.email)
                    .lineLimit(1)

                if let progress = progress, progress.status.isActive {
                    Text(progress.status.rawValue)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if let lastBackup = account.lastBackupDate {
                    Text(lastBackup, style: .relative)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            // Progress indicator
            if let progress = progress, progress.status.isActive {
                ProgressView(value: progress.progress)
                    .progressViewStyle(.circular)
                    .scaleEffect(0.6)
            }
        }
        .padding(.vertical, 4)
        .opacity(account.isEnabled ? 1.0 : 0.5)
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
    List {
        AccountRowView(account: EmailAccount.gmail(email: "test@gmail.com", appPassword: "xxxx"))
    }
    .environmentObject(BackupManager())
}
