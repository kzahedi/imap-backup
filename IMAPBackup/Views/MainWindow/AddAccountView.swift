import SwiftUI

struct AddAccountView: View {
    @EnvironmentObject var backupManager: BackupManager
    @Environment(\.dismiss) private var dismiss

    @State private var accountType: AccountType = .gmail
    @State private var email = ""
    @State private var password = ""
    @State private var imapServer = "imap.gmail.com"  // Default for Gmail
    @State private var port = "993"
    @State private var useSSL = true

    @State private var isTesting = false
    @State private var testResult: TestResult?

    enum AccountType: String, CaseIterable {
        case gmail = "Gmail"
        case ionos = "IONOS"
        case custom = "Custom IMAP"
    }

    enum TestResult {
        case success
        case failure(String)
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Add Email Account")
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
                // Account type picker
                Picker("Account Type", selection: $accountType) {
                    ForEach(AccountType.allCases, id: \.self) { type in
                        Text(type.rawValue).tag(type)
                    }
                }
                .onChange(of: accountType) { _, newValue in
                    switch newValue {
                    case .gmail:
                        imapServer = "imap.gmail.com"
                        port = "993"
                    case .ionos:
                        imapServer = "imap.ionos.com"
                        port = "993"
                    case .custom:
                        imapServer = ""
                        port = "993"
                    }
                }

                // Email
                TextField("Email Address", text: $email)
                    .textContentType(.emailAddress)

                // Password
                SecureField(accountType == .gmail ? "App Password" : "Password", text: $password)

                // Server settings for custom
                if accountType == .custom {
                    TextField("IMAP Server", text: $imapServer)
                    TextField("Port", text: $port)
                    Toggle("Use SSL/TLS", isOn: $useSSL)
                }

                // Help text for Gmail
                if accountType == .gmail {
                    Text("Use an App Password, not your regular password.")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    Link("How to create an App Password",
                         destination: URL(string: "https://myaccount.google.com/apppasswords")!)
                        .font(.caption)
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
                Button("Test Connection") {
                    testConnection()
                }
                .disabled(isTesting || !isFormValid)

                if isTesting {
                    ProgressView()
                        .scaleEffect(0.7)
                }

                Spacer()

                Button("Add Account") {
                    addAccount()
                }
                .buttonStyle(.borderedProminent)
                .disabled(!isFormValid)
            }
            .padding()
        }
        .frame(width: 450, height: 400)
    }

    var isFormValid: Bool {
        !email.isEmpty && !password.isEmpty && !imapServer.isEmpty && !port.isEmpty
    }

    func testConnection() {
        isTesting = true
        testResult = nil

        let account = createAccount()

        Task {
            let service = IMAPService(account: account)
            do {
                try await service.connect()
                try await service.login()
                try await service.logout()

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

    func addAccount() {
        let account = createAccount()
        backupManager.addAccount(account)
        dismiss()
    }

    func createAccount() -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: imapServer,
            port: Int(port) ?? 993,
            password: password,
            useSSL: useSSL
        )
    }
}

#Preview {
    AddAccountView()
        .environmentObject(BackupManager())
}
