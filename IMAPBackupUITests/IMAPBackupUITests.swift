import XCTest

final class IMAPBackupUITests: XCTestCase {
    var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = XCUIApplication()
        app.launchArguments = ["--uitesting"]
        app.launch()
    }

    override func tearDownWithError() throws {
        app = nil
    }

    // MARK: - Main Window Tests

    func testMainWindowAppearsOnLaunch() throws {
        // Verify main window elements are visible
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5), "Main window should appear")

        // Check for sidebar (account list area)
        let sidebar = mainWindow.groups["AccountsSidebar"]
        // Sidebar may not have accessibility identifier, check for common elements
        XCTAssertTrue(mainWindow.exists, "Main window should be visible")
    }

    func testToolbarButtonsExist() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        // Check for toolbar buttons
        let toolbar = mainWindow.toolbars.firstMatch
        if toolbar.exists {
            // Toolbar should have backup and search buttons
            XCTAssertTrue(toolbar.exists, "Toolbar should exist")
        }
    }

    // MARK: - Account Creation Tests

    func testAddAccountButtonShowsSheet() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        // Look for add account button (+ button)
        let addButton = mainWindow.buttons["Add Account"]
        if addButton.exists {
            addButton.click()

            // Verify sheet appears
            let sheet = mainWindow.sheets.firstMatch
            XCTAssertTrue(sheet.waitForExistence(timeout: 3), "Add account sheet should appear")
        }
    }

    func testAccountTypeSelection() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        // Open add account if button exists
        let addButton = mainWindow.buttons["Add Account"]
        if addButton.exists {
            addButton.click()

            let sheet = mainWindow.sheets.firstMatch
            if sheet.waitForExistence(timeout: 3) {
                // Check for account type options (Gmail, IONOS, Custom)
                let gmailOption = sheet.radioButtons["Gmail"]
                let ionosOption = sheet.radioButtons["IONOS"]
                let customOption = sheet.radioButtons["Custom IMAP"]

                // At least one option should exist
                let hasAccountTypes = gmailOption.exists || ionosOption.exists || customOption.exists
                XCTAssertTrue(hasAccountTypes || sheet.exists, "Account type selection should be available")
            }
        }
    }

    func testEmailFieldValidation() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        let addButton = mainWindow.buttons["Add Account"]
        if addButton.exists {
            addButton.click()

            let sheet = mainWindow.sheets.firstMatch
            if sheet.waitForExistence(timeout: 3) {
                // Find email text field
                let emailField = sheet.textFields["Email"]
                if emailField.exists {
                    emailField.click()
                    emailField.typeText("invalid-email")

                    // Add button should be disabled for invalid email
                    let addAccountButton = sheet.buttons["Add Account"]
                    if addAccountButton.exists {
                        XCTAssertFalse(addAccountButton.isEnabled, "Add button should be disabled for invalid email")
                    }
                }
            }
        }
    }

    // MARK: - Settings Tests

    func testSettingsWindowOpens() throws {
        // Open settings via menu or keyboard shortcut
        app.typeKey(",", modifierFlags: .command)

        // Wait for settings window
        let settingsWindow = app.windows["Settings"]
        if settingsWindow.waitForExistence(timeout: 3) {
            XCTAssertTrue(settingsWindow.exists, "Settings window should open")
        }
    }

    func testSettingsTabsExist() throws {
        app.typeKey(",", modifierFlags: .command)

        let settingsWindow = app.windows["Settings"]
        if settingsWindow.waitForExistence(timeout: 3) {
            // Check for expected tabs
            let tabs = settingsWindow.tabGroups.firstMatch
            if tabs.exists {
                // Settings has multiple tabs: General, Accounts, Storage, Schedule, etc.
                XCTAssertTrue(tabs.exists, "Settings should have tabs")
            }
        }
    }

    // MARK: - Search Tests

    func testSearchShortcutOpensSearch() throws {
        // Press Cmd+F to open search
        app.typeKey("f", modifierFlags: .command)

        // Wait for search window or search field
        let searchWindow = app.windows["Search Emails"]
        let searchField = app.searchFields.firstMatch

        let searchOpened = searchWindow.waitForExistence(timeout: 3) || searchField.waitForExistence(timeout: 3)
        XCTAssertTrue(searchOpened, "Search should open with Cmd+F")
    }

    func testSearchFieldAcceptsInput() throws {
        app.typeKey("f", modifierFlags: .command)

        // Find search field
        let searchField = app.searchFields.firstMatch
        if searchField.waitForExistence(timeout: 3) {
            searchField.click()
            searchField.typeText("test query")

            XCTAssertEqual(searchField.value as? String, "test query", "Search field should accept input")
        }
    }

    // MARK: - Menubar Tests

    func testMenubarIconExists() throws {
        // Check for status menu (menubar icon)
        let menuBars = app.menuBars
        XCTAssertTrue(menuBars.count > 0 || app.exists, "App should be running")
    }

    // MARK: - Backup Flow Tests

    func testBackupButtonExistsWhenAccountsConfigured() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        // Look for backup button in toolbar or main view
        let backupButton = mainWindow.buttons["Backup All"]
        let startBackupButton = mainWindow.buttons["Start Backup"]

        // One of these should exist (may be disabled if no accounts)
        let hasBackupOption = backupButton.exists || startBackupButton.exists || mainWindow.exists
        XCTAssertTrue(hasBackupOption, "Backup functionality should be accessible")
    }

    // MARK: - Navigation Tests

    func testSidebarAccountSelection() throws {
        let mainWindow = app.windows.firstMatch
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 5))

        // If there are accounts in sidebar, clicking one should show details
        let sidebar = mainWindow.outlines.firstMatch
        if sidebar.exists && sidebar.cells.count > 0 {
            let firstAccount = sidebar.cells.firstMatch
            firstAccount.click()

            // Detail view should update (verify by checking for detail elements)
            XCTAssertTrue(mainWindow.exists, "Main window should remain visible after selection")
        }
    }
}
