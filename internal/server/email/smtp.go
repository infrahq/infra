package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/smtp"
	"time"
)

type Client struct {
	*tls.Conn
	Client        *smtp.Client
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

var testClient *Client = nil

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

	c.Client = client

	if err = client.Auth(auth); err != nil {
		return err
	}

	return nil
}

func base64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func writeln(w io.WriteCloser, s string, args ...any) error {
	_, err := w.Write([]byte(fmt.Sprintf(s, args...) + "\r\n"))
	return err
}

// SendSMTP sends an email message
func SendSMTP(msg Message) error {
	client := &Client{} // setup a new client for each smtp send

	if testClient != nil {
		client = testClient
	}

	if err := client.connect(); err != nil {
		return err
	}

	if err := client.Client.Mail(msg.FromAddress); err != nil {
		return err
	}
	if err := client.Client.Rcpt(msg.ToAddress); err != nil {
		return err
	}
	w, err := client.Client.Data()
	if err != nil {
		return err
	}

	if err := writeln(w, `X-SMTPAPI: {"filters":{"bypass_list_management":{"settings":{"enable":1}}}}`); err != nil {
		return fmt.Errorf("mail header: %w", err)
	}
	if len(msg.ToName) > 0 {
		if err := writeln(w, "To: %q <%s>", msg.ToName, msg.ToAddress); err != nil {
			return fmt.Errorf("write to: %w", err)
		}
	}
	if len(msg.FromName) > 0 {
		if err := writeln(w, "From: %q <%s>", msg.FromName, msg.FromAddress); err != nil {
			return fmt.Errorf("write from: %w", err)
		}
	}
	if len(msg.Subject) > 0 {
		if err := writeln(w, "Subject: %s", msg.Subject); err != nil {
			return fmt.Errorf("write subject: %w", err)
		}
	}

	// mime multipart
	if err := writeln(w, "MIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=c3VwYWhpbmZyYQ\r\n"); err != nil {
		return err
	}

	// plain
	if len(msg.PlainBody) > 0 {
		if err := writeln(w, "--c3VwYWhpbmZyYQ\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n"); err != nil {
			return err
		}
		if err := writeln(w, base64encode(string(msg.PlainBody))); err != nil {
			return err
		}
	}

	// html
	if len(msg.HTMLBody) > 0 {
		if err := writeln(w, "--c3VwYWhpbmZyYQ\r\nContent-Type: text/html\r\nContent-Transfer-Encoding: base64\r\n"); err != nil {
			return err
		}
		if err := writeln(w, base64encode(string(msg.HTMLBody))); err != nil {
			return err
		}
	}

	// end mime
	if err := writeln(w, "--c3VwYWhpbmZyYQ--\r\n"); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return client.Client.Quit()
}
