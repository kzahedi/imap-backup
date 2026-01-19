# Contributing to IMAP Backup

Thank you for your interest in contributing to IMAP Backup! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- macOS 14.0 (Sonoma) or later
- Xcode 15.0 or later
- Swift 5.9 or later

### Setting Up Development Environment

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/imap-backup.git
   cd imap-backup
   ```
3. Open in Xcode:
   ```bash
   open IMAPBackup.xcodeproj
   ```
4. Build and run with **Cmd+R**

## Development Guidelines

### Code Style

- Follow Swift API Design Guidelines
- Use SwiftUI for all new views
- Use async/await for asynchronous operations
- Add `@MainActor` to UI-related services
- Keep functions focused and under 50 lines where possible

### Architecture

The app follows a service-oriented architecture:

- **Models**: Data structures and business logic
- **Services**: Stateful services that manage app functionality
- **Views**: SwiftUI views and components

Services use the singleton pattern for shared state:
```swift
@MainActor
final class ExampleService {
    static let shared = ExampleService()
    private init() {}
}
```

### Concurrency

- Use Swift Concurrency (async/await) for all async operations
- Use actors for thread-safe mutable state
- Mark UI services with `@MainActor`
- Avoid `DispatchQueue` unless interfacing with legacy code

### Error Handling

- Use typed errors where appropriate
- Log errors using LoggingService
- Provide user-friendly error messages in the UI
- Handle network failures gracefully with retries

## Testing

### Running Tests

```bash
# Run all tests
xcodebuild test -scheme IMAPBackup -destination 'platform=macOS'

# Run specific test suite
xcodebuild test -scheme IMAPBackup -destination 'platform=macOS' \
  -only-testing:IMAPBackupTests/ModelTests
```

### Writing Tests

- Add tests for all new services and models
- Use `XCTest` and async test methods
- Create temporary directories for file-based tests
- Clean up resources in `tearDown`

Example test structure:
```swift
final class ExampleServiceTests: XCTestCase {
    var tempDirectory: URL!

    override func setUp() async throws {
        try await super.setUp()
        tempDirectory = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: tempDirectory, withIntermediateDirectories: true)
    }

    override func tearDown() async throws {
        try? FileManager.default.removeItem(at: tempDirectory)
        try await super.tearDown()
    }

    func testExample() async throws {
        // Test implementation
    }
}
```

### Test Categories

- **Model Tests**: Test data structures and computed properties
- **Service Tests**: Test service logic with mocked dependencies
- **Integration Tests**: Test with real file system operations

## Pull Requests

### Before Submitting

1. Run all tests and ensure they pass
2. Test manually with real email accounts if touching IMAP code
3. Update documentation if adding new features
4. Add tests for new functionality

### PR Process

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Push to your fork
4. Open a pull request against `main`
5. Fill in the PR template with:
   - Description of changes
   - Testing performed
   - Screenshots (for UI changes)

### Commit Messages

Use clear, descriptive commit messages:
```
Add attachment extraction feature

- Extract attachments to separate folders
- Support base64 and quoted-printable encoding
- Handle RFC 2047 encoded filenames
```

## Reporting Issues

### Bug Reports

Include:
- macOS version
- App version (or commit hash)
- Steps to reproduce
- Expected vs actual behavior
- Log output (from `~/Library/Logs/IMAPBackup/`)

### Feature Requests

- Describe the use case
- Explain why existing features don't solve it
- Propose a solution if you have one

## Security

### Handling Credentials

- Never log passwords or tokens
- Always use Keychain for credential storage
- Don't commit test credentials

### Reporting Vulnerabilities

Please report security issues privately via email rather than public issues.

## Questions?

Feel free to open an issue for questions about contributing.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
