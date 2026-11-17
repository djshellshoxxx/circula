package onboarding

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// A deliberately small HTML scanner for structural/accessibility assertions
// (the standard library has no HTML parser and this module takes no third-party
// dependency). It is a regression net, not a substitute for axe/screen-reader
// qualification, which remain open gates.

type el struct {
	Name  string
	Attrs map[string]string
	Text  string // visible+hidden text content with tags stripped
	Pos   int
}

var (
	tagRe  = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][a-zA-Z0-9]*)((?:\s+[^\s=>/]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+))?)*)\s*/?>`)
	attrRe = regexp.MustCompile(`([^\s=>/]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+)))?`)
	void   = map[string]bool{"meta": true, "link": true, "input": true, "br": true, "hr": true, "img": true}
)

func parseHTML(t *testing.T, doc string) []el {
	t.Helper()
	var out []el
	type open struct{ idx, textStart int }
	var stack []open
	for _, m := range tagRe.FindAllStringSubmatchIndex(doc, -1) {
		closing := doc[m[2]:m[3]] == "/"
		name := strings.ToLower(doc[m[4]:m[5]])
		if closing {
			for i := len(stack) - 1; i >= 0; i-- {
				if out[stack[i].idx].Name == name {
					raw := doc[stack[i].textStart:m[0]]
					out[stack[i].idx].Text = html.UnescapeString(strings.Join(strings.Fields(tagRe.ReplaceAllString(raw, " ")), " "))
					stack = stack[:i]
					break
				}
			}
			continue
		}
		attrs := map[string]string{}
		for _, a := range attrRe.FindAllStringSubmatch(doc[m[6]:m[7]], -1) {
			v := a[2] + a[3] + a[4]
			attrs[strings.ToLower(a[1])] = html.UnescapeString(v)
		}
		out = append(out, el{Name: name, Attrs: attrs, Pos: m[0]})
		if !void[name] {
			stack = append(stack, open{len(out) - 1, m[1]})
		}
	}
	return out
}

func byName(els []el, name string) []el {
	var r []el
	for _, e := range els {
		if e.Name == name {
			r = append(r, e)
		}
	}
	return r
}

// checkA11y applies structural WCAG-minded assertions to a rendered page.
func checkA11y(t *testing.T, doc string) []el {
	t.Helper()
	els := parseHTML(t, doc)
	if !strings.HasPrefix(doc, "<!doctype html>") {
		t.Error("missing doctype")
	}
	htmls := byName(els, "html")
	if len(htmls) != 1 || htmls[0].Attrs["lang"] == "" {
		t.Error("html lang missing")
	}
	if len(byName(els, "main")) != 1 {
		t.Error("want exactly one <main>")
	}
	if len(byName(els, "title")) != 1 || byName(els, "title")[0].Text == "" {
		t.Error("title missing")
	}
	// meta viewport must not disable zoom
	for _, m := range byName(els, "meta") {
		if m.Attrs["name"] == "viewport" && (strings.Contains(m.Attrs["content"], "user-scalable") || strings.Contains(m.Attrs["content"], "maximum-scale")) {
			t.Error("viewport blocks zoom")
		}
	}
	h1 := byName(els, "h1")
	if len(h1) != 1 {
		t.Errorf("want one h1, got %d", len(h1))
	}
	// heading levels never skip downward
	prev := 0
	for _, e := range els {
		if len(e.Name) == 2 && e.Name[0] == 'h' && e.Name[1] >= '1' && e.Name[1] <= '6' {
			lvl := int(e.Name[1] - '0')
			if e.Text == "" {
				t.Errorf("empty heading %s", e.Name)
			}
			if prev != 0 && lvl > prev+1 {
				t.Errorf("heading level jumps h%d -> h%d", prev, lvl)
			}
			prev = lvl
		}
	}
	ids := map[string]bool{}
	for _, e := range els {
		if id := e.Attrs["id"]; id != "" {
			if ids[id] {
				t.Errorf("duplicate id %q", id)
			}
			ids[id] = true
		}
	}
	for _, e := range els {
		for _, ref := range []string{"aria-describedby", "aria-labelledby", "aria-controls"} {
			for _, id := range strings.Fields(e.Attrs[ref]) {
				if !ids[id] {
					t.Errorf("<%s> %s points at missing id %q", e.Name, ref, id)
				}
			}
		}
		if e.Name == "label" && e.Attrs["for"] != "" && !ids[e.Attrs["for"]] {
			t.Errorf("label for=%q missing", e.Attrs["for"])
		}
		if e.Name == "a" {
			if href := e.Attrs["href"]; strings.HasPrefix(href, "#") && href != "#" && !ids[href[1:]] {
				t.Errorf("fragment link %q has no target", href)
			}
			if e.Text == "" && e.Attrs["aria-label"] == "" {
				t.Errorf("link without name: %v", e.Attrs)
			}
		}
		if e.Name == "button" && e.Text == "" && e.Attrs["aria-label"] == "" {
			t.Error("button without accessible name")
		}
		// No script, inline style, event handlers, external loads or images without alt.
		if e.Name == "script" || e.Name == "iframe" || e.Name == "object" || e.Name == "embed" {
			t.Errorf("forbidden element <%s>", e.Name)
		}
		for k, v := range e.Attrs {
			if strings.HasPrefix(k, "on") || k == "style" {
				t.Errorf("forbidden attribute %s on <%s>", k, e.Name)
			}
			if (k == "src" || k == "action" || (k == "href" && e.Name == "link")) && (strings.Contains(v, "://") || strings.HasPrefix(v, "//")) {
				t.Errorf("external load %s=%q", k, v)
			}
		}
		if e.Name == "img" && e.Attrs["alt"] == "" {
			t.Error("img without alt")
		}
		if e.Name == "meta" && strings.EqualFold(e.Attrs["http-equiv"], "refresh") {
			t.Error("meta refresh")
		}
		if e.Name == "form" && strings.ToLower(e.Attrs["method"]) == "post" && e.Attrs["action"] == "" {
			t.Error("post form without explicit action")
		}
	}
	// Every visible form control has a label via for/id.
	labelFor := map[string]bool{}
	for _, l := range byName(els, "label") {
		labelFor[l.Attrs["for"]] = true
	}
	for _, e := range els {
		if e.Name == "textarea" || e.Name == "select" || (e.Name == "input" && e.Attrs["type"] != "hidden" && e.Attrs["type"] != "submit") {
			if !labelFor[e.Attrs["id"]] && e.Attrs["aria-label"] == "" {
				t.Errorf("control %q has no label", e.Attrs["name"])
			}
		}
	}
	// Radio groups are in a fieldset with a legend.
	if len(byName(els, "input")) > 0 {
		radios := 0
		for _, e := range byName(els, "input") {
			if e.Attrs["type"] == "radio" {
				radios++
			}
		}
		if radios > 0 && (len(byName(els, "fieldset")) == 0 || len(byName(els, "legend")) == 0) {
			t.Error("radios without fieldset/legend")
		}
	}
	// Tables have header cells with scope.
	for _, e := range byName(els, "th") {
		if e.Attrs["scope"] == "" {
			t.Errorf("th %q without scope", e.Text)
		}
	}
	return els
}
