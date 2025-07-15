package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"imap-backup/internal/auth"
	"imap-backup/internal/charset"
	"imap-backup/internal/config"
	"imap-backup/internal/security"
	"io"
	"log"
	"mime"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
)

const (
	// Connection timeouts
	DefaultDialTimeout = 30 * time.Second
	DefaultReadTimeout = 60 * time.Second
	DefaultWriteTimeout = 60 * time.Second
	
	// Message processing limits
	MaxMessageSize = 50 * 1024 * 1024 // 50MB
	MaxConcurrentMessages = 10
	
	// IMAP operation timeouts
	IMAPSelectTimeout = 30 * time.Second
	IMAPFetchTimeout = 5 * time.Minute
)

type Client struct {
	conn        *client.Client
	account     config.Account
	rateLimiter *RateLimiter
}

type Folder struct {
	Name       string
	Delimiter  string
	Attributes []string
}

type Message struct {
	UID         uint32
	Subject     string
	From        string
	To          string
	Date        time.Time
	Flags       []string
	Body        string
	HTMLBody    string
	Headers     map[string][]string
	Attachments []Attachment
	Raw         []byte
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func NewClient(ctx context.Context, account config.Account) (*Client, error) {
	// Validate account configuration
	if err := validateAccount(account); err != nil {
		return nil, fmt.Errorf("invalid account configuration: %w", err)
	}

	var c *client.Client
	var err error

	addr := fmt.Sprintf("%s:%d", account.Host, account.Port)

	if account.UseSSL {
		// Secure TLS configuration
		tlsConfig := &tls.Config{
			ServerName:         account.Host,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			// Add cipher suite preferences for better security
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		
		// Create connection with timeout
		c, err = client.DialTLS(addr, tlsConfig)
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server %s: %w", addr, err)
	}

	// Set timeouts on the connection - removed as go-imap client doesn't expose raw connection

	// Authenticate (try OAuth2 first, then password)
	if err := authenticateClient(ctx, c, account); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return &Client{
		conn:        c,
		account:     account,
		rateLimiter: DefaultRateLimiter(),
	}, nil
}

// validateAccount validates account configuration for security
func validateAccount(account config.Account) error {
	if err := security.ValidateHostname(account.Host); err != nil {
		return fmt.Errorf("invalid hostname: %w", err)
	}
	
	if err := security.ValidateUsername(account.Username); err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}
	
	if account.Port <= 0 || account.Port > 65535 {
		return fmt.Errorf("invalid port: %d", account.Port)
	}
	
	if account.Name == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	
	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ListFolders() ([]*Folder, error) {
	// Rate limit the List operation
	ctx, cancel := context.WithTimeout(context.Background(), DefaultReadTimeout)
	defer cancel()
	
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}
	
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.conn.List("", "*", mailboxes)
	}()

	var folders []*Folder
	for m := range mailboxes {
		folder := &Folder{
			Name:       m.Name,
			Delimiter:  m.Delimiter,
			Attributes: m.Attributes,
		}
		folders = append(folders, folder)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}

func (c *Client) GetMessages(folderName string, existingUIDs map[uint32]bool) ([]*Message, error) {
	// Rate limit the Select operation
	ctx, cancel := context.WithTimeout(context.Background(), IMAPSelectTimeout)
	defer cancel()
	
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}
	
	// Select folder using EXAMINE for read-only access (preserves read/unread status)
	mbox, err := c.conn.Select(folderName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	// Search for all messages
	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	// Rate limit the Fetch operation
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), IMAPFetchTimeout)
	defer fetchCancel()
	
	if err := c.rateLimiter.Wait(fetchCtx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded for fetch: %w", err)
	}
	
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.conn.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, imap.FetchRFC822}, messages)
	}()

	var result []*Message
	for msg := range messages {
		// Skip messages with invalid UIDs or if we already have this message
		if msg.Uid == 0 || existingUIDs[msg.Uid] {
			if msg.Uid == 0 {
				log.Printf("Skipping message with invalid UID 0")
			}
			continue
		}

		// Check if envelope exists
		if msg.Envelope == nil {
			log.Printf("Skipping message UID %d: no envelope data", msg.Uid)
			continue
		}

		parsedMsg, err := c.parseMessage(msg)
		if err != nil {
			// Create a minimal message with just the raw data if parsing fails
			log.Printf("Failed to parse message UID %d: %v (saving raw message)", msg.Uid, err)
			
			// Get raw message data for failed parsing
			var raw []byte
			if r, ok := msg.Body[nil]; ok && r != nil {
				data, readErr := io.ReadAll(r)
				if readErr == nil {
					raw = data
				}
			} else {
				// Fallback: try any available body section
				for _, r := range msg.Body {
					if r != nil {
						data, readErr := io.ReadAll(r)
						if readErr == nil {
							raw = data
							break
						}
					}
				}
			}
			
			// Create minimal message structure with safe envelope access
			subject := ""
			from := ""
			to := ""
			date := time.Time{}
			
			if msg.Envelope != nil {
				subject = msg.Envelope.Subject
				from = getAddressString(msg.Envelope.From)
				to = getAddressString(msg.Envelope.To)
				date = msg.Envelope.Date
			}
			
			parsedMsg = &Message{
				UID:     msg.Uid,
				Subject: subject,
				From:    from,
				To:      to,
				Date:    date,
				Flags:   msg.Flags,
				Body:    fmt.Sprintf("(Message parsing failed: %v)", err),
				Raw:     raw,
				Headers: make(map[string][]string),
			}
		}

		result = append(result, parsedMsg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return result, nil
}

func (c *Client) parseMessage(msg *imap.Message) (*Message, error) {
	// Extract raw message data
	raw, err := c.extractRawMessageData(msg)
	if err != nil {
		return nil, err
	}

	// Parse the message entity
	entity, err := c.parseMessageEntity(raw)
	if err != nil {
		return nil, err
	}

	// Extract headers
	headers := c.extractHeaders(entity)

	// Create base message structure
	parsedMsg := &Message{
		UID:     msg.Uid,
		Subject: msg.Envelope.Subject,
		From:    getAddressString(msg.Envelope.From),
		To:      getAddressString(msg.Envelope.To),
		Date:    msg.Envelope.Date,
		Flags:   msg.Flags,
		Headers: headers,
		Raw:     raw,
	}

	// Parse mail parts (body and attachments)
	if err := c.parseMailParts(entity, parsedMsg, msg.Uid); err != nil {
		return nil, err
	}

	return parsedMsg, nil
}

// extractRawMessageData extracts raw message data from IMAP message body
func (c *Client) extractRawMessageData(msg *imap.Message) ([]byte, error) {
	// Look for RFC822 data (stored with nil key when using FetchRFC822)
	if r, ok := msg.Body[nil]; ok && r != nil {
		return c.readLimitedData(r, msg.Uid, "RFC822 body")
	}

	// Fallback: try any available body section
	for section, r := range msg.Body {
		if r != nil {
			data, err := c.readLimitedData(r, msg.Uid, fmt.Sprintf("section %v", section))
			if err != nil {
				log.Printf("Failed to read body section %v: %v", section, err)
				continue
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("no message body found")
}

// readLimitedData reads data with size limits and truncation warnings
func (c *Client) readLimitedData(reader io.Reader, uid uint32, context string) ([]byte, error) {
	limitedReader := io.LimitReader(reader, MaxMessageSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", context, err)
	}

	// Check if message was truncated
	if len(data) == MaxMessageSize {
		log.Printf("Warning: Message UID %d %s truncated at %d bytes", uid, context, MaxMessageSize)
	}

	return data, nil
}

// parseMessageEntity parses raw message data into a message entity
func (c *Client) parseMessageEntity(raw []byte) (*message.Entity, error) {
	// Set our custom charset reader globally for this parsing
	originalCharsetReader := message.CharsetReader
	message.CharsetReader = func(charsetName string, input io.Reader) (io.Reader, error) {
		return charset.NewReader(input, charsetName)
	}
	defer func() {
		message.CharsetReader = originalCharsetReader
	}()

	// Parse the message with charset support
	entity, err := message.Read(strings.NewReader(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return entity, nil
}

// extractHeaders extracts all headers from a message entity
func (c *Client) extractHeaders(entity *message.Entity) map[string][]string {
	headers := make(map[string][]string)
	headerFields := entity.Header.Fields()
	for headerFields.Next() {
		key := headerFields.Key()
		value := headerFields.Value()
		headers[key] = append(headers[key], value)
	}
	return headers
}

// parseMailParts parses mail parts (body and attachments) from the message entity
func (c *Client) parseMailParts(entity *message.Entity, parsedMsg *Message, uid uint32) error {
	mailReader := mail.NewReader(entity)

	for {
		part, err := mailReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			// Log charset errors but continue processing
			if strings.Contains(err.Error(), "charset") || strings.Contains(err.Error(), "unknown charset") {
				log.Printf("Charset error in message UID %d: %v (continuing with raw data)", uid, err)
				break
			}
			return fmt.Errorf("failed to read message part: %w", err)
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			c.processInlinePart(part, h, parsedMsg, uid)
		case *mail.AttachmentHeader:
			c.processAttachmentPart(part, h, parsedMsg, uid)
		}
	}

	return nil
}

// processInlinePart processes inline message parts (text/html body)
func (c *Client) processInlinePart(part *mail.Part, header *mail.InlineHeader, parsedMsg *Message, uid uint32) {
	body, charset := c.readBodyWithCharset(part.Body, header.Get("Content-Type"))
	
	contentType := header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		parsedMsg.HTMLBody = body
	} else {
		parsedMsg.Body = body
	}
	
	// Log charset info for debugging (only for non-UTF-8 charsets)
	if charset != "" && charset != "utf-8" && charset != "UTF-8" {
		log.Printf("Decoded body from charset %s for message UID %d", charset, uid)
	}
}

// processAttachmentPart processes attachment parts
func (c *Client) processAttachmentPart(part *mail.Part, header *mail.AttachmentHeader, parsedMsg *Message, uid uint32) {
	filename, _ := header.Filename()
	if filename == "" {
		filename = "untitled"
	}

	// Use limited reader to prevent memory exhaustion from large attachments
	limitedReader := io.LimitReader(part.Body, MaxMessageSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("Failed to read attachment %s in message UID %d: %v", filename, uid, err)
		return
	}
	
	// Check if attachment was truncated
	if len(data) == MaxMessageSize {
		log.Printf("Warning: Attachment %s in message UID %d truncated at %d bytes", filename, uid, MaxMessageSize)
	}

	attachment := Attachment{
		Filename:    filename,
		ContentType: header.Get("Content-Type"),
		Data:        data,
	}
	parsedMsg.Attachments = append(parsedMsg.Attachments, attachment)
}

func getAddressString(addresses []*imap.Address) string {
	if addresses == nil {
		return ""
	}
	
	var result []string
	for _, addr := range addresses {
		if addr == nil {
			continue
		}
		
		// Validate address components
		if addr.MailboxName == "" || addr.HostName == "" {
			continue // Skip invalid addresses
		}
		
		if addr.PersonalName != "" {
			result = append(result, fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			result = append(result, fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName))
		}
	}
	return strings.Join(result, ", ")
}

// authenticateClient attempts to authenticate with OAuth2 first, then falls back to password
func authenticateClient(ctx context.Context, c *client.Client, account config.Account) error {
	// Detect authentication type
	authType := auth.DetectAuthType(account.Username)
	
	if authType == "oauth2" {
		if err := tryOAuth2Authentication(c, account); err != nil {
			log.Printf("OAuth2 authentication failed for %s: %v", account.Username, err)
			log.Printf("Falling back to password authentication...")
		} else {
			return nil // OAuth2 successful
		}
	}
	
	// Fall back to password authentication
	return authenticateWithPassword(c, account)
}

// tryOAuth2Authentication attempts OAuth2 authentication
func tryOAuth2Authentication(c *client.Client, account config.Account) error {
	return authenticateOAuth2(c, account)
}

// authenticateWithPassword performs password-based authentication
func authenticateWithPassword(c *client.Client, account config.Account) error {
	if account.Password == "" {
		return fmt.Errorf("no password provided for account %s", account.Username)
	}
	
	return c.Login(account.Username, account.Password)
}

// authenticateOAuth2 performs OAuth2 authentication
func authenticateOAuth2(c *client.Client, account config.Account) error {
	// Get OAuth2 token
	token, err := getOAuth2Token(account.Username)
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}
	
	// Refresh token if needed
	token, err = refreshTokenIfNeeded(token)
	if err != nil {
		return fmt.Errorf("failed to refresh OAuth2 token: %w", err)
	}
	
	// Authenticate using XOAUTH2 mechanism
	return c.Authenticate(sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: account.Username,
		Token:    token.AccessToken,
	}))
}

// getOAuth2Token retrieves OAuth2 token from Mac's keychain or Internet Accounts
func getOAuth2Token(username string) (*auth.OAuth2Token, error) {
	// Try to get OAuth2 token from Mac's keychain
	token, err := auth.GetOAuth2TokenFromMac(username, "Gmail")
	if err != nil {
		// Try Internet Accounts
		token, err = auth.GetOAuth2TokenFromAccounts(username)
		if err != nil {
			return nil, err
		}
	}
	return token, nil
}

// refreshTokenIfNeeded refreshes the OAuth2 token if it's expired
func refreshTokenIfNeeded(token *auth.OAuth2Token) (*auth.OAuth2Token, error) {
	if auth.IsTokenExpired(token) && token.RefreshToken != "" {
		config := auth.GetGoogleOAuth2Config()
		newToken, err := auth.RefreshOAuth2Token(config, token)
		if err != nil {
			return nil, err
		}
		return newToken, nil
	}
	return token, nil
}

// readBodyWithCharset reads body content and handles charset decoding
func (c *Client) readBodyWithCharset(body io.Reader, contentType string) (string, string) {
	// Parse content-type to extract charset
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If parsing fails, read as-is
		data, _ := io.ReadAll(body)
		return string(data), ""
	}
	
	charsetName := params["charset"]
	if charsetName == "" {
		// No charset specified, assume UTF-8
		data, _ := io.ReadAll(body)
		return string(data), ""
	}
	
	// Read the raw data first
	data, err := io.ReadAll(body)
	if err != nil {
		return "", charsetName
	}
	
	// Try to decode the charset
	if charset.IsSupported(charsetName) {
		decoded, err := charset.DecodeString(string(data), charsetName)
		if err == nil {
			return decoded, charsetName
		}
		// If decoding fails, log and return original
		log.Printf("Failed to decode charset %s: %v", charsetName, err)
	} else {
		log.Printf("Unsupported charset: %s", charsetName)
	}
	
	// Return original data if charset handling fails
	return string(data), charsetName
}