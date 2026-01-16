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
}
