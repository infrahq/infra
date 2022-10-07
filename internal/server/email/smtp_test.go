package email

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func setupClient(srv net.Listener) {
	SendgridAPIKey = "api-key"
	SMTPServer = srv.Addr().String()
	testClient = &Client{skipTLSVerify: true, deliverExampleEmails: true}
}

func setupSMTPServer(t *testing.T, handler func(t *testing.T, c net.Conn)) net.Listener {
	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyPem := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		assert.NilError(t, err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", "127.0.0.1:", cfg)
	if err != nil {
		assert.NilError(t, err)
	}
	go func() {
		// wait for one connection and then close.
		c, err := listener.Accept()
		assert.NilError(t, err)
		handler(t, c)
		c.Close()
	}()
	return listener
}

type captureSentEmail struct {
	To      string
	From    string
	Subject string
	Text    string
	HTML    string
}

func (e *captureSentEmail) successCase(t *testing.T, c net.Conn) {
	reader := bufio.NewReader(c)
	read := func(msg string) string {
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		s, err := reader.ReadString('\n')
		t.Log("Read:", s, err)
		assert.NilError(t, err, fmt.Sprintf("waiting for read %q", msg))
		s = strings.TrimRight(s, "\r\n")
		if msg != "" {
			if !strings.HasPrefix(s, msg) {
				t.Errorf("expected %q to match message %q", s, msg)
				t.FailNow()
			}
		}
		return s
	}
	write := func(s string) {
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		_, err := c.Write([]byte(s + "\r\n"))
		assert.NilError(t, err, fmt.Sprintf("waiting for write %q", s))
		if err != nil {
			t.FailNow()
		}
	}
	write("220 server ready")
	read("EHLO")
	write("250 Hi")
	msg := read("AUTH PLAIN")
	msg = strings.TrimPrefix(msg, "AUTH PLAIN ")
	decoded, err := base64.StdEncoding.DecodeString(msg)
	assert.NilError(t, err)
	assert.Equal(t, "\x00apikey\x00api-key", string(decoded))
	write("235 Authentication successful")
	assert.Assert(t, strings.HasPrefix(read(""), "MAIL FROM:"))
	write("250 Sender address accepted")
	assert.Assert(t, strings.HasPrefix(read(""), "RCPT TO:"))
	write("250 Recipient address accepted")
	read("DATA")
	write("354 Ready")
	read(`X-SMTPAPI: {"filters":{"bypass_list_management":{"settings":{"enable":1}}}}`)
	for s := read(""); s != "."; s = read("") {
		switch {
		case strings.HasPrefix(s, "To: "):
			e.To = strings.SplitN(s, ": ", 2)[1]
		case strings.HasPrefix(s, "From: "):
			e.From = strings.SplitN(s, ": ", 2)[1]
		case strings.HasPrefix(s, "Subject: "):
			e.Subject = strings.SplitN(s, ": ", 2)[1]
		case strings.HasPrefix(s, "MIME-Version: "):
			// read until end of mime block
			isPlain := true
			for s = read(""); s != "--c3VwYWhpbmZyYQ--"; s = read("") {
				switch s {
				case "Content-Type: multipart/alternative; boundary=c3VwYWhpbmZyYQ":
				case "--c3VwYWhpbmZyYQ":
				case "Content-Transfer-Encoding: base64":
				case "Content-Type: text/plain":
					isPlain = true
				case "Content-Type: text/html":
					isPlain = false
				case "--c3VwYWhpbmZyYQ--":
					break
				case "":
				default:
					b, err := base64.StdEncoding.DecodeString(s)
					assert.NilError(t, err)
					if isPlain {
						e.Text = string(b)
					} else {
						e.HTML = string(b)
					}
				}
			}
			read("") // blank line at the end
		default:
			t.Error("unexpected message: ", s)
		}
	}
	write("250 Ok")
	read("QUIT")
	write("221 bye!")
}

func TestSendEmail(t *testing.T) {
	result := &captureSentEmail{}
	srv := setupSMTPServer(t, result.successCase)
	setupClient(srv)

	err := SendSMTP(Message{
		ToName:      "Steven",
		ToAddress:   "steven@example.com",
		FromName:    "Also Steven",
		FromAddress: "steven@example.com",
		Subject:     "The art of emails",
		PlainBody:   []byte("Hello world\n.\n."),
		HTMLBody:    []byte("<h2> HELLO WORLD <h2>"),
	}, BypassListManagement)
	assert.NilError(t, err)

	expected := &captureSentEmail{
		To:      `"Steven" <steven@example.com>`,
		From:    `"Also Steven" <steven@example.com>`,
		Subject: "The art of emails",
		Text:    "Hello world\n.\n.",
		HTML:    "<h2> HELLO WORLD <h2>",
	}
	assert.DeepEqual(t, result, expected)
}

func TestSendPasswordResetEmail(t *testing.T) {
	result := &captureSentEmail{}
	srv := setupSMTPServer(t, result.successCase)
	setupClient(srv)

	err := SendPasswordResetEmail("", "steven@example.com", PasswordResetData{
		Link: "https://example.com?himom=1",
	})
	assert.NilError(t, err)

	// Run tests with -update to automatically update these expected values.
	// If these expected values get much larger consider changing to golden.Assert
	// to store the expected values in a separate file.
	expectedText := `Someone has requested a password reset for your Infra account. If this was not you, you can safely ignore this email.

Click here to reset your password:
  https://example.com?himom=1
`
	expectedHTML := `
<p>Someone has requested a password reset for your Infra account. If this was not you, you can safely ignore this email.</p>

<p>
  <a href="https://example.com?himom=1">Click here to reset your password</a>
</p>
`
	assert.Equal(t, result.Text, expectedText)
	assert.Equal(t, result.HTML, expectedHTML)
}

func TestSendUserInviteEmail(t *testing.T) {
	result := &captureSentEmail{}
	srv := setupSMTPServer(t, result.successCase)
	setupClient(srv)

	err := SendUserInviteEmail("", "steven@example.com", UserInviteData{
		FromUserName: "joe bill",
		Link:         "https://example.com?himom=1",
	})
	assert.NilError(t, err)

	// Run tests with -update to automatically update these expected values.
	// If these expected values get much larger consider changing to golden.Assert
	// to store the expected values in a separate file.
	expectedText := `joe bill has invited you to Infra!

Infra is the simplest way to manage infrastructure access.

Log in to access your infrastructure:
  https://example.com?himom=1
`
	expectedHTML := `<p>joe bill has invited you to Infra!</p>

<p>Infra is the simplest way to manage infrastructure access.</p>

<p>
  <a href="https://example.com?himom=1">Login to access your infrastructure</a>
</p>`
	assert.Equal(t, result.Text, expectedText)
	assert.Equal(t, result.HTML, expectedHTML)
}

func TestSendSignupEmail(t *testing.T) {
	result := &captureSentEmail{}
	srv := setupSMTPServer(t, result.successCase)
	setupClient(srv)

	err := SendSignupEmail("", "steven@example.com", SignupData{
		Link:        "https://supahdomain.example.com/login",
		WrappedLink: "https://supahdomain.example.com/login",
	})
	assert.NilError(t, err)

	// Run tests with -update to automatically update these expected values.
	// If these expected values get much larger consider changing to golden.Assert
	// to store the expected values in a separate file.
	expectedText := `Welcome to Infra!

You can sign into your account any time from https://supahdomain.example.com/login

Please verify your email address by clicking this link: https://supahdomain.example.com/login
`
	expectedHTML := `<h3>Welcome to Infra!</h3>

<p>
  You can sign into your account any time from 
  <a href="https://supahdomain.example.com/login">https://supahdomain.example.com/login</a>.
</p>

<p>
  Please <a href="https://supahdomain.example.com/login">verify your email address</a>.
</p>`

	assert.Equal(t, result.Text, expectedText)
	assert.Equal(t, result.HTML, expectedHTML)
}

func TestSendForgotDomainsEmail(t *testing.T) {
	result := &captureSentEmail{}
	srv := setupSMTPServer(t, result.successCase)
	setupClient(srv)

	err := SendForgotDomainsEmail("", "hannibal@ateam.org", ForgottenDomainData{
		Organizations: []models.ForgottenDomain{
			{
				OrganizationName:   "A Team",
				OrganizationDomain: "ateam.infrahq.com",
				LastSeenAt:         time.Now(),
			},
		},
	})
	assert.NilError(t, err)

	// Run tests with -update to automatically update these expected values.
	// If these expected values get much larger consider changing to golden.Assert
	// to store the expected values in a separate file.
	expectedText := `Someone has requested links to each of the organizations associated with your Infra account. If this was not you, you can safely ignore this email.

You can sign in to your organization here:
  A Team	https://ateam.infrahq.com/login 	(last seen less than a second ago)

`
	expectedHTML := `
<p>Someone has requested links to each of the organizations associated with your Infra account. If this was not you, you can safely ignore this email.</p>

<p>A Team <a href="https://ateam.infrahq.com/login">https://ateam.infrahq.com/login</a> (last seen less than a second ago)</p>

`
	assert.Equal(t, result.Text, expectedText)
	assert.Equal(t, result.HTML, expectedHTML)
}
