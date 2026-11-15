package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// body reads the request body once (<= limit, API-037) and checks the media
// type when a body is present (API-003, API-006).
func (c *reqCtx) body() ([]byte, error) {
	if c.rawRead {
		return c.raw, nil
	}
	c.rawRead = true
	if c.r.Body == nil || c.r.Body == http.NoBody {
		return nil, nil
	}
	b, err := io.ReadAll(http.MaxBytesReader(c.w, c.r.Body, c.s.opt.MaxBodyBytes))
	if err != nil {
		var mb *http.MaxBytesError
		if errors.As(err, &mb) {
			return nil, errTooLarge
		}
		return nil, invalidBody("request body could not be read")
	}
	if len(b) == 0 {
		return nil, nil
	}
	want := "application/json"
	if c.r.Method == http.MethodPatch {
		want = "application/merge-patch+json"
	}
	mt, params, err := mime.ParseMediaType(c.r.Header.Get("Content-Type"))
	if err != nil || mt != want || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		return nil, &apiError{status: 415, code: "unsupported-media-type", detail: "Content-Type must be " + want + "; charset=utf-8"}
	}
	c.raw = b
	return b, nil
}

// decode strictly parses the body into v: unknown fields, duplicate keys and
// trailing data are rejected (API-003, API-070). Empty bodies are rejected.
func (c *reqCtx) decode(v any) error {
	b, err := c.body()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return invalidBody("a JSON request body is required")
	}
	return strictUnmarshal(b, v)
}

// decodeOptional is decode for commands whose body may be absent.
func (c *reqCtx) decodeOptional(v any) error {
	b, err := c.body()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	return strictUnmarshal(b, v)
}

func strictUnmarshal(b []byte, v any) error {
	if err := checkDuplicateKeys(b); err != nil {
		return invalidBody(err.Error())
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return invalidBody("malformed JSON, unknown field or wrong type: " + sanitize(err.Error()))
	}
	// dec.More is false before a stray closing delimiter ("{}}"), so ask for
	// the next token: anything but end of input is trailing data.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return invalidBody("unexpected data after the JSON document")
	}
	return nil
}

// sanitize keeps decoder messages short and free of Go internals.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "json: ", "")
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// checkDuplicateKeys walks the token stream and rejects repeated object keys.
func checkDuplicateKeys(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	var walk func() error
	walk = func() error {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		d, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch d {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return err
				}
				k, _ := kt.(string)
				if seen[k] {
					return fmt.Errorf("duplicate key %q", k)
				}
				seen[k] = true
				if err := walk(); err != nil {
					return err
				}
			}
		case '[':
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		}
		_, err = dec.Token() // closing delimiter
		return err
	}
	return walk()
}

// parseMinor parses a canonical base-10 integer string of minor units (API-004).
// Positive values only unless allowZero, which admits "0" as well.
func parseMinor(field, s string, allowZero bool) (int64, error) {
	bad := func(d string) error { return validation(field, "invalid-amount", d) }
	if s == "" {
		return 0, bad("required: a string of minor units")
	}
	if len(s) > 19 {
		return 0, bad("too large")
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, bad("must be a canonical base-10 integer string without sign, point, exponent or whitespace")
		}
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, bad("leading zeros are not allowed")
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, bad("does not fit a signed 64-bit integer")
	}
	if v == 0 && !allowZero {
		return 0, bad("must be greater than zero")
	}
	return v, nil
}

func minor(v int64) string { return strconv.FormatInt(v, 10) }

func minorPtr(v *int64) *string {
	if v == nil {
		return nil
	}
	s := minor(*v)
	return &s
}

// ts formats a timestamp per API-005 (RFC 3339 UTC, milliseconds, Z).
func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func tsPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := ts(*t)
	return &s
}

// etagOf builds a strong ETag from the parts.
func etagOf(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return `"` + hex.EncodeToString(h.Sum(nil)[:12]) + `"`
}

// requireIfMatch enforces API-026/027 against the current ETag.
func (c *reqCtx) requireIfMatch(current string) error {
	h := c.r.Header.Get("If-Match")
	if h == "" {
		return &apiError{status: 428, code: "precondition-required", detail: "If-Match header is required"}
	}
	if !ifMatchOK(h, current) {
		return &apiError{status: 412, code: "precondition-failed", detail: "If-Match does not match the current version"}
	}
	return nil
}

func ifMatchOK(header, current string) bool {
	for _, p := range strings.Split(header, ",") {
		p = strings.TrimSpace(p)
		if p == "*" || p == current {
			return true
		}
	}
	return false
}
