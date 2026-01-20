import Foundation

/// Centralized constants for the application
enum Constants {
    // MARK: - File Size Thresholds

    /// Default threshold for streaming emails directly to disk (10 MB)
    static let defaultStreamingThresholdBytes = 10 * 1024 * 1024

    /// Maximum email size to process (50 MB)
    static let maxEmailSizeBytes = 50 * 1024 * 1024

    /// Maximum header size to read for search indexing (32 KB)
    static let maxHeaderSizeForSearch = 32 * 1024

    // MARK: - Logging

    /// Maximum log file size before rotation (10 MB)
    static let maxLogFileSizeBytes: Int64 = 10 * 1024 * 1024

    /// Maximum number of log files to keep
    static let maxLogFileCount = 5

    // MARK: - Time Intervals

    /// Nanoseconds per second for Task.sleep
    static let nanosecondsPerSecond: UInt64 = 1_000_000_000

    /// Nanoseconds per millisecond for Task.sleep
    static let nanosecondsPerMillisecond: UInt64 = 1_000_000

    // MARK: - Retry Configuration

    /// Maximum number of retry attempts for failed operations
    static let maxRetryAttempts = 3

    // MARK: - Testing

    /// Mock UID validity for tests
    static let mockUIDValidity: UInt32 = 12345
}
