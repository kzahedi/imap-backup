import Foundation

struct EmailAccount: Identifiable, Codable, Hashable {
    let id: UUID
    var email: String
    var imapServer: String
    var port: Int
    var username: String
    var password: String
    var useSSL: Bool
    var isEnabled: Bool
    var lastBackupDate: Date?

    init(
        id: UUID = UUID(),
        email: String,
        imapServer: String,
        port: Int = 993,
        username: String? = nil,
        password: String,
        useSSL: Bool = true,
        isEnabled: Bool = true,
        lastBackupDate: Date? = nil
    ) {
        self.id = id
        self.email = email
        self.imapServer = imapServer
        self.port = port
        self.username = username ?? email
        self.password = password
        self.useSSL = useSSL
        self.isEnabled = isEnabled
        self.lastBackupDate = lastBackupDate
    }

    // Convenience initializer for Gmail
    static func gmail(email: String, appPassword: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.gmail.com",
            port: 993,
            password: appPassword,
            useSSL: true
        )
    }

    // Convenience initializer for IONOS
    static func ionos(email: String, password: String) -> EmailAccount {
        EmailAccount(
            email: email,
            imapServer: "imap.ionos.de",
            port: 993,
            password: password,
            useSSL: true
        )
    }
}
