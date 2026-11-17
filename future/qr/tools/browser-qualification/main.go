//go:build sqlite_cgo_test

// Command browser-qualification starts the module's web handler on 127.0.0.1 with a fixture
// core and a fake-camera Y4M of a one-time request QR, for tools/browser-qualification/run.js.
// Test tooling only: never registered, never part of an application build.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
	"github.com/djshellshoxxx/circula/future/qr/web"
)

type auth struct{}

func (auth) Authenticate(r *http.Request) (web.Session, error) {
	c, err := r.Cookie("m")
	if err != nil {
		return web.Session{}, web.ErrUnauthenticated
	}
	names := map[string]string{"a": "Alice", "b": "Bob"}
	return web.Session{Principal: qr.Principal{NetworkID: "n1", MemberID: c.Value}, CookieAuth: true, CSRF: "csrf-token-0123456789", DisplayName: names[c.Value], NetworkName: "Alpha"}, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	dir := os.Args[1]
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(dir, "b.db")+"?_busy_timeout=5000&_journal_mode=WAL")
	must(err)
	ctx := context.Background()
	must(qrfixture.Migrate(ctx, db))
	must(qrfixture.AddNetwork(ctx, db, "n1", "Alpha", true))
	must(qrfixture.AddMember(ctx, db, "n1", "a", "Alice", 1000))
	must(qrfixture.AddMember(ctx, db, "n1", "b", "Bob", 1000))
	st := qr.NewStore(db)
	must(st.Migrate(ctx))
	svc, err := qr.NewService(qr.Config{Origin: "https://circula.example", NetworkEnabled: func(context.Context, qr.DBTX, string) (bool, error) { return true, nil }, AliasSecret: []byte("0123456789abcdef0123456789abcdef-browser")}, st, &qrfixture.Fixture{})
	must(err)
	h, err := web.NewHandler(web.Config{Service: svc, Auth: auth{}})
	must(err)
	// make a request for Alice and write a fake-camera y4m of its QR
	cr, err := svc.CreateRequest(ctx, qr.Principal{NetworkID: "n1", MemberID: "a"}, "browser-create-1", qr.CreateRequestInput{AmountMinor: 700, Note: "browser test"})
	must(err)
	b, err := qr.RenderPNG(cr.URL, qr.ECHigh, 230) // small enough to fit the scanner's centered scan region
	must(err)
	img, err := png.Decode(bytes.NewReader(b))
	must(err)
	must(writeY4M(filepath.Join(dir, "cam.y4m"), img))
	must(os.WriteFile(filepath.Join(dir, "info.json"), mustJSON(map[string]string{"url": cr.URL, "token": cr.Token, "id": cr.View.ID}), 0o600))
	mux := http.NewServeMux()
	mux.Handle("/", h)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "port"), []byte(fmt.Sprint(ln.Addr().(*net.TCPAddr).Port)), 0o600))
	must(http.Serve(ln, mux))
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func writeY4M(path string, src image.Image) error {
	const W, H, frames = 640, 480, 15
	canvas := image.NewGray(image.Rect(0, 0, W, H))
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	sb := src.Bounds()
	off := image.Pt((W-sb.Dx())/2, (H-sb.Dy())/2)
	draw.Draw(canvas, sb.Add(off), src, sb.Min, draw.Src)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "YUV4MPEG2 W%d H%d F10:1 Ip A1:1 C420jpeg\n", W, H)
	uv := bytes.Repeat([]byte{128}, W*H/4)
	for i := 0; i < frames; i++ {
		buf.WriteString("FRAME\n")
		buf.Write(canvas.Pix)
		buf.Write(uv)
		buf.Write(uv)
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}
