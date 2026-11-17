package notifytest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// Subscriber plays the browser: it owns the P-256 key and auth secret a
// PushSubscription would hold, and can decrypt what the application server sent.
type Subscriber struct {
	priv *ecdh.PrivateKey
	Auth []byte
}

// NewSubscriber creates fresh receiver keys.
func NewSubscriber() *Subscriber {
	k, _ := ecdh.P256().GenerateKey(rand.Reader)
	a := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, a)
	return &Subscriber{priv: k, Auth: a}
}

// P256dh is the base64url public key as PushSubscription.toJSON() reports it.
func (s *Subscriber) P256dh() string {
	return base64.RawURLEncoding.EncodeToString(s.priv.PublicKey().Bytes())
}

// AuthB64 is the base64url auth secret.
func (s *Subscriber) AuthB64() string { return base64.RawURLEncoding.EncodeToString(s.Auth) }

// Decrypt reverses RFC 8291 aes128gcm encryption of a single-record message.
func (s *Subscriber) Decrypt(body []byte) ([]byte, error) {
	if len(body) < 16+4+1+65+16 {
		return nil, errors.New("short body")
	}
	salt := body[:16]
	rs := binary.BigEndian.Uint32(body[16:20])
	idlen := int(body[20])
	if idlen != 65 || len(body) < 21+idlen {
		return nil, errors.New("bad key id")
	}
	asPub := body[21 : 21+idlen]
	ct := body[21+idlen:]
	if uint32(len(ct)) > rs {
		return nil, errors.New("record too large")
	}
	as, err := ecdh.P256().NewPublicKey(asPub)
	if err != nil {
		return nil, err
	}
	secret, err := s.priv.ECDH(as)
	if err != nil {
		return nil, err
	}
	info := append(append([]byte("WebPush: info\x00"), s.priv.PublicKey().Bytes()...), asPub...)
	ikm, err := hkdf.Key(sha256.New, secret, s.Auth, string(info), 32)
	if err != nil {
		return nil, err
	}
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	blk, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(blk)
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	for i := len(pt) - 1; i >= 0; i-- { // strip padding up to the 0x02 delimiter
		if pt[i] == 0 {
			continue
		}
		if pt[i] == 2 {
			return pt[:i], nil
		}
		break
	}
	return nil, errors.New("bad padding")
}

// PushRequest is one request the fake push service received.
type PushRequest struct {
	Path    string
	Header  http.Header
	Body    []byte
	Subject *Subscriber // set when the path was registered for decryption
	Plain   []byte      // decrypted payload, nil if decryption failed
}

// PushServer is a fake push service on a TLS httptest server.
type PushServer struct {
	*httptest.Server
	mu    sync.Mutex
	reqs  []PushRequest
	subs  map[string]*Subscriber
	reply map[string]func(http.ResponseWriter) // per-path override
	// Default status when no override is set (201).
	Status int
}

// StartPush starts the service with cert.
func StartPush(cert tlsCertificate) *PushServer {
	p := &PushServer{subs: map[string]*Subscriber{}, reply: map[string]func(http.ResponseWriter){}, Status: http.StatusCreated}
	p.Server = httptest.NewUnstartedServer(http.HandlerFunc(p.handle))
	p.Server.TLS = tlsConfig(cert)
	p.Server.StartTLS()
	return p
}

// Register associates a path with a Subscriber so bodies are decrypted.
func (p *PushServer) Register(path string, s *Subscriber) {
	p.mu.Lock()
	p.subs[path] = s
	p.mu.Unlock()
}

// ReplyWith overrides the response for a path.
func (p *PushServer) ReplyWith(path string, f func(http.ResponseWriter)) {
	p.mu.Lock()
	p.reply[path] = f
	p.mu.Unlock()
}

// Requests returns what was received.
func (p *PushServer) Requests() []PushRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PushRequest(nil), p.reqs...)
}

func (p *PushServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	p.mu.Lock()
	sub := p.subs[r.URL.Path]
	f := p.reply[r.URL.Path]
	status := p.Status
	req := PushRequest{Path: r.URL.Path, Header: r.Header.Clone(), Body: body, Subject: sub}
	if sub != nil && r.Method == http.MethodPost {
		req.Plain, _ = sub.Decrypt(body)
	}
	p.reqs = append(p.reqs, req)
	p.mu.Unlock()
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if f != nil {
		f(w)
		return
	}
	w.WriteHeader(status)
}
