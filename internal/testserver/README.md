# Test IMAP Server

A lightweight, in-memory IMAP server implementation for testing email backup functionality.

## Features

- **Full IMAP4rev1 Protocol Support**: Implements core IMAP commands (LOGIN, LIST, SELECT, FETCH, etc.)
- **Realistic Test Data**: Generates sample emails with various subjects, senders, and content
- **Multiple User Support**: Create multiple test accounts with different credentials
- **Standard Folder Structure**: INBOX, Sent, Drafts, and Trash folders
- **Message Flags**: Support for standard IMAP flags (\\Seen, \\Flagged, \\Draft, etc.)
- **RFC822 Compliance**: Generates proper RFC822 email messages
- **Thread Safe**: Concurrent connection handling with proper synchronization

## Quick Start

```go
package main

import (
    "log"
    "imap-backup/internal/testserver"
)

func main() {
    // Create a test server with sample data
    server := testserver.CreateTestServer("localhost:1993")
    
    // Start the server
    err := server.Start()
    if err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
    defer server.Stop()
    
    // Server is now ready for IMAP connections
    log.Println("Test IMAP server running on localhost:1993")
    
    // Available test accounts:
    // - testuser@example.com / password123
    // - alice@company.com / secret456  
    // - bob@business.com / pass789
    
    // Each account has 25 messages in INBOX, 5 in Sent, 1 draft
}
```

## Manual Server Setup

```go
// Create an empty server
server := testserver.NewIMAPServer("localhost:1993")

// Add users manually
server.AddUser("user@example.com", "password")

// Add custom messages
message := &testserver.Message{
    Subject: "Test Message",
    From:    "sender@example.com", 
    To:      "user@example.com",
    Date:    time.Now(),
    Body:    "Hello, this is a test message!",
}
server.AddMessage("user@example.com", "INBOX", message)

// Or populate with realistic sample data
server.PopulateWithSampleData("user@example.com")
```

## Testing Your IMAP Client

The test server is perfect for testing IMAP backup functionality:

```go
func TestIMAPBackup(t *testing.T) {
    // Start test server
    server := testserver.CreateTestServer("localhost:0") // Use port 0 for auto-assignment
    server.Start()
    defer server.Stop()
    
    // Get actual server address
    addr := server.GetListener().Addr().String()
    
    // Test your IMAP client against the server
    client := your_imap_client.New(addr)
    err := client.Login("testuser@example.com", "password123")
    assert.NoError(t, err)
    
    messages, err := client.FetchAllMessages("INBOX")
    assert.NoError(t, err)
    assert.Equal(t, 25, len(messages)) // Server has 25 sample messages
}
```

## Available Test Data

The test server generates realistic email data including:

### Sample Senders
- alice@example.com
- bob@company.com  
- newsletter@techsite.com
- support@service.com
- noreply@bank.com

### Sample Subjects
- "Important project update"
- "Weekly newsletter - Tech trends"
- "Meeting reminder for tomorrow"
- "Your account statement is ready"
- "Welcome to our service!"
- And more...

### Message Properties
- **Varied Dates**: Messages from the last 30 days
- **Realistic Content**: Professional email bodies
- **Random Flags**: 60% marked as read, 10% flagged as important
- **Proper Headers**: Message-ID, X-Mailer, Content-Type, etc.
- **RFC822 Format**: Valid raw email format

## Server Statistics

```go
stats := server.GetServerStats()
fmt.Printf("Running: %v\n", stats["running"])
fmt.Printf("Users: %v\n", stats["user_count"])

// Per-user folder and message counts
users := stats["users"].(map[string]interface{})
for username, userStats := range users {
    // Access folder message counts
}
```

## Supported IMAP Commands

- **CAPABILITY**: Lists server capabilities
- **LOGIN**: Authenticate users
- **LIST**: List available folders  
- **SELECT**: Select a folder for operations
- **FETCH**: Retrieve message data (UID, FLAGS, ENVELOPE, RFC822)
- **LOGOUT**: End session
- **NOOP**: Keep connection alive

## Use Cases

- **Unit Testing**: Test IMAP client functionality
- **Integration Testing**: Full backup workflow testing
- **Development**: Develop against consistent test data
- **CI/CD**: Automated testing without external dependencies
- **Debugging**: Inspect IMAP protocol interactions

## Implementation Notes

- **In-Memory Only**: No persistence between restarts
- **Thread Safe**: Multiple concurrent connections supported
- **Lightweight**: Minimal resource usage
- **Fast**: No network delays or external dependencies
- **Deterministic**: Consistent test data generation

The test server implements enough of the IMAP protocol to thoroughly test email backup scenarios while remaining simple and focused on testing needs.