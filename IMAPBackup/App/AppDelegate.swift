import Cocoa
import SwiftUI

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ notification: Notification) {
        // App initialization
        print("IMAP Backup started")
    }

    func applicationWillTerminate(_ notification: Notification) {
        // Cleanup
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        // Keep running in menubar when window is closed
        return false
    }

    @objc func openSearchWindow() {
        // Find and activate the search window
        for window in NSApp.windows {
            if window.title == "Search Emails" {
                window.makeKeyAndOrderFront(nil)
                NSApp.activate(ignoringOtherApps: true)
                return
            }
        }

        // If search window doesn't exist, open it via the scene
        if let url = URL(string: "imapbackup://search") {
            NSWorkspace.shared.open(url)
        }
    }
}
