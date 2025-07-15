package testserver

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IMAPServer is a simple test IMAP server implementation
type IMAPServer struct {
	addr         string
	tlsConfig    *tls.Config
	users        map[string]*User
	running      bool
	listener     net.Listener
	mu           sync.RWMutex
	tagSequence  int
	capabilities []string
}

// User represents a user account on the test server
type User struct {
	Username string
	Password string
	Folders  map[string]*Folder
}

// Folder represents an IMAP folder
type Folder struct {
	Name      string
	Messages  []*Message
	Delimiter string
	Flags     []string
}

// Message represents an email message
type Message struct {
	UID         uint32
	Subject     string
	From        string
	To          string
	Date        time.Time
	Flags       []string
	Body        string
	HTMLBody    string
	Headers     map[string]string
	Attachments []Attachment
	Raw         string
	Size        int
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
	Encoding    string
}

// Connection represents a client connection
type Connection struct {
	conn        net.Conn
	reader      *bufio.Reader
	writer      *bufio.Writer
	user        *User
	selected    *Folder
	server      *IMAPServer
	id          string
	authenticated bool
}

// NewIMAPServer creates a new test IMAP server
func NewIMAPServer(addr string) *IMAPServer {
	return &IMAPServer{
		addr:  addr,
		users: make(map[string]*User),
		capabilities: []string{
			"IMAP4rev1",
			"STARTTLS",
			"AUTH=PLAIN",
			"AUTH=LOGIN",
			"UIDPLUS",
			"MOVE",
			"IDLE",
		},
	}
}

// SetTLSConfig sets the TLS configuration for the server
func (s *IMAPServer) SetTLSConfig(config *tls.Config) {
	s.tlsConfig = config
}

// AddUser adds a user to the server
func (s *IMAPServer) AddUser(username, password string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := &User{
		Username: username,
		Password: password,
		Folders:  make(map[string]*Folder),
	}

	// Create default folders
	user.Folders["INBOX"] = &Folder{
		Name:      "INBOX",
		Messages:  []*Message{},
		Delimiter: "/",
		Flags:     []string{"\\HasNoChildren"},
	}

	user.Folders["Sent"] = &Folder{
		Name:      "Sent",
		Messages:  []*Message{},
		Delimiter: "/",
		Flags:     []string{"\\HasNoChildren", "\\Sent"},
	}

	user.Folders["Drafts"] = &Folder{
		Name:      "Drafts",
		Messages:  []*Message{},
		Delimiter: "/",
		Flags:     []string{"\\HasNoChildren", "\\Drafts"},
	}

	user.Folders["Trash"] = &Folder{
		Name:      "Trash",
		Messages:  []*Message{},
		Delimiter: "/",
		Flags:     []string{"\\HasNoChildren", "\\Trash"},
	}

	s.users[username] = user
	return user
}

// AddMessage adds a message to a user's folder
func (s *IMAPServer) AddMessage(username, folderName string, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	folder, exists := user.Folders[folderName]
	if !exists {
		return fmt.Errorf("folder %s not found", folderName)
	}

	// Set UID if not set
	if msg.UID == 0 {
		msg.UID = uint32(len(folder.Messages) + 1)
	}

	// Generate raw message if not provided
	if msg.Raw == "" {
		msg.Raw = s.generateRawMessage(msg)
	}

	// Set size
	msg.Size = len(msg.Raw)

	folder.Messages = append(folder.Messages, msg)
	return nil
}

// generateRawMessage generates a raw RFC822 message
func (s *IMAPServer) generateRawMessage(msg *Message) string {
	var raw strings.Builder

	// Headers
	raw.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	raw.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))
	raw.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	raw.WriteString(fmt.Sprintf("Date: %s\r\n", msg.Date.Format(time.RFC1123Z)))
	raw.WriteString("MIME-Version: 1.0\r\n")

	// Additional headers
	for key, value := range msg.Headers {
		raw.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Determine content structure based on attachments
	if len(msg.Attachments) > 0 {
		// Mixed content with attachments
		boundary := "boundary_main_" + fmt.Sprintf("%d", time.Now().UnixNano())
		raw.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		raw.WriteString("\r\n")
		
		// Text content part
		raw.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if msg.HTMLBody != "" {
			// Alternative text/html within mixed
			altBoundary := "boundary_alt_" + fmt.Sprintf("%d", time.Now().UnixNano())
			raw.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
			raw.WriteString("\r\n")
			
			raw.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
			raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			raw.WriteString("\r\n")
			raw.WriteString(msg.Body)
			raw.WriteString("\r\n")
			
			raw.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
			raw.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			raw.WriteString("\r\n")
			raw.WriteString(msg.HTMLBody)
			raw.WriteString("\r\n")
			
			raw.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
		} else {
			raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			raw.WriteString("\r\n")
			raw.WriteString(msg.Body)
			raw.WriteString("\r\n")
		}
		
		// Attachment parts
		for _, attachment := range msg.Attachments {
			raw.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			raw.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
			raw.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
			raw.WriteString(fmt.Sprintf("Content-Transfer-Encoding: %s\r\n", attachment.Encoding))
			raw.WriteString("\r\n")
			
			if attachment.Encoding == "base64" {
				encoded := base64.StdEncoding.EncodeToString(attachment.Data)
				// Split into 76-character lines as per RFC
				for i := 0; i < len(encoded); i += 76 {
					end := i + 76
					if end > len(encoded) {
						end = len(encoded)
					}
					raw.WriteString(encoded[i:end] + "\r\n")
				}
			} else {
				raw.WriteString(string(attachment.Data))
				raw.WriteString("\r\n")
			}
		}
		
		raw.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.HTMLBody != "" {
		// Alternative text/html without attachments
		raw.WriteString("Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n")
		raw.WriteString("\r\n")
		raw.WriteString("--boundary123\r\n")
		raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		raw.WriteString("\r\n")
		raw.WriteString(msg.Body)
		raw.WriteString("\r\n--boundary123\r\n")
		raw.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		raw.WriteString("\r\n")
		raw.WriteString(msg.HTMLBody)
		raw.WriteString("\r\n--boundary123--\r\n")
	} else {
		// Simple text message
		raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		raw.WriteString("\r\n")
		raw.WriteString(msg.Body)
		raw.WriteString("\r\n")
	}

	return raw.String()
}

// Start starts the IMAP server
func (s *IMAPServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	log.Printf("Test IMAP server started on %s", s.addr)

	go s.acceptConnections()
	return nil
}

// Stop stops the IMAP server
func (s *IMAPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	log.Printf("Test IMAP server stopped")
	return nil
}

// acceptConnections accepts incoming connections
func (s *IMAPServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				log.Printf("Error accepting connection: %v", err)
			}
			return
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a client connection
func (s *IMAPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	connection := &Connection{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		server: s,
		id:     fmt.Sprintf("conn-%d", time.Now().UnixNano()),
	}

	// Send greeting
	connection.writeResponse("* OK [CAPABILITY " + strings.Join(s.capabilities, " ") + "] Test IMAP server ready")

	// Handle commands
	for {
		line, _, err := connection.reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from connection: %v", err)
			}
			return
		}

		command := strings.TrimSpace(string(line))
		if command == "" {
			continue
		}

		if err := connection.handleCommand(command); err != nil {
			log.Printf("Error handling command: %v", err)
			return
		}
	}
}

// handleCommand processes IMAP commands
func (c *Connection) handleCommand(command string) error {
	parts := strings.SplitN(command, " ", 3)
	if len(parts) < 2 {
		c.writeResponse("* BAD Invalid command format")
		return nil
	}

	tag := parts[0]
	cmd := strings.ToUpper(parts[1])
	args := ""
	if len(parts) > 2 {
		args = parts[2]
	}

	switch cmd {
	case "CAPABILITY":
		c.writeResponse("* CAPABILITY " + strings.Join(c.server.capabilities, " "))
		c.writeResponse(tag + " OK CAPABILITY completed")

	case "LOGIN":
		return c.handleLogin(tag, args)

	case "LIST":
		return c.handleList(tag, args)

	case "SELECT":
		return c.handleSelect(tag, args)

	case "EXAMINE":
		return c.handleExamine(tag, args)

	case "FETCH":
		return c.handleFetch(tag, args)

	case "LOGOUT":
		c.writeResponse("* BYE Logging out")
		c.writeResponse(tag + " OK LOGOUT completed")
		return fmt.Errorf("logout")

	case "NOOP":
		c.writeResponse(tag + " OK NOOP completed")

	default:
		c.writeResponse(tag + " BAD Unknown command")
	}

	return nil
}

// handleLogin processes LOGIN command
func (c *Connection) handleLogin(tag, args string) error {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) != 2 {
		c.writeResponse(tag + " BAD LOGIN expects username and password")
		return nil
	}

	username := strings.Trim(parts[0], "\"")
	password := strings.Trim(parts[1], "\"")

	c.server.mu.RLock()
	user, exists := c.server.users[username]
	c.server.mu.RUnlock()

	if !exists || user.Password != password {
		c.writeResponse(tag + " NO LOGIN failed")
		return nil
	}

	c.user = user
	c.authenticated = true
	c.writeResponse(tag + " OK LOGIN completed")
	return nil
}

// handleList processes LIST command
func (c *Connection) handleList(tag, args string) error {
	if !c.authenticated {
		c.writeResponse(tag + " NO Not authenticated")
		return nil
	}

	// Parse LIST arguments (simplified)
	// Format: LIST reference mailbox
	parts := strings.SplitN(args, " ", 2)
	if len(parts) != 2 {
		c.writeResponse(tag + " BAD LIST expects reference and mailbox")
		return nil
	}

	_ = strings.Trim(parts[0], "\"")
	mailbox := strings.Trim(parts[1], "\"")

	// For simplicity, list all folders if mailbox is "*"
	if mailbox == "*" {
		for _, folder := range c.user.Folders {
			flags := "(" + strings.Join(folder.Flags, " ") + ")"
			c.writeResponse(fmt.Sprintf("* LIST %s \"%s\" %s", flags, folder.Delimiter, folder.Name))
		}
	}

	c.writeResponse(tag + " OK LIST completed")
	return nil
}

// handleSelect processes SELECT command
func (c *Connection) handleSelect(tag, args string) error {
	if !c.authenticated {
		c.writeResponse(tag + " NO Not authenticated")
		return nil
	}

	folderName := strings.Trim(args, "\"")
	folder, exists := c.user.Folders[folderName]
	if !exists {
		c.writeResponse(tag + " NO Folder not found")
		return nil
	}

	c.selected = folder
	
	// Send folder status
	c.writeResponse(fmt.Sprintf("* %d EXISTS", len(folder.Messages)))
	c.writeResponse("* 0 RECENT")
	c.writeResponse("* OK [UIDVALIDITY 1] UID validity")
	c.writeResponse(fmt.Sprintf("* OK [UIDNEXT %d] Predicted next UID", len(folder.Messages)+1))
	c.writeResponse("* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)")
	c.writeResponse("* OK [PERMANENTFLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft \\*)] Permanent flags")
	
	c.writeResponse(tag + " OK [READ-WRITE] SELECT completed")
	return nil
}

// handleExamine processes EXAMINE command (read-only SELECT)
func (c *Connection) handleExamine(tag, args string) error {
	if !c.authenticated {
		c.writeResponse(tag + " NO Not authenticated")
		return nil
	}

	folderName := strings.Trim(args, "\"")
	folder, exists := c.user.Folders[folderName]
	if !exists {
		c.writeResponse(tag + " NO Folder not found")
		return nil
	}

	c.selected = folder
	
	// Send folder status (same as SELECT but read-only)
	c.writeResponse(fmt.Sprintf("* %d EXISTS", len(folder.Messages)))
	c.writeResponse("* 0 RECENT")
	c.writeResponse("* OK [UIDVALIDITY 1] UID validity")
	c.writeResponse(fmt.Sprintf("* OK [UIDNEXT %d] Predicted next UID", len(folder.Messages)+1))
	c.writeResponse("* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)")
	c.writeResponse("* OK [PERMANENTFLAGS ()] Permanent flags")
	
	c.writeResponse(tag + " OK [READ-ONLY] EXAMINE completed")
	return nil
}

// handleFetch processes FETCH command
func (c *Connection) handleFetch(tag, args string) error {
	if !c.authenticated {
		c.writeResponse(tag + " NO Not authenticated")
		return nil
	}

	if c.selected == nil {
		c.writeResponse(tag + " NO No folder selected")
		return nil
	}

	// Parse FETCH arguments (simplified)
	// Format: FETCH sequence items
	parts := strings.SplitN(args, " ", 2)
	if len(parts) != 2 {
		c.writeResponse(tag + " BAD FETCH expects sequence and items")
		return nil
	}

	sequence := parts[0]
	items := strings.ToUpper(parts[1])

	// Handle sequence (simplified - assume 1:* for all messages)
	var messages []*Message
	if sequence == "1:*" || sequence == "*" {
		messages = c.selected.Messages
	} else {
		// Parse specific sequence numbers
		if seqNum, err := strconv.Atoi(sequence); err == nil && seqNum > 0 && seqNum <= len(c.selected.Messages) {
			messages = []*Message{c.selected.Messages[seqNum-1]}
		}
	}

	// Process each message
	for i, msg := range messages {
		seqNum := i + 1
		response := fmt.Sprintf("* %d FETCH (", seqNum)
		
		var fetchItems []string
		
		if strings.Contains(items, "UID") {
			fetchItems = append(fetchItems, fmt.Sprintf("UID %d", msg.UID))
		}
		
		if strings.Contains(items, "FLAGS") {
			flags := "(" + strings.Join(msg.Flags, " ") + ")"
			fetchItems = append(fetchItems, fmt.Sprintf("FLAGS %s", flags))
		}
		
		if strings.Contains(items, "ENVELOPE") {
			envelope := fmt.Sprintf("ENVELOPE (\"%s\" \"%s\" ((\"%s\" NIL \"%s\" NIL)) NIL NIL ((\"%s\" NIL \"%s\" NIL)) NIL NIL NIL NIL)",
				msg.Date.Format("02-Jan-2006 15:04:05 -0700"),
				msg.Subject,
				msg.From, msg.From,
				msg.To, msg.To)
			fetchItems = append(fetchItems, envelope)
		}
		
		if strings.Contains(items, "RFC822") {
			fetchItems = append(fetchItems, fmt.Sprintf("RFC822 {%d}\r\n%s", len(msg.Raw), msg.Raw))
		}
		
		response += strings.Join(fetchItems, " ") + ")"
		c.writeResponse(response)
	}

	c.writeResponse(tag + " OK FETCH completed")
	return nil
}

// writeResponse writes a response to the client
func (c *Connection) writeResponse(response string) {
	c.writer.WriteString(response + "\r\n")
	c.writer.Flush()
	log.Printf("-> %s", response)
}

// generateSamplePDF creates a simple PDF file content
func (s *IMAPServer) generateSamplePDF() []byte {
	// Simple PDF header and content (minimal valid PDF)
	pdf := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 4 0 R
/Resources <<
/Font <<
/F1 5 0 R
>>
>>
>>
endobj

4 0 obj
<<
/Length 85
>>
stream
BT
/F1 12 Tf
100 700 Td
(This is a sample PDF document generated by the test IMAP server.) Tj
ET
endstream
endobj

5 0 obj
<<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
endobj

xref
0 6
0000000000 65535 f 
0000000010 00000 n 
0000000053 00000 n 
0000000100 00000 n 
0000000244 00000 n 
0000000381 00000 n 
trailer
<<
/Size 6
/Root 1 0 R
>>
startxref
458
%%EOF`
	return []byte(pdf)
}

// generateSampleExcel creates a simple Excel file content (CSV format for simplicity)
func (s *IMAPServer) generateSampleExcel() []byte {
	// Simple Excel-like CSV content that can be opened by Excel
	excel := `Name,Email,Department,Salary,Start Date
John Doe,john.doe@company.com,Engineering,75000,2023-01-15
Jane Smith,jane.smith@company.com,Marketing,65000,2023-02-01
Mike Johnson,mike.johnson@company.com,Sales,58000,2023-01-20
Sarah Wilson,sarah.wilson@company.com,HR,62000,2023-03-10
David Brown,david.brown@company.com,Engineering,78000,2022-11-15
Lisa Garcia,lisa.garcia@company.com,Finance,70000,2023-01-05
Tom Anderson,tom.anderson@company.com,Operations,55000,2023-02-15
Emily Davis,emily.davis@company.com,Marketing,63000,2023-01-30
`
	return []byte(excel)
}

// generateSampleWord creates a simple Word document content (RTF format for compatibility)
func (s *IMAPServer) generateSampleWord() []byte {
	// RTF format that can be opened by Word
	word := `{\rtf1\ansi\deff0 {\fonttbl {\f0 Times New Roman;}}
\f0\fs24 \b Project Proposal: Email Backup System\b0\par
\par
\b Executive Summary\b0\par
This document outlines the requirements and implementation plan for the new email backup system.\par
\par
\b Objectives\b0\par
\pard{\pntext\f0 1.\tab}{\*\pn\pnlvlblt\pnf0\pnindent0{\pntxtb 1.}}\fi-360\li720 Implement automated email backup functionality\par
{\pntext\f0 2.\tab}Ensure data integrity and security\par
{\pntext\f0 3.\tab}Provide easy restore capabilities\par
{\pntext\f0 4.\tab}Support multiple email providers\par
\pard\par
\b Technical Requirements\b0\par
- IMAP protocol support\par
- OAuth2 authentication\par
- Incremental backup strategy\par
- Attachment handling\par
- Cross-platform compatibility\par
\par
\b Timeline\b0\par
Phase 1: Core IMAP implementation (4 weeks)\par
Phase 2: Authentication systems (2 weeks)\par
Phase 3: User interface development (3 weeks)\par
Phase 4: Testing and deployment (2 weeks)\par
\par
\b Conclusion\b0\par
The proposed email backup system will provide reliable, secure, and user-friendly email archiving capabilities.\par
}`
	return []byte(word)
}

// generateSampleImage creates a simple image file (PNG header for realism)
func (s *IMAPServer) generateSampleImage() []byte {
	// Simple PNG file signature and minimal content
	// This creates a 1x1 transparent PNG
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 image
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, // RGBA, CRC
		0x89, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x62, 0x00, 0x02, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, // Image data
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}
	return png
}

// generateAttachmentForMessage creates appropriate attachments for a message
func (s *IMAPServer) generateAttachmentForMessage(uid uint32, msgType string) []Attachment {
	rand.Seed(time.Now().UnixNano() + int64(uid))
	
	var attachments []Attachment
	
	// Determine if this message should have attachments (30% chance)
	if rand.Float32() > 0.7 {
		return attachments // No attachments
	}
	
	// Choose attachment type based on message content and randomization
	attachmentType := rand.Intn(6) // 0=PDF, 1=Excel, 2=Word, 3=Image, 4=Multiple, 5=Mixed
	
	switch attachmentType {
	case 0: // PDF attachment
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("report_%d.pdf", uid),
			ContentType: "application/pdf",
			Data:        s.generateSamplePDF(),
			Encoding:    "base64",
		})
	case 1: // Excel attachment
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("employee_data_%d.csv", uid),
			ContentType: "application/vnd.ms-excel",
			Data:        s.generateSampleExcel(),
			Encoding:    "base64",
		})
	case 2: // Word attachment
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("proposal_%d.rtf", uid),
			ContentType: "application/msword",
			Data:        s.generateSampleWord(),
			Encoding:    "base64",
		})
	case 3: // Image attachment
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("chart_%d.png", uid),
			ContentType: "image/png",
			Data:        s.generateSampleImage(),
			Encoding:    "base64",
		})
	case 4: // Multiple similar attachments
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("quarterly_report_%d.pdf", uid),
			ContentType: "application/pdf",
			Data:        s.generateSamplePDF(),
			Encoding:    "base64",
		})
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("budget_%d.csv", uid),
			ContentType: "application/vnd.ms-excel",
			Data:        s.generateSampleExcel(),
			Encoding:    "base64",
		})
	case 5: // Mixed attachment types
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("presentation_%d.rtf", uid),
			ContentType: "application/msword",
			Data:        s.generateSampleWord(),
			Encoding:    "base64",
		})
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("data_analysis_%d.csv", uid),
			ContentType: "application/vnd.ms-excel",
			Data:        s.generateSampleExcel(),
			Encoding:    "base64",
		})
		attachments = append(attachments, Attachment{
			Filename:    fmt.Sprintf("screenshot_%d.png", uid),
			ContentType: "image/png",
			Data:        s.generateSampleImage(),
			Encoding:    "base64",
		})
	}
	
	return attachments
}

// PopulateWithSampleData adds sample messages to a user's folders
func (s *IMAPServer) PopulateWithSampleData(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	// Generate sample messages for INBOX
	for i := 1; i <= 25; i++ {
		msg := s.generateSampleMessage(uint32(i))
		user.Folders["INBOX"].Messages = append(user.Folders["INBOX"].Messages, msg)
	}

	// Generate a few messages for Sent folder
	for i := 1; i <= 5; i++ {
		msg := s.generateSentMessage(uint32(i))
		user.Folders["Sent"].Messages = append(user.Folders["Sent"].Messages, msg)
	}

	// Generate a draft message
	draft := s.generateDraftMessage(1)
	user.Folders["Drafts"].Messages = append(user.Folders["Drafts"].Messages, draft)

	return nil
}

// generateSampleMessage creates a realistic sample message
func (s *IMAPServer) generateSampleMessage(uid uint32) *Message {
	senders := []string{
		"alice@example.com",
		"bob@company.com",
		"newsletter@techsite.com",
		"support@service.com",
		"noreply@bank.com",
	}

	subjects := []string{
		"Important project update",
		"Weekly newsletter - Tech trends",
		"Meeting reminder for tomorrow",
		"Your account statement is ready",
		"Welcome to our service!",
		"Action required: Verify your email",
		"Invoice #12345 from Acme Corp",
		"Team lunch this Friday?",
		"Security alert: New device login",
		"Monthly report - January 2024",
		"Quarterly financial report (attached)",
		"Employee roster update - please review",
		"Project proposal for Q2 initiatives",
		"Budget spreadsheet for approval",
		"Contract documents for review",
		"Performance metrics analysis",
		"Training materials for new hires",
		"Invoice and receipt attached",
	}

	bodies := []string{
		"Hi there,\\n\\nJust wanted to give you a quick update on the project. Everything is progressing well and we should be able to meet the deadline.\\n\\nBest regards,\\nAlice",
		"Dear Subscriber,\\n\\nHere are this week's top tech trends that you shouldn't miss:\\n\\n1. AI developments\\n2. Cloud computing updates\\n3. Cybersecurity news\\n\\nStay informed!\\n\\nTech Newsletter Team",
		"Hello,\\n\\nThis is a reminder about our meeting scheduled for tomorrow at 2 PM in the conference room.\\n\\nPlease come prepared with your reports.\\n\\nThanks,\\nBob",
		"Dear Valued Customer,\\n\\nYour monthly account statement is now available for download in your online banking portal.\\n\\nRegards,\\nCustomer Service",
		"Welcome!\\n\\nThank you for signing up for our service. We're excited to have you on board.\\n\\nTo get started, please verify your email address by clicking the link below.\\n\\nWelcome Team",
		"Hi team,\\n\\nPlease find the quarterly report attached for your review. Let me know if you have any questions.\\n\\nRegards,\\nFinance Team",
		"Hello,\\n\\nI've attached the updated employee roster and budget spreadsheet. Please review and provide feedback by Friday.\\n\\nThanks,\\nHR Department",
		"Dear colleague,\\n\\nAttached you'll find the project proposal we discussed. Please review the documentation and let me know your thoughts.\\n\\nBest,\\nProject Manager",
		"Hi,\\n\\nThe contract documents are attached for your signature. Please review the terms and return the signed copy.\\n\\nRegards,\\nLegal Team",
		"Team,\\n\\nI've prepared the performance analysis report. The data and charts are in the attached files.\\n\\nBest regards,\\nAnalytics Team",
	}

	rand.Seed(time.Now().UnixNano() + int64(uid))
	sender := senders[rand.Intn(len(senders))]
	subject := subjects[rand.Intn(len(subjects))]
	body := bodies[rand.Intn(len(bodies))]

	// Create message with some random attributes
	msg := &Message{
		UID:         uid,
		Subject:     subject,
		From:        sender,
		To:          "user@example.com",
		Date:        time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
		Flags:       []string{},
		Body:        body,
		Attachments: s.generateAttachmentForMessage(uid, "inbox"),
		Headers: map[string]string{
			"Message-ID":   fmt.Sprintf("<%d@testserver.local>", uid),
			"X-Mailer":     "Test IMAP Server",
		},
	}

	// Randomly mark some messages as read (30% will be unread for backup testing)
	if rand.Float32() < 0.3 {
		msg.Flags = append(msg.Flags, "\\Seen")
	}

	// Randomly mark some messages as important
	if rand.Float32() < 0.1 {
		msg.Flags = append(msg.Flags, "\\Flagged")
	}

	// Generate raw message
	msg.Raw = s.generateRawMessage(msg)
	msg.Size = len(msg.Raw)

	return msg
}

// generateSentMessage creates a sample sent message
func (s *IMAPServer) generateSentMessage(uid uint32) *Message {
	subjects := []string{
		"Re: Project proposal",
		"Meeting notes from today",
		"Thank you for the presentation",
		"Follow up on our discussion",
		"Quarterly results summary",
	}

	recipients := []string{
		"client@company.com",
		"team@mycompany.com",
		"manager@office.com",
		"partner@business.com",
	}

	rand.Seed(time.Now().UnixNano() + int64(uid) + 1000)
	subject := subjects[rand.Intn(len(subjects))]
	recipient := recipients[rand.Intn(len(recipients))]

	msg := &Message{
		UID:         uid,
		Subject:     subject,
		From:        "user@example.com",
		To:          recipient,
		Date:        time.Now().Add(-time.Duration(rand.Intn(15)) * 24 * time.Hour),
		Flags:       []string{"\\Seen"},
		Body:        "Hi,\\n\\nPlease find the information you requested attached.\\n\\nBest regards,\\nUser",
		Attachments: s.generateAttachmentForMessage(uid+100, "sent"), // Offset UID to get different attachments
		Headers: map[string]string{
			"Message-ID": fmt.Sprintf("<sent-%d@testserver.local>", uid),
			"X-Mailer":   "Test IMAP Server",
		},
	}

	msg.Raw = s.generateRawMessage(msg)
	msg.Size = len(msg.Raw)

	return msg
}

// generateDraftMessage creates a sample draft message
func (s *IMAPServer) generateDraftMessage(uid uint32) *Message {
	msg := &Message{
		UID:         uid,
		Subject:     "Draft: Quarterly planning meeting with attachments",
		From:        "user@example.com",
		To:          "team@mycompany.com",
		Date:        time.Now(),
		Flags:       []string{"\\Draft"},
		Body:        "Hi team,\\n\\nI wanted to schedule our quarterly planning meeting for next week. Please find the agenda and budget docs attached.\\n\\n[This is a draft - not sent yet]",
		Attachments: []Attachment{
			{
				Filename:    "meeting_agenda.rtf",
				ContentType: "application/msword",
				Data:        s.generateSampleWord(),
				Encoding:    "base64",
			},
			{
				Filename:    "budget_2024.csv",
				ContentType: "application/vnd.ms-excel",
				Data:        s.generateSampleExcel(),
				Encoding:    "base64",
			},
			{
				Filename:    "charts.png",
				ContentType: "image/png",
				Data:        s.generateSampleImage(),
				Encoding:    "base64",
			},
		},
		Headers: map[string]string{
			"Message-ID": fmt.Sprintf("<draft-%d@testserver.local>", uid),
			"X-Mailer":   "Test IMAP Server",
		},
	}

	msg.Raw = s.generateRawMessage(msg)
	msg.Size = len(msg.Raw)

	return msg
}

// CreateTestServer creates a pre-configured test server with sample data
func CreateTestServer(addr string) *IMAPServer {
	server := NewIMAPServer(addr)
	
	// Add test users
	server.AddUser("testuser@example.com", "password123")
	server.AddUser("alice@company.com", "secret456")
	server.AddUser("bob@business.com", "pass789")
	
	// Populate with sample data
	server.PopulateWithSampleData("testuser@example.com")
	server.PopulateWithSampleData("alice@company.com")
	server.PopulateWithSampleData("bob@business.com")
	
	return server
}

// GetServerStats returns statistics about the server
func (s *IMAPServer) GetServerStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"running":    s.running,
		"address":    s.addr,
		"user_count": len(s.users),
		"users":      make(map[string]interface{}),
	}

	for username, user := range s.users {
		userStats := map[string]interface{}{
			"folder_count": len(user.Folders),
			"folders":      make(map[string]interface{}),
		}

		for folderName, folder := range user.Folders {
			userStats["folders"].(map[string]interface{})[folderName] = map[string]interface{}{
				"message_count": len(folder.Messages),
				"flags":         folder.Flags,
			}
		}

		stats["users"].(map[string]interface{})[username] = userStats
	}

	return stats
}