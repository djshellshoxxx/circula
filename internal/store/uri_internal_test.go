package store

import "testing"

// AUD4-D01: the encoding itself, on every platform (Windows cannot create the
// '?' paths that TestSQLiteOpensExactPathWithURIMetacharacters opens).
func TestFileURIEscaping(t *testing.T) {
	for in, want := range map[string]string{
		"/var/lib/circula/circula.db":   "file:/var/lib/circula/circula.db",
		`C:\data\circula.db`:            `file:C:\data\circula.db`,
		"/b/a?b/x.db":                   "file:/b/a%3Fb/x.db",
		"/b/a#b/x.db":                   "file:/b/a%23b/x.db",
		"/b/a%41b/x.db":                 "file:/b/a%2541b/x.db",
		"/b/a?&_pragma=foreign_keys(0)": "file:/b/a%3F&_pragma=foreign_keys(0)",
	} {
		if got := fileURI(in); got != want {
			t.Errorf("fileURI(%q) = %q, want %q", in, got, want)
		}
	}
}
