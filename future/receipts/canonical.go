package receipts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gowebpki/jcs"
	"io"
	"strconv"
	"strings"
	"time"
)

const MaxDocumentBytes = 16384

// validateDocument checks version and exact integer range before JCS can apply
// IEEE-754 serialization. A digest is not a signature or an anonymity guarantee.
func validateDocument(d Document) error {
	if d.Schema != "circula-receipt/v2" || d.Status != "completed" || d.Explanation != Explanation+" Mutual-credit record; not bank/cash settlement." || d.Revision == 0 || d.Revision > 1<<50 {
		return errors.New("unsupported or invalid receipt")
	}
	completed, e := time.Parse(time.RFC3339Nano, d.CompletedAt)
	if e != nil || completed.UTC().Format(time.RFC3339Nano) != d.CompletedAt {
		return errors.New("invalid completion timestamp")
	}
	if d.Supplement != nil {
		observed, e := time.Parse(time.RFC3339Nano, d.Supplement.ObservedAt)
		if e != nil || observed.UTC().Format(time.RFC3339Nano) != d.Supplement.ObservedAt {
			return errors.New("invalid supplemental timestamp")
		}
	}
	t := Trade{Snapshot: Snapshot{d.Network, d.Transaction, d.Payer, d.Payee, d.Unit, d.Status, d.Amount, completed}, Precision: d.Precision, PayerEntry: d.LedgerEntries[0], PayeeEntry: d.LedgerEntries[1], Revision: d.Revision, Supplement: d.Supplement}
	_, e = project(Actor{d.Network, d.Payer}, d.Transaction, t)
	return e
}

// Canonical is an experimental RFC8785 serializer for the versioned DTO. It
// selects no signature algorithm and grants no independent verification claim.
func (d Document) Canonical() ([]byte, error) {
	if e := validateDocument(d); e != nil {
		return nil, e
	}
	b, e := d.JSON()
	if e != nil || len(b) > MaxDocumentBytes {
		return nil, errors.New("invalid receipt size")
	}
	return canonicalJSON(b, MaxDocumentBytes)
}
func (d Document) Digest() (string, error) {
	b, e := d.Canonical()
	if e != nil {
		return "", e
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// DecodeDocument is an offline bounded parser. It rejects ambiguous JSON before
// DTO decoding and never fetches a URL, private transaction or public key.
func DecodeDocument(r io.Reader) (Document, error) {
	var d Document
	b, e := io.ReadAll(io.LimitReader(r, MaxDocumentBytes+1))
	if e != nil || len(b) > MaxDocumentBytes {
		return d, errors.New("invalid receipt size")
	}
	canonical, e := canonicalJSON(b, MaxDocumentBytes)
	if e != nil {
		return d, e
	}
	dec := json.NewDecoder(bytes.NewReader(canonical))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&d); e != nil {
		return Document{}, e
	}
	if e = validateDocument(d); e != nil {
		return Document{}, e
	}
	// JCS uses IEEE-754 numbers. Check the original tokens against the bounded
	// integer DTO so a fractional value cannot round into an accepted integer.
	var numbers struct {
		Amount    json.RawMessage `json:"amount_minor"`
		Precision json.RawMessage `json:"precision"`
		Revision  json.RawMessage `json:"revision"`
	}
	if e = json.Unmarshal(b, &numbers); e != nil ||
		!sameJSONInteger(numbers.Amount, strconv.FormatInt(d.Amount, 10)) ||
		!sameJSONInteger(numbers.Precision, strconv.Itoa(d.Precision)) ||
		!sameJSONInteger(numbers.Revision, strconv.FormatUint(d.Revision, 10)) {
		return Document{}, errors.New("inexact receipt integer")
	}
	reconstructed, e := d.Canonical()
	if e != nil || !bytes.Equal(reconstructed, canonical) {
		return Document{}, errors.New("nonconforming receipt schema")
	}
	return d, nil
}

// sameJSONInteger compares a syntax-checked JSON number with an already
// validated nonnegative integer. Decimal/exponent spellings remain valid when
// their exact value agrees; no large integer or power-of-ten allocation is used.
func sameJSONInteger(raw json.RawMessage, integer string) bool {
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return false
	}
	negative := strings.HasPrefix(token, "-")
	token = strings.TrimPrefix(token, "-")
	exponent := "0"
	if at := strings.IndexAny(token, "eE"); at >= 0 {
		exponent, token = token[at+1:], token[:at]
	}
	fractionDigits := 0
	if at := strings.IndexByte(token, '.'); at >= 0 {
		fractionDigits = len(token) - at - 1
		token = token[:at] + token[at+1:]
	}
	digits := strings.TrimLeft(token, "0")
	if digits == "" {
		return integer == "0"
	}
	if negative || integer == "0" {
		return false
	}
	significant := strings.TrimRight(digits, "0")
	expected := strings.TrimRight(integer, "0")
	if significant != expected {
		return false
	}
	shift := fractionDigits + len(integer) - len(expected) - len(digits) + len(significant)
	parsed, err := strconv.ParseInt(exponent, 10, 64)
	return err == nil && parsed == int64(shift)
}

// canonicalJSON caps nesting before invoking the upstream parser. Its lexical
// scan ignores braces inside strings; JCS then checks complete JSON syntax,
// duplicate keys, UTF-8 and surrogate validity.
func canonicalJSON(raw []byte, limit int) ([]byte, error) {
	if len(raw) == 0 || len(raw) > limit {
		return nil, errors.New("invalid JSON size")
	}
	depth := 0
	quoted, escaped := false, false
	for _, c := range raw {
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
			continue
		}
		if c == '{' || c == '[' {
			depth++
			if depth > 16 {
				return nil, errors.New("JSON nesting limit")
			}
		} else if c == '}' || c == ']' {
			depth--
			if depth < 0 {
				return nil, errors.New("invalid JSON nesting")
			}
		}
	}
	return jcs.Transform(raw)
}
