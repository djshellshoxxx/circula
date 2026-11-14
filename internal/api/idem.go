package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// The API keeps its own replay cache so that a retry returns the exact first
// response with Idempotent-Replayed: true (API-022) and so that commands the
// service does not yet make idempotent still honour the header for the
// retention period of this process. The service's durable idempotency
// (7 days, shared with the HTML adapter) still backs the commands that have it.
// Failures are never cached (API-025, OQ-API-04 chosen: key stays reusable).

type idemRec struct {
	fp     string
	done   bool
	status int
	header http.Header
	body   []byte
	at     time.Time
	person string // for dropping a person's records on erasure (A3-06)
}

type idemStore struct {
	mu  sync.Mutex
	ttl time.Duration
	max int
	m   map[string]*idemRec
}

func newIdemStore(ttl time.Duration, max int) *idemStore {
	return &idemStore{ttl: ttl, max: max, m: map[string]*idemRec{}}
}

type idemState int

const (
	idemNew idemState = iota
	idemReplay
	idemBusy
	idemMismatch
)

func (st *idemStore) begin(key, fp, person string, now time.Time) (idemState, *idemRec) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if rec, ok := st.m[key]; ok {
		if now.Sub(rec.at) < st.ttl {
			switch {
			case rec.fp != fp:
				return idemMismatch, nil
			case !rec.done:
				return idemBusy, nil
			}
			return idemReplay, rec
		}
		delete(st.m, key)
	}
	if len(st.m) >= st.max {
		for k, r := range st.m { // drop expired, then arbitrary completed entries
			if now.Sub(r.at) >= st.ttl || (r.done && len(st.m) >= st.max) {
				delete(st.m, k)
				if len(st.m) < st.max {
					break
				}
			}
		}
	}
	st.m[key] = &idemRec{fp: fp, at: now, person: person}
	return idemNew, nil
}

// dropPerson forgets every record of one person, in flight or done: cached
// replays hold the bodies of their earlier responses (A3-06).
func (st *idemStore) dropPerson(person string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for k, r := range st.m {
		if r.person == person {
			delete(st.m, k)
		}
	}
}

func (st *idemStore) finish(key string, status int, h http.Header, body []byte, keep bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !keep {
		delete(st.m, key)
		return
	}
	if rec := st.m[key]; rec != nil {
		rec.done, rec.status, rec.header, rec.body = true, status, h, body
	}
}

type recorder struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
	wrote  bool
}

func (r *recorder) Header() http.Header { return r.w.Header() }
func (r *recorder) WriteHeader(s int) {
	if !r.wrote {
		r.status, r.wrote = s, true
	}
}
func (r *recorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	return r.buf.Write(b)
}

var replayHeaders = []string{"Content-Type", "Location", "ETag"}

func validKey(k string) bool {
	if len(k) < 1 || len(k) > 255 {
		return false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < 0x20 || k[i] > 0x7e {
			return false
		}
	}
	return true
}

func (s *server) runIdempotent(c *reqCtx, rt route) error {
	key := c.r.Header.Get("Idempotency-Key")
	if key == "" {
		if rt.idem == idemRequired {
			return &apiError{status: 400, code: "idempotency-key-required", detail: "the Idempotency-Key header is required"}
		}
		return rt.h(c)
	}
	if !validKey(key) {
		return &apiError{status: 400, code: "invalid-header", detail: "Idempotency-Key must be 1 to 255 printable ASCII characters"}
	}
	body, err := c.body()
	if err != nil {
		return err
	}
	var canon bytes.Buffer
	if len(body) > 0 {
		if json.Compact(&canon, body) != nil {
			canon.Reset()
			canon.Write(body)
		}
	}
	sum := sha256.Sum256([]byte(c.r.Method + "\x00" + c.r.URL.Path + "\x00" + c.netID + "\x00" + canon.String()))
	fp := hex.EncodeToString(sum[:])
	scoped := c.actor.PersonID + "\x00" + key
	now := s.opt.Now()
	switch state, rec := s.idem.begin(scoped, fp, c.actor.PersonID, now); state {
	case idemMismatch:
		return &apiError{status: 422, code: "idempotency-key-reused", detail: "the key was used with a different request"}
	case idemBusy:
		return &apiError{status: 409, code: "idempotency-request-in-progress",
			detail: "a request with this key is still being processed", retryAfter: "1"}
	case idemReplay:
		for _, h := range replayHeaders {
			if v := rec.header.Get(h); v != "" {
				c.w.Header().Set(h, v)
			}
		}
		c.w.Header().Set("Idempotent-Replayed", "true")
		c.w.WriteHeader(rec.status)
		_, _ = c.w.Write(rec.body)
		return nil
	}
	c.idemKey = key
	rw := &recorder{w: c.w}
	c.w = rw
	finished := false
	defer func() {
		// A panic in the handler must not leave the record "in progress"
		// until it expires: release the key so the client can retry (API-06).
		if !finished {
			c.w = rw.w
			s.idem.finish(scoped, 0, nil, nil, false)
		}
	}()
	herr := rt.h(c)
	finished = true
	c.w = rw.w
	if herr != nil {
		s.idem.finish(scoped, 0, nil, nil, false)
		return herr
	}
	status := rw.status
	if status == 0 {
		status = http.StatusOK
	}
	keep := status >= 200 && status < 300
	hdr := http.Header{}
	for _, h := range replayHeaders {
		if v := c.w.Header().Get(h); v != "" {
			hdr.Set(h, v)
		}
	}
	s.idem.finish(scoped, status, hdr, append([]byte(nil), rw.buf.Bytes()...), keep)
	c.w.WriteHeader(status)
	_, _ = c.w.Write(rw.buf.Bytes())
	return nil
}
