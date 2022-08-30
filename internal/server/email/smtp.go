package email

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

type Client struct {
	*tls.Conn
	reader *bufio.Reader
	sync.Mutex
	skipTLSVerify bool
}

type Message struct {
	FromName    string
	FromAddress string
	ToName      string
	ToAddress   string
	Subject     string
	PlainBody   []byte
	HTMLBody    []byte
}

var client = &Client{}

func (c *Client) connect() (err error) {
	if c.Conn != nil {
		return nil
	}

	if SendgridAPIKey == "" {
		return fmt.Errorf("Sendgrid API key is not set")
	}

	conn, err := tls.Dial("tcp", SMTPServer, &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		// nolint:gosec
		InsecureSkipVerify: c.skipTLSVerify, // only used by tests
	})
	if err != nil {
		return err
	}

	err = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}
	c.Conn = conn

	defer func() {
		if err != nil {
			c.Conn = nil
		}
	}()

	auth := smtp.PlainAuth("", "apikey", SendgridAPIKey, SMTPServer)

	client, err := smtp.NewClient(conn, SMTPServer)
	if err != nil {
		return err
	}

	if err = client.Auth(auth); err != nil {
		return err
	}

	c.reader = bufio.NewReader(c.Conn)

	return nil
}

func base64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// readln reads a line of text ending in \n. if expected is empty it returns the result.
// if expected is not empty, the read string must match or start with the expected string
func (c *Client) readln(expected string) (result string, err error) {
	defer func() {
		if err != nil {
			c.Conn = nil
		}
	}()

	if err := c.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", fmt.Errorf("set deadline: %w", err)
	}

	s, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	s = strings.TrimRight(s, "\r\n")

	if expected != "" && !strings.HasPrefix(s, expected) {
		return s, fmt.Errorf("Unexpected value read: %q while waiting for %q", s, expected)
	}

	return s, nil
}

func (c *Client) writeln(s string) (err error) {
	defer func() {
		if err != nil {
			c.Conn = nil
		}
	}()

	s += "\r\n"
	if err = c.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	written, err := c.Write([]byte(s))
	if err != nil {
		return fmt.Errorf("write auth login: %w", err)
	}
	if written != len(s) {
		return fmt.Errorf("partial write %d bytes of %d", written, len(s))
	}

	return nil
}

// SendSMTP sends an email message
func SendSMTP(msg Message) error {
	client.Lock()
	defer client.Unlock()

	if err := client.connect(); err != nil {
		return err
	}

	if err := client.writeln(fmt.Sprintf("MAIL FROM: %s", msg.FromAddress)); err != nil {
		return fmt.Errorf("write mail from: %w", err)
	}
	if _, err := client.readln("250 "); err != nil { // Sender address accepted
		return err
	}

	if err := client.writeln(fmt.Sprintf("RCPT TO: %s", msg.ToAddress)); err != nil {
		return fmt.Errorf("write rcpt to: %w", err)
	}
	if _, err := client.readln("250 "); err != nil { // Recipient address accepted
		return err
	}

	if err := client.writeln("DATA"); err != nil {
		return err
	}
	if _, err := client.readln("354 "); err != nil {
		return err
	}

	if len(msg.ToName) > 0 {
		if err := client.writeln(fmt.Sprintf("To: %q <%s>", msg.ToName, msg.ToAddress)); err != nil {
			return fmt.Errorf("write to: %w", err)
		}
	}
	if len(msg.FromName) > 0 {
		if err := client.writeln(fmt.Sprintf("From: %q <%s>", msg.FromName, msg.FromAddress)); err != nil {
			return fmt.Errorf("write from: %w", err)
		}
	}
	if len(msg.Subject) > 0 {
		if err := client.writeln(fmt.Sprintf("Subject: %s", msg.Subject)); err != nil {
			return fmt.Errorf("write subject: %w", err)
		}
	}

	// mime multipart
	if err := client.writeln("MIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=c3VwYWhpbmZyYQ\r\n"); err != nil {
		return err
	}

	// plain
	if len(msg.PlainBody) > 0 {
		if err := client.writeln("--c3VwYWhpbmZyYQ\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n"); err != nil {
			return err
		}
		if err := client.writeln(base64encode(string(msg.PlainBody))); err != nil {
			return err
		}
	}

	// html
	if len(msg.HTMLBody) > 0 {
		if err := client.writeln("--c3VwYWhpbmZyYQ\r\nContent-Type: text/html\r\nContent-Transfer-Encoding: base64\r\n"); err != nil {
			return err
		}
		if err := client.writeln(base64encode(string(msg.HTMLBody))); err != nil {
			return err
		}
	}

	// end mime
	if err := client.writeln("--c3VwYWhpbmZyYQ--\r\n"); err != nil {
		return err
	}

	if err := client.writeln("."); err != nil {
		return fmt.Errorf("write send line: %w", err)
	}

	result, err := client.readln("")
	if err != nil {
		return fmt.Errorf("reading send result: %w", err)
	}

	if !strings.HasPrefix(result, "250 ") { // Ok
		return fmt.Errorf("expected 250 Ok result, but got %q", result)
	}
	return nil
}
