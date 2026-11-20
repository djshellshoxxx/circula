package web

import "testing"

// PER-022: a server-rendered freshness statement survives disabled scripting
// on both balance and history views, with an exact machine-readable time.
func TestFinancialPagesStateSnapshotTime(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	for _, path := range []string{w.base(), w.base() + "/history", w.base() + "/trades/new"} {
		p := b.get(path)
		if p.Status != 200 {
			t.Fatalf("%s status=%d", path, p.Status)
		}
		b.mustContain(p, `id="figure-freshness"`, `datetime="2026-03-01T12:00:00Z"`,
			"Figures are a snapshot and do not update automatically.", `id="figure-stale" class="notice" role="status" hidden`)
	}
	b.mustNotContain(b.get("/login"), `id="figure-freshness"`)
}
