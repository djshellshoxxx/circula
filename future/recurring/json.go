package recurring

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// decodeJSON bounds nesting and rejects duplicate/unknown keys before decoding
// trusted DTOs. It is shared by HTTP and embedding CLI input paths.
func decodeJSON(raw []byte, out any) error {
	if len(raw) > 65536 || !utf8.Valid(raw) {
		return errors.New("invalid JSON bounds or encoding")
	}
	if err := validJSONUnicode(raw); err != nil {
		return err
	}
	tokens := json.NewDecoder(bytes.NewReader(raw))
	var value func(int) error
	value = func(depth int) error {
		if depth > 16 {
			return errors.New("JSON nesting too deep")
		}
		token, err := tokens.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for tokens.More() {
				key, e := tokens.Token()
				if e != nil {
					return e
				}
				s, ok := key.(string)
				if !ok {
					return errors.New("invalid JSON key")
				}
				folded := jsonNameKey(s)
				if seen[folded] {
					return errors.New("duplicate JSON key")
				}
				seen[folded] = true
				if e = value(depth + 1); e != nil {
					return e
				}
			}
			end, e := tokens.Token()
			if e != nil || end != json.Delim('}') {
				return errors.New("invalid object")
			}
		case '[':
			for tokens.More() {
				if e := value(depth + 1); e != nil {
					return e
				}
			}
			end, e := tokens.Token()
			if e != nil || end != json.Delim(']') {
				return errors.New("invalid array")
			}
		default:
			return errors.New("invalid delimiter")
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := tokens.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

// Match encoding/json's Unicode simple-fold name equivalence, including long-s
// and Kelvin-sign aliases. This comparison key does not rewrite input names or
// apply multi-rune case expansions; the decoder still handles single aliases
// and rejects unknown fields under its existing rules.
func jsonNameKey(name string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			if folded < least {
				least = folded
			}
		}
		return least
	}, name)
}

// encoding/json repairs isolated escaped surrogates. Reject them instead of
// silently changing an identifier, counterparty or private note.
func validJSONUnicode(raw []byte) error {
	inString := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || raw[i] != '\\' {
			continue
		}
		if i+1 >= len(raw) {
			return errors.New("truncated escape")
		}
		if raw[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(raw) {
			return errors.New("truncated Unicode escape")
		}
		code, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		if err != nil {
			return err
		}
		if code >= 0xd800 && code <= 0xdbff {
			if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
				return errors.New("unpaired high surrogate")
			}
			low, err := strconv.ParseUint(string(raw[i+8:i+12]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return errors.New("unpaired high surrogate")
			}
			i += 11
		} else if code >= 0xdc00 && code <= 0xdfff {
			return errors.New("unpaired low surrogate")
		} else {
			i += 5
		}
	}
	return nil
}
