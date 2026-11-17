package qr

import (
	"strings"
	"testing"
	"time"
)

func TestLinks(t *testing.T) {
	a, e := NewToken()
	if e != nil {
		t.Fatal(e)
	}
	b, _ := NewToken()
	if a == b {
		t.Fatal("duplicate entropy")
	}
	u, e := Link("https://c.test", a)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Parse("https://c.test", u)
	if e != nil || got != a {
		t.Fatal(got, e)
	}
	for _, bad := range []string{u + "?amount=1", u + "#x", u + "#", "https://evil.test/qr/v1/" + a, "https://c.test/qr/v1/short"} {
		if _, e := Parse("https://c.test", bad); e == nil {
			t.Fatal(bad)
		}
	}
}
func TestRequestExpiryAndScope(t *testing.T) {
	now := time.Unix(100, 0)
	r := Request{Network: "n", Recipient: "b", Amount: 1, Expires: now.Add(time.Minute)}
	if r.Validate("n", "a", now) != nil {
		t.Fatal("valid")
	}
	if r.Validate("x", "a", now) == nil || r.Validate("n", "b", now) == nil || r.Validate("n", "a", r.Expires) == nil {
		t.Fatal("bad boundary")
	}
	r.ClaimedTransaction = "tx"
	if r.Validate("n", "a", now) == nil {
		t.Fatal("claimed")
	}
}

func FuzzParse(f *testing.F) {
	f.Add("https://c.test/qr/v1/AAAAAAAAAAAAAAAAAAAAAA")
	f.Add("javascript:alert(1)")
	f.Fuzz(func(t *testing.T, raw string) {
		token, err := Parse("https://c.test", raw)
		if err != nil {
			return
		}
		roundtrip, err := Link("https://c.test", token)
		if err != nil || roundtrip != raw {
			t.Fatalf("accepted noncanonical URL: %q", raw)
		}
	})
}

func TestAliasLinksAndManualCodes(t *testing.T) {
	tok, _ := NewToken()
	al, err := AliasLink("https://c.test:8443", tok)
	if err != nil || al != "https://c.test:8443/qr/v1/a/"+tok {
		t.Fatal(al, err)
	}
	p, err := ParseLink("https://c.test:8443", al)
	if err != nil || p.Kind != KindAlias || p.Code != tok {
		t.Fatal(p, err)
	}
	if _, err := Parse("https://c.test:8443", al); err == nil {
		t.Fatal("Parse must reject alias URLs")
	}
	if _, err := ParseLink("https://c.test", al); err == nil {
		t.Fatal("port mismatch accepted")
	}
	req, _ := Link("https://c.test", tok)
	for in, want := range map[string]Parsed{
		req:                                      {KindRequest, tok},
		"  " + req + "\n":                        {KindRequest, tok},
		"r." + tok:                               {KindRequest, tok},
		"a." + tok:                               {KindAlias, tok},
		"\t" + CompactCode(KindAlias, tok) + " ": {KindAlias, tok},
	} {
		got, err := ParseManual("https://c.test", in)
		if err != nil || got != want {
			t.Fatalf("%q: %v %v", in, got, err)
		}
	}
	for _, bad := range []string{"", " ", "x." + tok, "r." + tok[:21], "r." + tok + "A", "r.", tok, "r. " + tok, "R." + tok, "r." + tok[:21] + "!", req + " " + req, "https://C.test/qr/v1/" + tok} {
		if _, err := ParseManual("https://c.test", bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	// noncanonical base64 (same bytes, extra trailing bits) is not accepted
	if _, err := TokenHash("AAAAAAAAAAAAAAAAAAAAAB"); err == nil {
		t.Fatal("noncanonical token accepted")
	}
	// origin validation
	for _, o := range []string{"http://c.test", "https://c.test/", "https://c.test/x", "https://u@c.test", "https://c.test?x=1", "https://c.test#x", "", "c.test", "https://"} {
		if ValidateOrigin(o) == nil {
			t.Fatalf("origin %q accepted", o)
		}
	}
	if ValidateOrigin("https://c.test:8443") != nil {
		t.Fatal("port origin rejected")
	}
	if _, err := Link("http://c.test", tok); err == nil {
		t.Fatal("link on non-https origin")
	}
	if _, err := Link("https://c.test", "bad"); err == nil {
		t.Fatal("link with bad token")
	}
}

func TestParseRejectsTamperingMatrix(t *testing.T) {
	tok, _ := NewToken()
	const o = "https://c.test"
	good := o + "/qr/v1/" + tok
	for _, bad := range []string{
		"https://c.test@evil.test/qr/v1/" + tok, "https://c.test.evil.test/qr/v1/" + tok, "https://evil.test/qr/v1/" + tok, "http://c.test/qr/v1/" + tok,
		"//c.test/qr/v1/" + tok, "c.test/qr/v1/" + tok, "HTTPS://c.test/qr/v1/" + tok + "x", good + "?amount=1", good + "?", good + "#", good + "/", good + "%00", good + "%2f",
		o + "/qr/v1/%2e%2e/" + tok, o + "/QR/v1/" + tok, o + "/qr/v1//" + tok, o + "/qr/v1/a/a/" + tok, o + "/qr/v0/" + tok, o + "/qr/" + tok,
		"javascript:alert(1)", "data:text/plain,x", "file:///etc/passwd", "https://c.test\\@evil.test/qr/v1/" + tok, "https://c.test:443/qr/v1/" + tok,
		good + "\x00", " " + good, good + " ", "https://c.test/qr/v1/" + tok[:10] + "\n" + tok[10:], strings.Repeat("a", MaxLinkLen+1),
	} {
		if p, err := ParseLink(o, bad); err == nil {
			t.Errorf("accepted %q as %v", bad, p)
		}
	}
	if p, err := ParseLink(o, good); err != nil || p.Code != tok {
		t.Fatal(err)
	}
}

func TestRedactPathAndLimits(t *testing.T) {
	tok, _ := NewToken()
	if got := RedactPath("/qr/v1/" + tok); strings.Contains(got, tok) {
		t.Fatal(got)
	}
	if RedactPath("/healthz") != "/healthz" {
		t.Fatal("unrelated path changed")
	}
}

func FuzzParseLinkAndManual(f *testing.F) {
	tok := "AAAAAAAAAAAAAAAAAAAAAA"
	f.Add("https://c.test/qr/v1/a/" + tok)
	f.Add("r." + tok)
	f.Add("https://c.test/qr/v1/" + tok + "?x")
	f.Add("%2e%2e")
	f.Fuzz(func(t *testing.T, raw string) {
		if p, err := ParseLink("https://c.test", raw); err == nil {
			var again string
			if p.Kind == KindAlias {
				again, _ = AliasLink("https://c.test", p.Code)
			} else {
				again, _ = Link("https://c.test", p.Code)
			}
			if again != raw {
				t.Fatalf("accepted noncanonical link %q", raw)
			}
		}
		if p, err := ParseManual("https://c.test", raw); err == nil {
			if _, err := TokenHash(p.Code); err != nil || (p.Kind != KindAlias && p.Kind != KindRequest) {
				t.Fatalf("manual parse returned unusable %q", raw)
			}
		}
		_ = RedactPath(raw)
	})
}
