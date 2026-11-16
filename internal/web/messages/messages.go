// Package messages owns complete UI messages and their formatting (UX-047).
// English is the only supported language until product scope is decided.
package messages

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

//go:embed en.json
var english []byte

type entry struct {
	One   string `json:"one,omitempty"`
	Other string `json:"other"`
}

var builder, keys = load()

func load() (*catalog.Builder, map[string]entry) {
	var entries map[string]entry
	if err := json.Unmarshal(english, &entries); err != nil {
		panic(fmt.Errorf("UI message catalogue: %w", err))
	}
	b := catalog.NewBuilder()
	for key, e := range entries {
		if key == "" || e.Other == "" {
			panic("UI message catalogue: empty key or fallback")
		}
		var err error
		if e.One == "" {
			err = b.SetString(language.English, key, e.Other)
		} else {
			err = b.Set(language.English, key, plural.Selectf(1, "%d", "one", e.One, "other", e.Other))
		}
		if err != nil {
			panic(fmt.Errorf("UI message %s: %w", key, err))
		}
	}
	return b, entries
}

// Text returns plain text, never trusted markup. Callers keep html/template's
// escaping boundary. Plural entries select on the first argument (an integer);
// positional placeholders let a catalogue reorder all sentence arguments.
func Text(key string, args ...any) string {
	if _, ok := keys[key]; !ok {
		panic("unknown UI message: " + key)
	}
	return message.NewPrinter(language.English, message.Catalog(builder)).Sprintf(key, args...)
}
