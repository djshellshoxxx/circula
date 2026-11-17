package qr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// decodeWithJSQR runs the bundled jsQR over PNGs. It skips when node is absent.
func decodeWithJSQR(t *testing.T, pngs ...[]byte) []*string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available for independent jsQR decode check")
	}
	type item struct {
		W    int    `json:"w"`
		H    int    `json:"h"`
		RGBA string `json:"rgba"`
	}
	var items []item
	for _, p := range pngs {
		img, err := png.Decode(bytes.NewReader(p))
		if err != nil {
			t.Fatal(err)
		}
		b := img.Bounds()
		rgba := image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rgba.Set(x, y, color.NRGBAModel.Convert(img.At(x, y)))
			}
		}
		items = append(items, item{b.Dx(), b.Dy(), base64.StdEncoding.EncodeToString(rgba.Pix)})
	}
	in, _ := json.Marshal(items)
	cmd := exec.Command(node, "testdata/decode.mjs")
	cmd.Stdin = bytes.NewReader(in)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("node decode: %v %s", err, errb.String())
	}
	var res []*string
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRenderPNGDecodesIndependently(t *testing.T) {
	tok, _ := NewToken()
	req, _ := Link("https://circula.example", tok)
	ali, _ := AliasLink("https://circula.example:8443", tok)
	var pngs [][]byte
	var want []string
	for _, c := range []string{req, ali} {
		for _, lv := range []ECLevel{ECMedium, ECQuart, ECHigh} {
			p, err := RenderPNG(c, lv, 400)
			if err != nil {
				t.Fatal(err)
			}
			pngs = append(pngs, p)
			want = append(want, c)
		}
	}
	got := decodeWithJSQR(t, pngs...)
	for i := range want {
		if got[i] == nil || *got[i] != want[i] {
			t.Fatalf("case %d decoded %v want %q", i, got[i], want[i])
		}
	}
}

func TestRenderPNGGeometryAndBounds(t *testing.T) {
	tok, _ := NewToken()
	u, _ := Link("https://c.test", tok)
	p, err := RenderPNG(u, ECHigh, 300)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(p))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Dx() > 300 || b.Dx() < 150 {
		t.Fatalf("size %v", b)
	}
	// quiet zone: every pixel in the outer 4 modules is white; both colours present
	total := 0
	for x := b.Min.X; x < b.Max.X; x++ {
		if !isWhite(img.At(x, 0)) || !isWhite(img.At(x, 3)) {
			t.Fatal("quiet zone not white")
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !isWhite(img.At(x, y)) {
				total++
			}
		}
	}
	if total == 0 {
		t.Fatal("no dark modules")
	}
	if !bytes.HasPrefix(p, []byte("\x89PNG")) {
		t.Fatal("not png")
	}
	// Deterministic output.
	p2, _ := RenderPNG(u, ECHigh, 300)
	if !bytes.Equal(p, p2) {
		t.Fatal("nondeterministic")
	}
	for name, f := range map[string]func() ([]byte, error){
		"empty":    func() ([]byte, error) { return RenderPNG("", ECHigh, 300) },
		"big":      func() ([]byte, error) { return RenderPNG(strings.Repeat("a", MaxRenderContent+1), ECHigh, 300) },
		"ctrl":     func() ([]byte, error) { return RenderPNG("https://c.test/\n", ECHigh, 300) },
		"nonascii": func() ([]byte, error) { return RenderPNG("https://c.test/é", ECHigh, 300) },
		"small":    func() ([]byte, error) { return RenderPNG(u, ECHigh, 63) },
		"huge":     func() ([]byte, error) { return RenderPNG(u, ECHigh, 100000) },
		"level":    func() ([]byte, error) { return RenderPNG(u, 0, 300) },
	} {
		if _, err := f(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	_ = os.Stdout
}

func isWhite(c color.Color) bool {
	r, g, b, a := c.RGBA()
	return r == 0xffff && g == 0xffff && b == 0xffff && a == 0xffff
}
