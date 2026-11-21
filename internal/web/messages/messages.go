// Package messages owns complete UI messages and their formatting (UX-047).
// English is the only supported language until product scope is decided.
package messages

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// Parts keeps the text surrounding one template-owned element as ordinary
// strings. The template, not the catalogue, supplies any trusted markup.
type Parts struct{ Before, After string }

const textSlot = "\x00"

// Around reserves the final string argument for exactly one template-owned
// element. Any preceding arguments must be trusted catalogue labels or decimal
// integer identifiers formatted by the application, never arbitrary text:
// the sentinel and formatting-error guard are reserved.
// Both returned strings still require contextual template escaping.
func Around(key string, labels ...string) Parts {
	args := make([]any, len(labels)+1)
	for i, label := range labels {
		args[i] = label
	}
	args[len(labels)] = textSlot
	return splitAround(Text(key, args...))
}

// AroundCount selects a counted message around one template-owned element.
// Its integer cannot introduce formatting markers or trusted HTML.
func AroundCount(key string, count int) Parts {
	return splitAround(Text(key, count, textSlot))
}

func splitAround(text string) Parts {
	if strings.Count(text, textSlot) != 1 || strings.Contains(text, "%!") {
		panic("UI message requires exactly one string slot")
	}
	before, after, _ := strings.Cut(text, textSlot)
	return Parts{Before: before, After: after}
}

// TwoParts keeps ordinary text around two template-owned elements in argument
// order. Catalogue text and values never become trusted HTML through this type.
type TwoParts struct{ Before, Between, After string }

const firstTextSlot = "\x00first\x00"
const secondTextSlot = "\x00second\x00"

// AroundTwo accepts only a message key, never user values. The template inserts
// its own elements into the two string slots; both must occur once, in order.
func AroundTwo(key string) TwoParts {
	return splitAroundTwo(Text(key, firstTextSlot, secondTextSlot))
}

// AroundTwoVersion keeps a version identifier ungrouped before two
// template-owned elements. Only the application-formatted integer enters the
// message; arbitrary text remains in the template's escaped data slots.
func AroundTwoVersion(key string, version int) TwoParts {
	return splitAroundTwo(Text(key, strconv.Itoa(version), firstTextSlot, secondTextSlot))
}

// AroundTwoCount selects a counted message before its two template-owned
// elements. Its integer cannot introduce formatting markers or trusted HTML.
func AroundTwoCount(key string, count int) TwoParts {
	return splitAroundTwo(Text(key, count, firstTextSlot, secondTextSlot))
}

func splitAroundTwo(text string) TwoParts {
	if strings.Count(text, firstTextSlot) != 1 || strings.Count(text, secondTextSlot) != 1 ||
		strings.Count(text, textSlot) != 4 || strings.Contains(text, "%!") ||
		strings.Index(text, firstTextSlot) > strings.Index(text, secondTextSlot) {
		panic("UI message requires exactly two ordered string slots")
	}
	before, rest, _ := strings.Cut(text, firstTextSlot)
	between, after, _ := strings.Cut(rest, secondTextSlot)
	return TwoParts{Before: before, Between: between, After: after}
}
