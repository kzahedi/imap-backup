import SwiftUI

@main
struct IMAPBackupApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var backupManager = BackupManager()

    var body: some Scene {
        // Main window
        WindowGroup {
            MainWindowView()
                .environmentObject(backupManager)
        }
        .windowStyle(.hiddenTitleBar)
        .defaultSize(width: 800, height: 600)

        // Menubar
        MenuBarExtra {
            MenubarView()
                .environmentObject(backupManager)
        } label: {
            Image(systemName: backupManager.isBackingUp ? "envelope.badge.shield.half.filled" : "envelope.fill")
        }
        .menuBarExtraStyle(.window)

        // Settings
        Settings {
            SettingsView()
                .environmentObject(backupManager)
        }
    }
}
