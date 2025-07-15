package testserver

import (
	"fmt"
	"log"
	"time"
)

// ExampleCreateTestServer demonstrates how to create and use a test IMAP server
func ExampleCreateTestServer() {
	// Create a test server with sample data
	server := CreateTestServer("localhost:1993")
	
	// Start the server
	err := server.Start()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()
	
	// Print server statistics
	stats := server.GetServerStats()
	fmt.Printf("Server running: %v\n", stats["running"])
	fmt.Printf("User count: %v\n", stats["user_count"])
	
	// Print user information
	users := stats["users"].(map[string]interface{})
	for username, userStats := range users {
		userMap := userStats.(map[string]interface{})
		fmt.Printf("User %s has %v folders\n", username, userMap["folder_count"])
		
		folders := userMap["folders"].(map[string]interface{})
		for folderName, folderStats := range folders {
			folderMap := folderStats.(map[string]interface{})
			fmt.Printf("  - %s: %v messages\n", folderName, folderMap["message_count"])
		}
	}
	
	fmt.Println("Test IMAP server is ready for connections on localhost:1993")
	fmt.Println("Available test accounts:")
	fmt.Println("  - testuser@example.com / password123")
	fmt.Println("  - alice@company.com / secret456")
	fmt.Println("  - bob@business.com / pass789")
	
	// Output:
	// Server running: true
	// User count: 3
	// User testuser@example.com has 4 folders
	//   - INBOX: 25 messages
	//   - Sent: 5 messages
	//   - Drafts: 1 messages
	//   - Trash: 0 messages
	// User alice@company.com has 4 folders
	//   - INBOX: 25 messages
	//   - Sent: 5 messages
	//   - Drafts: 1 messages
	//   - Trash: 0 messages
	// User bob@business.com has 4 folders
	//   - INBOX: 25 messages
	//   - Sent: 5 messages
	//   - Drafts: 1 messages
	//   - Trash: 0 messages
	// Test IMAP server is ready for connections on localhost:1993
	// Available test accounts:
	//   - testuser@example.com / password123
	//   - alice@company.com / secret456
	//   - bob@business.com / pass789
}

// ExampleIMAPServer_AddMessage demonstrates how to add custom messages to the server
func ExampleIMAPServer_AddMessage() {
	// Create a simple server
	server := NewIMAPServer("localhost:1994")
	
	// Add a user
	server.AddUser("custom@example.com", "password")
	
	// Create a custom message
	message := &Message{
		Subject: "Welcome to our service!",
		From:    "admin@testserver.local",
		To:      "custom@example.com",
		Date:    time.Now(),
		Body:    "Thank you for signing up! We're excited to have you aboard.",
		Headers: map[string]string{
			"Message-ID":     "<welcome-1@testserver.local>",
			"X-Priority":     "1",
			"X-Custom-Field": "Welcome Message",
		},
	}
	
	// Add the message to the user's INBOX
	err := server.AddMessage("custom@example.com", "INBOX", message)
	if err != nil {
		log.Printf("Failed to add message: %v", err)
		return
	}
	
	// Get server stats to verify
	stats := server.GetServerStats()
	users := stats["users"].(map[string]interface{})
	userStats := users["custom@example.com"].(map[string]interface{})
	folders := userStats["folders"].(map[string]interface{})
	inboxStats := folders["INBOX"].(map[string]interface{})
	
	fmt.Printf("Messages in INBOX: %v\n", inboxStats["message_count"])
	
	// Output:
	// Messages in INBOX: 1
}

// ExampleIMAPServer_PopulateWithSampleData shows how to populate a user with sample data
func ExampleIMAPServer_PopulateWithSampleData() {
	server := NewIMAPServer("localhost:1995")
	
	// Add a user
	server.AddUser("demo@example.com", "demopass")
	
	// Populate with sample data
	err := server.PopulateWithSampleData("demo@example.com")
	if err != nil {
		log.Printf("Failed to populate sample data: %v", err)
		return
	}
	
	// Check the results
	stats := server.GetServerStats()
	users := stats["users"].(map[string]interface{})
	userStats := users["demo@example.com"].(map[string]interface{})
	folders := userStats["folders"].(map[string]interface{})
	
	for folderName, folderStats := range folders {
		folderMap := folderStats.(map[string]interface{})
		fmt.Printf("%s: %v messages\n", folderName, folderMap["message_count"])
	}
	
	// Output:
	// INBOX: 25 messages
	// Sent: 5 messages
	// Drafts: 1 messages
	// Trash: 0 messages
}

// Example shows how to use the server for testing your IMAP backup code  
func Example() {
	// This example shows how you might use the test server
	// to test your IMAP backup functionality
	
	// Create and start test server
	server := CreateTestServer("localhost:1996")
	err := server.Start()
	if err != nil {
		log.Fatalf("Failed to start test server: %v", err)
	}
	defer server.Stop()
	
	fmt.Println("Test server started for IMAP backup testing")
	fmt.Println("You can now connect your IMAP backup client to:")
	fmt.Println("  Host: localhost")
	fmt.Println("  Port: 1996")
	fmt.Println("  Username: testuser@example.com")
	fmt.Println("  Password: password123")
	fmt.Println("")
	fmt.Println("The server contains realistic sample data including:")
	fmt.Println("  - 25 messages in INBOX with various subjects and senders")
	fmt.Println("  - 5 messages in Sent folder")
	fmt.Println("  - 1 draft message")
	fmt.Println("  - Messages with different flags (read/unread, flagged)")
	fmt.Println("  - Realistic email headers and content")
	
	// Your IMAP backup test code would go here
	// For example:
	// client := imap.NewClient("localhost:1996")
	// client.Login("testuser@example.com", "password123")
	// messages := client.FetchMessages("INBOX")
	// assert.Equal(t, 25, len(messages))
	
	// Output:
	// Test server started for IMAP backup testing
	// You can now connect your IMAP backup client to:
	//   Host: localhost
	//   Port: 1996
	//   Username: testuser@example.com
	//   Password: password123
	//
	// The server contains realistic sample data including:
	//   - 25 messages in INBOX with various subjects and senders
	//   - 5 messages in Sent folder
	//   - 1 draft message
	//   - Messages with different flags (read/unread, flagged)
	//   - Realistic email headers and content
}