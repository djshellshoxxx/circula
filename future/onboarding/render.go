package onboarding

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"sync"
)

//go:embed web/templates/*.html
var templateFS embed.FS

//go:embed web/assets/onboarding.css
var cssBytes []byte

var (
	tplOnce sync.Once
	tpl     *template.Template
	cssETag string
)

// templates parses the embedded templates lazily, so importing the package has
// no side effects. All dynamic values go through html/template's contextual
// escaping; admin copy is plain text and never marked safe.
func templates() *template.Template {
	tplOnce.Do(func() {
		sub, err := fs.Sub(templateFS, "web/templates")
		if err != nil {
			panic(err)
		}
		tpl = template.Must(template.New("onboarding").ParseFS(sub, "*.html"))
		sum := sha256.Sum256(cssBytes)
		cssETag = `"` + hex.EncodeToString(sum[:8]) + `"`
	})
	return tpl
}

// CSS returns the embedded stylesheet (for tests and for hosts that prefer to
// serve it themselves).
func CSS() []byte { return append([]byte(nil), cssBytes...) }

type fieldErr struct{ Field, Msg string }

// page is embedded in every page view model.
type page struct {
	Lang     string
	T        map[string]string
	Title    string
	CSS      string
	Practice bool
	CSRF     string
	Errors   []fieldErr
}

func newPage(locale, title, css string, practice bool) page {
	loc := pickLocale(locale)
	return page{Lang: loc, T: catalogs[loc], Title: title, CSS: css, Practice: practice}
}

func render(name string, data any) ([]byte, error) {
	var b bytes.Buffer
	if err := templates().ExecuteTemplate(&b, name, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return b.Bytes(), nil
}

type errorView struct {
	page
	Heading, Message, BackHref, BackText string
}

type stepView struct {
	Key, Name, Hint, Status, HintID, StatusID, Href string
	Muted, Next, Completed, Skipped                 bool
}

type checklistView struct {
	page
	Intro, GovernanceURL, TermsURL string
	ProgressText                   string
	Dismissed, Share               bool
	Steps                          []stepView
	ActionURL, PracticeURL         string
	AdminURL                       string
	Curriculum                     int
}

type countRow struct {
	Name  string
	Value string
}

type adminView struct {
	page
	Intro, GovernanceURL, TermsURL string
	IntroErr, GovErr, TermsErr     bool
	Revision                       int64
	Saved, MetricsOn               bool
	MetricsHelp                    string
	Counts                         []countRow
	CopyURL, MetricsURL, BackURL   string
}

func stepName(t map[string]string, s Step) string { return t["step_"+s.Key()+"_name"] }
func stepHint(t map[string]string, s Step) string { return t["step_"+s.Key()+"_hint"] }
