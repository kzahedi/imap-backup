package imap

import (
	"crypto/tls"
	"fmt"
	"imap-backup/internal/auth"
	"imap-backup/internal/charset"
	"imap-backup/internal/config"
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

type Client struct {
	conn    *client.Client
	account config.Account
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

func NewClient(account config.Account) (*Client, error) {
	var c *client.Client
	var err error

	addr := fmt.Sprintf("%s:%d", account.Host, account.Port)

	if account.UseSSL {
		c, err = client.DialTLS(addr, &tls.Config{})
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	// Authenticate (try OAuth2 first, then password)
	if err := authenticateClient(c, account); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return &Client{
		conn:    c,
		account: account,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ListFolders() ([]*Folder, error) {
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
	// Get the raw message - RFC822 fetch puts data in msg.Body with nil key
	var raw []byte
	
	
	// Look for RFC822 data (stored with nil key when using FetchRFC822)
	if r, ok := msg.Body[nil]; ok && r != nil {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read RFC822 body: %w", err)
		}
		raw = data
	} else {
		// Fallback: try any available body section
		for section, r := range msg.Body {
			if r != nil {
				data, err := io.ReadAll(r)
				if err != nil {
					log.Printf("Failed to read body section %v: %v", section, err)
					continue
				}
				raw = data
				break
			}
		}
	}
	
	if len(raw) == 0 {
		return nil, fmt.Errorf("no message body found")
	}

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

	// Extract headers
	headers := make(map[string][]string)
	headerFields := entity.Header.Fields()
	for headerFields.Next() {
		key := headerFields.Key()
		value := headerFields.Value()
		headers[key] = append(headers[key], value)
	}

	// Parse as mail message
	mailReader := mail.NewReader(entity)
	
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

	// Extract body and attachments with charset handling
	for {
		part, err := mailReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			// Log charset errors but continue processing
			if strings.Contains(err.Error(), "charset") || strings.Contains(err.Error(), "unknown charset") {
				log.Printf("Charset error in message UID %d: %v (continuing with raw data)", msg.Uid, err)
				break
			}
			return nil, fmt.Errorf("failed to read message part: %w", err)
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			// This is the message body
			body, charset := c.readBodyWithCharset(part.Body, h.Get("Content-Type"))
			
			contentType := h.Get("Content-Type")
			if strings.Contains(contentType, "text/html") {
				parsedMsg.HTMLBody = body
			} else {
				parsedMsg.Body = body
			}
			
			// Log charset info for debugging
			if charset != "" && charset != "utf-8" {
				log.Printf("Decoded body from charset %s for message UID %d", charset, msg.Uid)
			}

		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			if filename == "" {
				filename = "untitled"
			}

			data, err := io.ReadAll(part.Body)
			if err != nil {
				log.Printf("Failed to read attachment %s in message UID %d: %v", filename, msg.Uid, err)
				continue
			}

			attachment := Attachment{
				Filename:    filename,
				ContentType: h.Get("Content-Type"),
				Data:        data,
			}
			parsedMsg.Attachments = append(parsedMsg.Attachments, attachment)
		}
	}

	return parsedMsg, nil
}

func getAddressString(addresses []*imap.Address) string {
	var result []string
	for _, addr := range addresses {
		if addr.PersonalName != "" {
			result = append(result, fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			result = append(result, fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName))
		}
	}
	return strings.Join(result, ", ")
}

// authenticateClient attempts to authenticate with OAuth2 first, then falls back to password
func authenticateClient(c *client.Client, account config.Account) error {
	// Detect authentication type
	authType := auth.DetectAuthType(account.Username)
	
	if authType == "oauth2" {
		// Try OAuth2 authentication
		if err := authenticateOAuth2(c, account); err != nil {
			log.Printf("OAuth2 authentication failed for %s: %v", account.Username, err)
			log.Printf("Falling back to password authentication...")
		} else {
			return nil // OAuth2 successful
		}
	}
	
	// Fall back to password authentication
	if account.Password == "" {
		return fmt.Errorf("no password provided for account %s", account.Username)
	}
	
	return c.Login(account.Username, account.Password)
}

// authenticateOAuth2 performs OAuth2 authentication
func authenticateOAuth2(c *client.Client, account config.Account) error {
	// Try to get OAuth2 token from Mac's keychain/Internet Accounts
	token, err := auth.GetOAuth2TokenFromMac(account.Username, "Gmail")
	if err != nil {
		// Try Internet Accounts
		token, err = auth.GetOAuth2TokenFromAccounts(account.Username)
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
	}
	
	// Check if token is expired and refresh if needed
	if auth.IsTokenExpired(token) && token.RefreshToken != "" {
		config := auth.GetGoogleOAuth2Config()
		newToken, err := auth.RefreshOAuth2Token(config, token)
		if err != nil {
			return fmt.Errorf("failed to refresh OAuth2 token: %w", err)
		}
		token = newToken
	}
	
	// Authenticate using XOAUTH2 mechanism
	return c.Authenticate(sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: account.Username,
		Token:    token.AccessToken,
	}))
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