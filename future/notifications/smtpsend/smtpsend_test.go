package smtpsend

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/notifications"
	"github.com/djshellshoxxx/circula/future/notifications/notifytest"
)

func smtpTransportFixture(t *testing.T, mode notifytest.SMTPMode, configure func(*Config, *notifytest.SMTPServer)) (*Sender, *notifytest.SMTPServer) {
	t.Helper()
	pki := notifytest.NewPKI()
	server := notifytest.StartSMTP(mode, pki.Leaf("smtp.example.test"))
	t.Cleanup(server.Close)
	server.User, server.Pass = "synthetic-operator", "synthetic-password-canary"
	cfg := Config{
		Host: "smtp.example.test", From: "notifications@circula.example.test",
		Username: server.User, Password: notifications.NewSecret(server.Pass),
		RootCAs: pki.Pool, Timeout: 2 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, server.Addr)
		},
	}
	if mode == notifytest.SMTPImplicit {
		cfg.TLS = Implicit
	}
	if configure != nil {
		configure(&cfg, server)
	}
	sender, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return sender, server
}

func smtpTransportMessage() notifications.EmailMessage {
	return notifications.EmailMessage{
		To:             "recipient-canary@example.test",
		Subject:        notifications.GenericSubject,
		Body:           "Open Circula to see new activity.",
		MessageID:      "message-canary@circula.example.test",
		UnsubscribeURL: "https://circula.example.test/unsubscribe?token=token-canary",
	}
}

func TestSendEmailOverVerifiedTLS(t *testing.T) {
	for _, mode := range []notifytest.SMTPMode{notifytest.SMTPStartTLS, notifytest.SMTPImplicit} {
		name := "starttls"
		if mode == notifytest.SMTPImplicit {
			name = "implicit"
		}
		t.Run(name, func(t *testing.T) {
			sender, server := smtpTransportFixture(t, mode, nil)
			message := smtpTransportMessage()
			if err := sender.SendEmail(context.Background(), message); err != nil {
				t.Fatal(err)
			}
			received := server.Messages()
			if len(received) != 1 || received[0].Auth != server.User || len(received[0].To) != 1 {
				t.Fatalf("delivery envelope/auth mismatch: messages=%d", len(received))
			}
			from, to := strings.Fields(received[0].From), strings.Fields(received[0].To[0])
			if len(from) == 0 || len(to) == 0 || from[0] != "<notifications@circula.example.test>" || to[0] != "<"+message.To+">" {
				t.Fatal("sender or recipient differs from the configured envelope")
			}
			parsed, err := mail.ReadMessage(strings.NewReader(received[0].Data))
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Header.Get("Subject") != notifications.GenericSubject || parsed.Header.Get("Message-Id") != "<"+message.MessageID+">" {
				t.Fatal("fixed subject or stable message ID was not preserved")
			}
			if !strings.Contains(parsed.Header.Get("List-Unsubscribe"), message.UnsubscribeURL) || parsed.Header.Get("List-Unsubscribe-Post") != "List-Unsubscribe=One-Click" {
				t.Fatal("one-click unsubscribe headers were not preserved")
			}
			var body io.Reader = parsed.Body
			switch strings.ToLower(parsed.Header.Get("Content-Transfer-Encoding")) {
			case "quoted-printable":
				body = quotedprintable.NewReader(body)
			case "base64":
				body = base64.NewDecoder(base64.StdEncoding, body)
			}
			data, err := io.ReadAll(body)
			if err != nil || strings.TrimSpace(string(data)) != message.Body {
				t.Fatalf("message body changed: %v", err)
			}
			if mode == notifytest.SMTPStartTLS {
				transcript := server.Transcript()
				tlsAt, authAt := strings.Index(transcript, "STARTTLS"), strings.Index(transcript, "AUTH ")
				if tlsAt < 0 || authAt <= tlsAt {
					t.Fatal("authentication did not follow the mandatory TLS upgrade")
				}
			}
		})
	}
}

func TestSendEmailRefusesTLSFailuresBeforeCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode notifytest.SMTPMode
		edit func(*Config, *notifytest.SMTPServer)
	}{
		{name: "downgrade", mode: notifytest.SMTPPlainOnly},
		{name: "wrong certificate name", mode: notifytest.SMTPStartTLS, edit: func(c *Config, _ *notifytest.SMTPServer) { c.ServerName = "wrong.example.test" }},
		{name: "untrusted certificate", mode: notifytest.SMTPStartTLS, edit: func(c *Config, _ *notifytest.SMTPServer) { c.RootCAs = notifytest.NewPKI().Pool }},
		{name: "implicit plaintext refusal", mode: notifytest.SMTPPlainOnly, edit: func(c *Config, _ *notifytest.SMTPServer) { c.TLS = Implicit }},
		{name: "implicit wrong certificate name", mode: notifytest.SMTPImplicit, edit: func(c *Config, _ *notifytest.SMTPServer) { c.ServerName = "wrong.example.test" }},
		{name: "implicit untrusted certificate", mode: notifytest.SMTPImplicit, edit: func(c *Config, _ *notifytest.SMTPServer) { c.RootCAs = notifytest.NewPKI().Pool }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender, server := smtpTransportFixture(t, tc.mode, tc.edit)
			err := sender.SendEmail(context.Background(), smtpTransportMessage())
			assertSMTPClass(t, err, notifications.ClassTransient)
			transcript := server.Transcript()
			if len(server.Messages()) != 0 || strings.Contains(transcript, "AUTH ") || strings.Contains(transcript, "MAIL FROM:") {
				t.Fatal("TLS failure allowed authentication or message delivery")
			}
		})
	}
}

func TestSendEmailImplicitDefaultDialer(t *testing.T) {
	sender, server := smtpTransportFixture(t, notifytest.SMTPImplicit, func(c *Config, s *notifytest.SMTPServer) {
		host, port, err := net.SplitHostPort(s.Addr)
		if err != nil {
			t.Fatal(err)
		}
		c.Port, err = strconv.Atoi(port)
		if err != nil {
			t.Fatal(err)
		}
		c.Host, c.ServerName, c.DialContext = host, "smtp.example.test", nil
	})
	if err := sender.SendEmail(context.Background(), smtpTransportMessage()); err != nil {
		t.Fatal(err)
	}
	if received := server.Messages(); len(received) != 1 || received[0].Auth != server.User {
		t.Fatal("default implicit TLS did not authenticate and deliver")
	}
}

func TestSendEmailClassifiesProviderFailuresWithoutLeaking(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edit  func(*Config, *notifytest.SMTPServer)
		class notifications.SendClass
	}{
		{name: "recipient gone", class: notifications.ClassTargetGone, edit: func(_ *Config, s *notifytest.SMTPServer) { s.RcptReply = "550 recipient-canary@example.test not found" }},
		{name: "recipient temporarily unavailable", class: notifications.ClassTransient, edit: func(_ *Config, s *notifytest.SMTPServer) {
			s.RcptReply = "450 recipient-canary@example.test unavailable"
		}},
		{name: "temporary data failure", class: notifications.ClassTransient, edit: func(_ *Config, s *notifytest.SMTPServer) { s.DataReply = "451 token-canary retry later" }},
		{name: "permanent data failure", class: notifications.ClassPermanent, edit: func(_ *Config, s *notifytest.SMTPServer) { s.DataReply = "550 token-canary rejected" }},
		{name: "operator authentication", class: notifications.ClassTransient, edit: func(c *Config, _ *notifytest.SMTPServer) {
			c.Password = notifications.NewSecret("incorrect-password-canary")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender, server := smtpTransportFixture(t, notifytest.SMTPStartTLS, tc.edit)
			err := sender.SendEmail(context.Background(), smtpTransportMessage())
			assertSMTPClass(t, err, tc.class)
			if len(server.Messages()) != 0 {
				t.Fatal("failed exchange was recorded as a delivered message")
			}
		})
	}
}

func TestSendEmailRejectsInjectedHeadersBeforeDial(t *testing.T) {
	for _, edit := range []func(*notifications.EmailMessage){
		func(m *notifications.EmailMessage) { m.To += "\r\nBcc: another@example.test" },
		func(m *notifications.EmailMessage) { m.Subject += "\r\nBcc: another@example.test" },
		func(m *notifications.EmailMessage) { m.MessageID += "\r\nBcc: another@example.test" },
	} {
		dials := 0
		sender, err := New(Config{Host: "smtp.example.test", From: "notifications@example.test", DialContext: func(context.Context, string, string) (net.Conn, error) {
			dials++
			return nil, errors.New("must not dial")
		}})
		if err != nil {
			t.Fatal(err)
		}
		message := smtpTransportMessage()
		edit(&message)
		if err := sender.SendEmail(context.Background(), message); err == nil || dials != 0 {
			t.Fatalf("injected header escaped validation: error=%v dials=%d", err, dials)
		}
	}
}

func TestSendEmailPreservesCancellation(t *testing.T) {
	sender, err := New(Config{Host: "smtp.example.test", From: "notifications@example.test", DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return nil, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sender.SendEmail(ctx, smtpTransportMessage())
	assertSMTPClass(t, err, notifications.ClassTransient)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("send error lost context cancellation")
	}
}

func TestSendEmailConcurrentMessagesRemainIndependent(t *testing.T) {
	sender, server := smtpTransportFixture(t, notifytest.SMTPStartTLS, nil)
	const count = 4
	errors := make(chan error, count)
	for i := 0; i < count; i++ {
		go func(i int) {
			message := smtpTransportMessage()
			message.MessageID = "concurrent-" + strconv.Itoa(i) + "@circula.example.test"
			errors <- sender.SendEmail(context.Background(), message)
		}(i)
	}
	for i := 0; i < count; i++ {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	received := server.Messages()
	if len(received) != count {
		t.Fatalf("received %d messages, want %d", len(received), count)
	}
	ids := make(map[string]bool)
	for _, message := range received {
		parsed, err := mail.ReadMessage(strings.NewReader(message.Data))
		if err != nil {
			t.Fatal(err)
		}
		id := parsed.Header.Get("Message-Id")
		if id == "" || ids[id] {
			t.Fatal("concurrent sends reused message identity")
		}
		ids[id] = true
	}
}

func assertSMTPClass(t *testing.T, err error, class notifications.SendClass) {
	t.Helper()
	var sendErr *notifications.SendError
	if !errors.As(err, &sendErr) || sendErr == nil || sendErr.Class != class {
		t.Fatalf("error=%v (%T), want class %s", err, err, class)
	}
	for _, secret := range []string{"recipient-canary", "password-canary", "token-canary"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("provider error leaked %s", secret)
		}
	}
}
