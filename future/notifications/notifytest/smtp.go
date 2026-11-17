package notifytest

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
)

// SMTPMode selects how the fake server handles TLS.
type SMTPMode int

const (
	// SMTPStartTLS advertises and honours STARTTLS.
	SMTPStartTLS SMTPMode = iota
	// SMTPImplicit speaks TLS from the first byte.
	SMTPImplicit
	// SMTPPlainOnly never offers STARTTLS (a downgrade the client must refuse).
	SMTPPlainOnly
)

// Received is one accepted message.
type Received struct {
	From string
	To   []string
	Data string
	Auth string // "user" the client authenticated as, if any
}

// SMTPServer is a minimal RFC 5321 server for client tests.
type SMTPServer struct {
	Addr string
	// Behaviour knobs; set before connections arrive.
	User, Pass string // when set, AUTH is required
	RcptReply  string // e.g. "550 5.1.1 no such user"; default 250
	DataReply  string // reply after DATA, e.g. "451 4.3.0 try later"; default 250
	Mode       SMTPMode

	ln   net.Listener
	cert tls.Certificate
	mu   sync.Mutex
	msgs []Received
	log  []string
	wg   sync.WaitGroup
}

// StartSMTP listens on 127.0.0.1:0.
func StartSMTP(mode SMTPMode, cert tls.Certificate) *SMTPServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	s := &SMTPServer{Addr: ln.Addr().String(), Mode: mode, ln: ln, cert: cert}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func() { defer s.wg.Done(); s.serve(c) }()
		}
	}()
	return s
}

// Close stops the server.
func (s *SMTPServer) Close() { _ = s.ln.Close(); s.wg.Wait() }

// Messages returns accepted messages.
func (s *SMTPServer) Messages() []Received {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Received(nil), s.msgs...)
}

// Transcript returns every client line seen, for leak assertions.
func (s *SMTPServer) Transcript() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.log, "\n")
}

func (s *SMTPServer) serve(raw net.Conn) {
	defer raw.Close()
	conn := raw
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{s.cert}, MinVersion: tls.VersionTLS12}
	if s.Mode == SMTPImplicit {
		conn = tls.Server(raw, tlsCfg)
	}
	r := bufio.NewReader(conn)
	w := func(f string, a ...any) { fmt.Fprintf(conn, f+"\r\n", a...) }
	w("220 fake.smtp ESMTP")
	var from, authed string
	var to []string
	secure := s.Mode == SMTPImplicit
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		up := strings.ToUpper(line)
		s.mu.Lock()
		if strings.HasPrefix(up, "AUTH ") {
			s.log = append(s.log, "AUTH <redacted-by-fixture>")
		} else {
			s.log = append(s.log, line)
		}
		s.mu.Unlock()
		switch {
		case strings.HasPrefix(up, "EHLO"), strings.HasPrefix(up, "HELO"):
			w("250-fake.smtp")
			if s.Mode == SMTPStartTLS && !secure {
				w("250-STARTTLS")
			}
			if s.User != "" && (secure || s.Mode == SMTPPlainOnly) {
				w("250-AUTH PLAIN LOGIN")
			}
			w("250 8BITMIME")
		case up == "STARTTLS":
			if s.Mode != SMTPStartTLS || secure {
				w("502 not offered")
				continue
			}
			w("220 ready")
			t := tls.Server(raw, tlsCfg)
			if err := t.Handshake(); err != nil {
				return
			}
			conn, secure = t, true
			r = bufio.NewReader(t)
		case strings.HasPrefix(up, "AUTH PLAIN"):
			parts := strings.Fields(line)
			var blob string
			if len(parts) >= 3 {
				blob = parts[2]
			} else {
				w("334 ")
				l, _ := r.ReadString('\n')
				blob = strings.TrimSpace(l)
			}
			b, _ := base64.StdEncoding.DecodeString(blob)
			f := strings.Split(string(b), "\x00")
			if len(f) == 3 && f[1] == s.User && f[2] == s.Pass {
				authed = f[1]
				w("235 2.7.0 ok")
			} else {
				w("535 5.7.8 bad credentials")
			}
		case strings.HasPrefix(up, "AUTH LOGIN"):
			w("334 VXNlcm5hbWU6")
			u, _ := r.ReadString('\n')
			w("334 UGFzc3dvcmQ6")
			p, _ := r.ReadString('\n')
			ub, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(u))
			pb, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(p))
			if string(ub) == s.User && string(pb) == s.Pass {
				authed = s.User
				w("235 2.7.0 ok")
			} else {
				w("535 5.7.8 bad credentials")
			}
		case strings.HasPrefix(up, "MAIL FROM:"):
			if s.User != "" && authed == "" {
				w("530 5.7.0 auth required")
				continue
			}
			from = strings.TrimSpace(line[len("MAIL FROM:"):])
			to = nil
			w("250 ok")
		case strings.HasPrefix(up, "RCPT TO:"):
			if s.RcptReply != "" {
				w("%s", s.RcptReply)
				continue
			}
			to = append(to, strings.TrimSpace(line[len("RCPT TO:"):]))
			w("250 ok")
		case up == "DATA":
			w("354 go ahead")
			var sb strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				sb.WriteString(strings.TrimPrefix(l, "."))
			}
			if s.DataReply != "" && !strings.HasPrefix(s.DataReply, "250") {
				w("%s", s.DataReply)
				continue
			}
			s.mu.Lock()
			s.msgs = append(s.msgs, Received{From: from, To: append([]string(nil), to...), Data: sb.String(), Auth: authed})
			s.mu.Unlock()
			w("250 2.0.0 queued")
		case up == "RSET", up == "NOOP":
			w("250 ok")
		case up == "QUIT":
			w("221 bye")
			return
		default:
			w("500 unknown")
		}
	}
}
