package testserver

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewIMAPServer(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	if server.addr != "localhost:1993" {
		t.Errorf("Expected address localhost:1993, got %s", server.addr)
	}
	
	if len(server.capabilities) == 0 {
		t.Error("Expected capabilities to be set")
	}
	
	expectedCapabilities := []string{"IMAP4rev1", "STARTTLS", "AUTH=PLAIN", "AUTH=LOGIN", "UIDPLUS", "MOVE", "IDLE"}
	for _, cap := range expectedCapabilities {
		found := false
		for _, serverCap := range server.capabilities {
			if serverCap == cap {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability %s not found", cap)
		}
	}
}

func TestAddUser(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	
	user := server.AddUser("test@example.com", "password123")
	
	if user == nil {
		t.Fatal("Expected user to be created")
	}
	
	if user.Username != "test@example.com" {
		t.Errorf("Expected username test@example.com, got %s", user.Username)
	}
	
	if user.Password != "password123" {
		t.Errorf("Expected password password123, got %s", user.Password)
	}
	
	// Check default folders
	expectedFolders := []string{"INBOX", "Sent", "Drafts", "Trash"}
	for _, folderName := range expectedFolders {
		if _, exists := user.Folders[folderName]; !exists {
			t.Errorf("Expected folder %s to be created", folderName)
		}
	}
	
	// Verify user is added to server
	if _, exists := server.users["test@example.com"]; !exists {
		t.Error("User not added to server users map")
	}
}

func TestAddMessage(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	server.AddUser("test@example.com", "password123")
	
	msg := &Message{
		Subject: "Test Message",
		From:    "sender@example.com",
		To:      "test@example.com",
		Date:    time.Now(),
		Body:    "This is a test message",
		Headers: map[string]string{
			"Message-ID": "<test@example.com>",
		},
	}
	
	err := server.AddMessage("test@example.com", "INBOX", msg)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	
	// Check message was added
	user := server.users["test@example.com"]
	inbox := user.Folders["INBOX"]
	
	if len(inbox.Messages) != 1 {
		t.Errorf("Expected 1 message in INBOX, got %d", len(inbox.Messages))
	}
	
	addedMsg := inbox.Messages[0]
	if addedMsg.Subject != "Test Message" {
		t.Errorf("Expected subject 'Test Message', got %s", addedMsg.Subject)
	}
	
	if addedMsg.UID == 0 {
		t.Error("Expected UID to be auto-assigned")
	}
	
	if addedMsg.Raw == "" {
		t.Error("Expected raw message to be generated")
	}
	
	if addedMsg.Size == 0 {
		t.Error("Expected message size to be calculated")
	}
	
	// Test adding to non-existent user
	err = server.AddMessage("nonexistent@example.com", "INBOX", msg)
	if err == nil {
		t.Error("Expected error when adding message to non-existent user")
	}
	
	// Test adding to non-existent folder
	err = server.AddMessage("test@example.com", "NonExistent", msg)
	if err == nil {
		t.Error("Expected error when adding message to non-existent folder")
	}
}

func TestGenerateRawMessage(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	
	msg := &Message{
		Subject: "Test Subject",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Body:    "This is the message body",
		Headers: map[string]string{
			"Message-ID": "<test@example.com>",
			"X-Custom":   "Custom header",
		},
	}
	
	raw := server.generateRawMessage(msg)
	
	// Check that raw message contains expected elements
	expectedElements := []string{
		"From: sender@example.com",
		"To: recipient@example.com",
		"Subject: Test Subject",
		"Date: Mon, 15 Jan 2024 10:30:00 +0000",
		"MIME-Version: 1.0",
		"Message-ID: <test@example.com>",
		"X-Custom: Custom header",
		"Content-Type: text/plain; charset=UTF-8",
		"This is the message body",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(raw, element) {
			t.Errorf("Expected raw message to contain '%s', but it didn't. Raw: %s", element, raw)
		}
	}
}

func TestPopulateWithSampleData(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	server.AddUser("test@example.com", "password123")
	
	err := server.PopulateWithSampleData("test@example.com")
	if err != nil {
		t.Fatalf("Failed to populate sample data: %v", err)
	}
	
	user := server.users["test@example.com"]
	
	// Check INBOX has 25 messages
	if len(user.Folders["INBOX"].Messages) != 25 {
		t.Errorf("Expected 25 messages in INBOX, got %d", len(user.Folders["INBOX"].Messages))
	}
	
	// Check Sent has 5 messages
	if len(user.Folders["Sent"].Messages) != 5 {
		t.Errorf("Expected 5 messages in Sent, got %d", len(user.Folders["Sent"].Messages))
	}
	
	// Check Drafts has 1 message
	if len(user.Folders["Drafts"].Messages) != 1 {
		t.Errorf("Expected 1 message in Drafts, got %d", len(user.Folders["Drafts"].Messages))
	}
	
	// Check that messages have required fields
	inboxMsg := user.Folders["INBOX"].Messages[0]
	if inboxMsg.Subject == "" {
		t.Error("Expected sample message to have subject")
	}
	if inboxMsg.From == "" {
		t.Error("Expected sample message to have from")
	}
	if inboxMsg.Body == "" {
		t.Error("Expected sample message to have body")
	}
	if inboxMsg.Raw == "" {
		t.Error("Expected sample message to have raw content")
	}
	
	// Test with non-existent user
	err = server.PopulateWithSampleData("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error when populating data for non-existent user")
	}
}

func TestCreateTestServer(t *testing.T) {
	server := CreateTestServer("localhost:1993")
	
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	// Check that users were added
	expectedUsers := []string{"testuser@example.com", "alice@company.com", "bob@business.com"}
	for _, username := range expectedUsers {
		if _, exists := server.users[username]; !exists {
			t.Errorf("Expected user %s to be created", username)
		}
	}
	
	// Check that sample data was populated
	for _, username := range expectedUsers {
		user := server.users[username]
		if len(user.Folders["INBOX"].Messages) == 0 {
			t.Errorf("Expected user %s to have sample messages", username)
		}
	}
}

func TestGetServerStats(t *testing.T) {
	server := CreateTestServer("localhost:1993")
	
	stats := server.GetServerStats()
	
	// Check basic stats
	if stats["running"] != false {
		t.Error("Expected running to be false")
	}
	
	if stats["address"] != "localhost:1993" {
		t.Errorf("Expected address localhost:1993, got %v", stats["address"])
	}
	
	if stats["user_count"] != 3 {
		t.Errorf("Expected 3 users, got %v", stats["user_count"])
	}
	
	// Check user stats
	users, ok := stats["users"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected users to be a map")
	}
	
	if len(users) != 3 {
		t.Errorf("Expected 3 users in stats, got %d", len(users))
	}
	
	// Check that a specific user has correct folder stats
	if userStats, exists := users["testuser@example.com"]; exists {
		userMap, ok := userStats.(map[string]interface{})
		if !ok {
			t.Fatal("Expected user stats to be a map")
		}
		
		if userMap["folder_count"] != 4 {
			t.Errorf("Expected 4 folders, got %v", userMap["folder_count"])
		}
		
		folders, ok := userMap["folders"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected folders to be a map")
		}
		
		if inboxStats, exists := folders["INBOX"]; exists {
			inboxMap, ok := inboxStats.(map[string]interface{})
			if !ok {
				t.Fatal("Expected inbox stats to be a map")
			}
			
			if inboxMap["message_count"] != 25 {
				t.Errorf("Expected 25 messages in INBOX, got %v", inboxMap["message_count"])
			}
		} else {
			t.Error("Expected INBOX stats to exist")
		}
	} else {
		t.Error("Expected testuser@example.com to exist in stats")
	}
}

// Integration test - requires actual server startup
func TestServerStartStop(t *testing.T) {
	server := NewIMAPServer("localhost:0") // Use port 0 for automatic assignment
	
	// Test starting server
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	
	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Test that server is running
	if !server.running {
		t.Error("Expected server to be running")
	}
	
	// Test stopping server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
	
	// Test that server is stopped
	if server.running {
		t.Error("Expected server to be stopped")
	}
	
	// Test starting already running server
	server.Start()
	err = server.Start()
	if err == nil {
		t.Error("Expected error when starting already running server")
	}
	server.Stop()
	
	// Test stopping already stopped server
	err = server.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped server")
	}
}

// Test actual IMAP commands via network connection
func TestIMAPCommands(t *testing.T) {
	server := CreateTestServer("localhost:0")
	server.AddUser("testuser", "testpass")
	
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()
	
	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Get the actual address the server is listening on
	addr := server.listener.Addr().String()
	
	// Connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	reader := bufio.NewReader(conn)
	
	// Read greeting
	greeting, _, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read greeting: %v", err)
	}
	
	if !strings.Contains(string(greeting), "OK") {
		t.Errorf("Expected OK greeting, got: %s", string(greeting))
	}
	
	// Test CAPABILITY command
	fmt.Fprintf(conn, "A001 CAPABILITY\r\n")
	
	// Read capability response
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read capability response: %v", err)
		}
		response := string(line)
		if strings.HasPrefix(response, "A001 OK") {
			break
		}
		if strings.HasPrefix(response, "* CAPABILITY") {
			if !strings.Contains(response, "IMAP4rev1") {
				t.Errorf("Expected IMAP4rev1 capability, got: %s", response)
			}
		}
	}
	
	// Test LOGIN command
	fmt.Fprintf(conn, "A002 LOGIN \"testuser\" \"testpass\"\r\n")
	
	loginResp, _, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read login response: %v", err)
	}
	
	if !strings.Contains(string(loginResp), "A002 OK") {
		t.Errorf("Expected successful login, got: %s", string(loginResp))
	}
	
	// Test LIST command
	fmt.Fprintf(conn, "A003 LIST \"\" \"*\"\r\n")
	
	listResponses := []string{}
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read list response: %v", err)
		}
		response := string(line)
		listResponses = append(listResponses, response)
		if strings.HasPrefix(response, "A003 OK") {
			break
		}
	}
	
	// Check that we got LIST responses for folders
	foundINBOX := false
	for _, resp := range listResponses {
		if strings.Contains(resp, "* LIST") && strings.Contains(resp, "INBOX") {
			foundINBOX = true
			break
		}
	}
	if !foundINBOX {
		t.Error("Expected to find INBOX in LIST response")
	}
	
	// Test SELECT command
	fmt.Fprintf(conn, "A004 SELECT INBOX\r\n")
	
	selectResponses := []string{}
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read select response: %v", err)
		}
		response := string(line)
		selectResponses = append(selectResponses, response)
		if strings.HasPrefix(response, "A004 OK") {
			break
		}
	}
	
	// Check for EXISTS response
	foundExists := false
	for _, resp := range selectResponses {
		if strings.Contains(resp, "EXISTS") {
			foundExists = true
			break
		}
	}
	if !foundExists {
		t.Error("Expected EXISTS response in SELECT")
	}
	
	// Test LOGOUT command
	fmt.Fprintf(conn, "A005 LOGOUT\r\n")
	
	logoutResp, _, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Failed to read logout response: %v", err)
	}
	
	if !strings.Contains(string(logoutResp), "BYE") {
		t.Errorf("Expected BYE response, got: %s", string(logoutResp))
	}
}

// Benchmark tests
func BenchmarkGenerateRawMessage(b *testing.B) {
	server := NewIMAPServer("localhost:1993")
	msg := &Message{
		Subject: "Benchmark Test",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Date:    time.Now(),
		Body:    "This is a benchmark test message",
		Headers: map[string]string{
			"Message-ID": "<benchmark@example.com>",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.generateRawMessage(msg)
	}
}

func BenchmarkGenerateSampleMessage(b *testing.B) {
	server := NewIMAPServer("localhost:1993")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.generateSampleMessage(uint32(i + 1))
	}
}

func TestMessageWithAttachments(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	server.AddUser("test@example.com", "password123")
	
	// Create a message with attachments
	msg := &Message{
		Subject: "Test Message with Attachments",
		From:    "sender@example.com",
		To:      "test@example.com",
		Date:    time.Now(),
		Body:    "Please find the attached documents.",
		Attachments: []Attachment{
			{
				Filename:    "test_report.pdf",
				ContentType: "application/pdf",
				Data:        server.generateSamplePDF(),
				Encoding:    "base64",
			},
			{
				Filename:    "employee_data.csv",
				ContentType: "application/vnd.ms-excel",
				Data:        server.generateSampleExcel(),
				Encoding:    "base64",
			},
		},
		Headers: map[string]string{
			"Message-ID": "<test-attachments@example.com>",
		},
	}
	
	err := server.AddMessage("test@example.com", "INBOX", msg)
	if err != nil {
		t.Fatalf("Failed to add message with attachments: %v", err)
	}
	
	// Verify message was added
	user := server.users["test@example.com"]
	inbox := user.Folders["INBOX"]
	
	if len(inbox.Messages) != 1 {
		t.Errorf("Expected 1 message in INBOX, got %d", len(inbox.Messages))
	}
	
	addedMsg := inbox.Messages[0]
	if len(addedMsg.Attachments) != 2 {
		t.Errorf("Expected 2 attachments, got %d", len(addedMsg.Attachments))
	}
	
	// Check raw message contains multipart content
	if !strings.Contains(addedMsg.Raw, "multipart/mixed") {
		t.Error("Expected raw message to contain multipart/mixed content type")
	}
	
	if !strings.Contains(addedMsg.Raw, "test_report.pdf") {
		t.Error("Expected raw message to contain PDF attachment filename")
	}
	
	if !strings.Contains(addedMsg.Raw, "employee_data.csv") {
		t.Error("Expected raw message to contain CSV attachment filename")
	}
	
	// Check attachment data is base64 encoded
	if !strings.Contains(addedMsg.Raw, "Content-Transfer-Encoding: base64") {
		t.Error("Expected raw message to contain base64 encoding header")
	}
}

func TestGenerateAttachments(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	
	// Test PDF generation
	pdf := server.generateSamplePDF()
	if len(pdf) == 0 {
		t.Error("Expected PDF data to be generated")
	}
	if !strings.Contains(string(pdf), "%PDF") {
		t.Error("Expected PDF to contain PDF header")
	}
	
	// Test Excel generation
	excel := server.generateSampleExcel()
	if len(excel) == 0 {
		t.Error("Expected Excel data to be generated")
	}
	if !strings.Contains(string(excel), "Name,Email,Department") {
		t.Error("Expected Excel to contain CSV headers")
	}
	
	// Test Word generation
	word := server.generateSampleWord()
	if len(word) == 0 {
		t.Error("Expected Word data to be generated")
	}
	if !strings.Contains(string(word), "rtf1") {
		t.Error("Expected Word to contain RTF header")
	}
	
	// Test Image generation
	image := server.generateSampleImage()
	if len(image) == 0 {
		t.Error("Expected Image data to be generated")
	}
	// Check PNG signature
	if len(image) < 8 || image[0] != 0x89 || image[1] != 0x50 || image[2] != 0x4E || image[3] != 0x47 {
		t.Error("Expected image to have valid PNG signature")
	}
}

func TestSampleDataWithAttachments(t *testing.T) {
	server := NewIMAPServer("localhost:1993")
	server.AddUser("test@example.com", "password123")
	
	err := server.PopulateWithSampleData("test@example.com")
	if err != nil {
		t.Fatalf("Failed to populate sample data: %v", err)
	}
	
	user := server.users["test@example.com"]
	
	// Check that some messages have attachments (30% chance, so with 25 messages we should have some)
	messagesWithAttachments := 0
	totalAttachments := 0
	
	for _, msg := range user.Folders["INBOX"].Messages {
		if len(msg.Attachments) > 0 {
			messagesWithAttachments++
			totalAttachments += len(msg.Attachments)
		}
	}
	
	// We expect at least some messages to have attachments (statistically)
	if messagesWithAttachments == 0 {
		t.Log("Warning: No messages with attachments generated (this can happen due to randomization)")
	} else {
		t.Logf("Generated %d messages with attachments (%d total attachments)", messagesWithAttachments, totalAttachments)
	}
	
	// Verify that messages with attachments have proper multipart structure
	for _, msg := range user.Folders["INBOX"].Messages {
		if len(msg.Attachments) > 0 {
			if !strings.Contains(msg.Raw, "multipart/mixed") {
				t.Error("Message with attachments should have multipart/mixed content type")
			}
			if !strings.Contains(msg.Raw, "Content-Transfer-Encoding: base64") {
				t.Error("Message with attachments should have base64 encoding")
			}
		}
	}
}

func BenchmarkAddMessage(b *testing.B) {
	server := NewIMAPServer("localhost:1993")
	server.AddUser("bench@example.com", "password")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &Message{
			Subject: fmt.Sprintf("Benchmark Message %d", i),
			From:    "sender@example.com",
			To:      "bench@example.com",
			Date:    time.Now(),
			Body:    fmt.Sprintf("This is benchmark message number %d", i),
		}
		server.AddMessage("bench@example.com", "INBOX", msg)
	}
}