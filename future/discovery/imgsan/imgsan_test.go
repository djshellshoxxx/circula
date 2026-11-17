package imgsan

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"
)

func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 90, 255})
		}
	}
	return img
}

func jpegBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, gradient(w, h), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{200, uint8(x), uint8(y), 128})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func segment(marker byte, payload []byte) []byte {
	out := []byte{0xFF, marker, 0, 0}
	binary.BigEndian.PutUint16(out[2:], uint16(len(payload)+2))
	return append(out, payload...)
}

// exifWithGPS builds an Exif APP1 payload with orientation and a GPS IFD.
func exifWithGPS(orientation uint16) []byte {
	var t bytes.Buffer
	t.WriteString("MM")
	binary.Write(&t, binary.BigEndian, uint16(42))
	binary.Write(&t, binary.BigEndian, uint32(8))
	binary.Write(&t, binary.BigEndian, uint16(2)) // two entries
	// orientation
	binary.Write(&t, binary.BigEndian, uint16(0x0112))
	binary.Write(&t, binary.BigEndian, uint16(3))
	binary.Write(&t, binary.BigEndian, uint32(1))
	binary.Write(&t, binary.BigEndian, orientation)
	binary.Write(&t, binary.BigEndian, uint16(0))
	// GPS IFD pointer
	binary.Write(&t, binary.BigEndian, uint16(0x8825))
	binary.Write(&t, binary.BigEndian, uint16(4))
	binary.Write(&t, binary.BigEndian, uint32(1))
	binary.Write(&t, binary.BigEndian, uint32(8+2+24+4))
	binary.Write(&t, binary.BigEndian, uint32(0)) // next IFD
	// GPS IFD: latitude ref "N", marker string
	binary.Write(&t, binary.BigEndian, uint16(1))
	binary.Write(&t, binary.BigEndian, uint16(1))
	binary.Write(&t, binary.BigEndian, uint16(2))
	binary.Write(&t, binary.BigEndian, uint32(2))
	t.Write([]byte{'N', 0, 0, 0})
	binary.Write(&t, binary.BigEndian, uint32(0))
	t.WriteString("GPSLAT49.2827N123.1207W")
	return append([]byte("Exif\x00\x00"), t.Bytes()...)
}

func insertAfterSOI(jpg []byte, segs ...[]byte) []byte {
	out := append([]byte{}, jpg[:2]...)
	for _, s := range segs {
		out = append(out, s...)
	}
	return append(out, jpg[2:]...)
}

func pngChunk(typ string, data []byte) []byte {
	out := make([]byte, 8, 12+len(data))
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], typ)
	out = append(out, data...)
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	crc.Write(data)
	return binary.BigEndian.AppendUint32(out, crc.Sum32())
}

func insertPNGChunk(p []byte, chunk []byte) []byte { // after IHDR (8 sig + 25)
	out := append([]byte{}, p[:33]...)
	out = append(out, chunk...)
	return append(out, p[33:]...)
}

func sanitize(t testing.TB, s *Sanitizer, b []byte) (*Result, error) {
	t.Helper()
	return s.Sanitize(context.Background(), bytes.NewReader(b))
}

func TestJPEGRoundTripAndThumbnail(t *testing.T) {
	s := New(Limits{FullMaxEdge: 400, ThumbMaxEdge: 100})
	r, err := sanitize(t, s, jpegBytes(t, 800, 400))
	if err != nil {
		t.Fatal(err)
	}
	if r.Full.MIME != "image/jpeg" || r.Full.Width != 400 || r.Full.Height != 200 {
		t.Fatalf("full %+v", r.Full)
	}
	if r.Thumb.Width != 100 || r.Thumb.Height != 50 {
		t.Fatalf("thumb %+v", r.Thumb)
	}
	if _, err := jpeg.Decode(bytes.NewReader(r.Full.Data)); err != nil {
		t.Fatal(err)
	}
	if len(r.SHA256) != 64 {
		t.Fatal("sha")
	}
	// never upscale
	r, err = sanitize(t, s, jpegBytes(t, 40, 30))
	if err != nil || r.Full.Width != 40 || r.Thumb.Width != 40 {
		t.Fatal(r, err)
	}
}

func TestEXIFGPSStrippedAndOrientationApplied(t *testing.T) {
	src := insertAfterSOI(jpegBytes(t, 60, 20), segment(0xE1, exifWithGPS(6)), segment(0xFE, []byte("camera: LatLong 49.28,-123.12 private")))
	if !bytes.Contains(src, []byte("GPSLAT")) {
		t.Fatal("fixture lacks gps")
	}
	r, err := sanitize(t, New(Limits{}), src)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"GPSLAT", "Exif", "LatLong", "123.12", "camera"} {
		if bytes.Contains(r.Full.Data, []byte(needle)) || bytes.Contains(r.Thumb.Data, []byte(needle)) {
			t.Fatalf("metadata %q survived", needle)
		}
	}
	// orientation 6 rotates a 60x20 landscape to 20x60.
	if r.Full.Width != 20 || r.Full.Height != 60 {
		t.Fatalf("orientation not applied: %dx%d", r.Full.Width, r.Full.Height)
	}
	if exifOrientation(exifWithGPS(6)) != 6 || exifOrientation([]byte("Exif\x00\x00II")) != 0 {
		t.Fatal("exif parser")
	}
}

func TestAllOrientationsKeepPixelCount(t *testing.T) {
	src := gradient(7, 3)
	for o := 2; o <= 8; o++ {
		d := orientImage(src, o)
		w, h := d.Bounds().Dx(), d.Bounds().Dy()
		if w*h != 21 || (o >= 5) != (w == 3) {
			t.Fatalf("o=%d %dx%d", o, w, h)
		}
	}
}

func TestPNGKeepsAlphaAndStripsText(t *testing.T) {
	src := insertPNGChunk(pngBytes(t, 50, 30), pngChunk("tEXt", []byte("Comment\x00GPS 49.28 secret-home")))
	r, err := sanitize(t, New(Limits{}), src)
	if err != nil {
		t.Fatal(err)
	}
	if r.Full.MIME != "image/png" || bytes.Contains(r.Full.Data, []byte("secret-home")) || bytes.Contains(r.Full.Data, []byte("tEXt")) {
		t.Fatal("png metadata or mime")
	}
	img, err := png.Decode(bytes.NewReader(r.Full.Data))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := img.At(10, 10).RGBA(); a == 0xffff {
		t.Fatal("alpha lost")
	}
}

func TestRejections(t *testing.T) {
	s := New(Limits{MaxInputBytes: 20000, MaxWidth: 500, MaxHeight: 500, MaxPixels: 100000})
	var gifb bytes.Buffer
	gif.Encode(&gifb, gradient(5, 5), nil)
	jp := jpegBytes(t, 20, 20)
	big := func() []byte { // valid structure, declared 60000x60000
		ihdr := make([]byte, 13)
		binary.BigEndian.PutUint32(ihdr, 60000)
		binary.BigEndian.PutUint32(ihdr[4:], 60000)
		ihdr[8], ihdr[9] = 8, 2
		out := append([]byte{}, pngSig...)
		out = append(out, pngChunk("IHDR", ihdr)...)
		return append(out, pngChunk("IEND", nil)...)
	}()
	cases := map[string]struct {
		in  []byte
		err error
	}{
		"empty":        {nil, ErrEmpty},
		"svg":          {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ErrUnsupportedType},
		"html":         {[]byte("<html><script>x</script></html>"), ErrUnsupportedType},
		"gif":          {gifb.Bytes(), ErrUnsupportedType},
		"zip":          {append([]byte("PK\x03\x04"), make([]byte, 100)...), ErrUnsupportedType},
		"pdf":          {[]byte("%PDF-1.7 ..."), ErrUnsupportedType},
		"oversize":     {append(append([]byte{}, jp...), make([]byte, 30000)...), ErrTooLarge},
		"jpeg trailer": {append(append([]byte{}, jp...), []byte("PK\x03\x04zipdata")...), ErrTrailingData},
		"jpeg+svg":     {append(append([]byte{}, jp...), []byte(`<svg onload=alert(1)>`)...), ErrTrailingData},
		"jpeg comment": {insertAfterSOI(jp, segment(0xFE, []byte("hi <script>alert(1)</script>"))), ErrSuspiciousMetadata},
		"jpeg zip app": {insertAfterSOI(jp, segment(0xE5, []byte("xxPK\x03\x04yy"))), ErrSuspiciousMetadata},
		"png trailer":  {append(pngBytes(t, 10, 10), []byte("<svg>")...), ErrTrailingData},
		"png text":     {insertPNGChunk(pngBytes(t, 10, 10), pngChunk("tEXt", []byte("a\x00<script>"))), ErrSuspiciousMetadata},
		"png bomb":     {big, ErrDimensions},
		"png no IEND":  {pngBytes(t, 10, 10)[:60], ErrCorrupt},
		"truncated":    {jp[:len(jp)/2], ErrCorrupt},
		"garbage jpeg": {[]byte{0xFF, 0xD8, 0xFF, 0x00, 0x01, 0x02}, ErrCorrupt},
		"too wide":     {jpegBytes(t, 600, 10), ErrDimensions},
		"too many px":  {jpegBytes(t, 400, 400), ErrDimensions},
	}
	for name, c := range cases {
		if _, err := sanitize(t, s, c.in); err != c.err {
			t.Errorf("%s: got %v want %v", name, err, c.err)
		}
	}
}

func TestJPEGDimensionBombDeclaredHuge(t *testing.T) {
	// Patch SOF0 height/width of a tiny JPEG to 65535x65535; must be rejected
	// by DecodeConfig limits before allocating.
	jp := jpegBytes(t, 16, 16)
	i := bytes.Index(jp, []byte{0xFF, 0xC0})
	if i < 0 {
		t.Fatal("no sof0")
	}
	binary.BigEndian.PutUint16(jp[i+5:], 65535)
	binary.BigEndian.PutUint16(jp[i+7:], 65535)
	if _, err := sanitize(t, New(Limits{}), jp); err != ErrDimensions {
		t.Fatal(err)
	}
}

func TestOutputLimit(t *testing.T) {
	s := New(Limits{MaxOutputBytes: 200})
	if _, err := sanitize(t, s, jpegBytes(t, 200, 200)); err != ErrOutputTooLarge {
		t.Fatal(err)
	}
}

func TestOriginalBytesNeverReturned(t *testing.T) {
	marker := "ORIGINAL-TRAILER-MARKER"
	src := insertAfterSOI(jpegBytes(t, 30, 30), segment(0xE2, []byte("profile "+marker)))
	r, err := sanitize(t, New(Limits{}), src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(r.Full.Data), marker) || strings.Contains(string(r.Thumb.Data), marker) {
		t.Fatal("original bytes leaked")
	}
}

func TestConcurrencyBoundAndContext(t *testing.T) {
	s := New(Limits{MaxConcurrent: 1})
	s.sem <- struct{}{} // occupy the only slot
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := s.Sanitize(ctx, bytes.NewReader(jpegBytes(t, 20, 20))); err != ErrBusy {
		t.Fatal(err)
	}
	<-s.sem
	var wg sync.WaitGroup
	src := jpegBytes(t, 120, 90)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Sanitize(context.Background(), bytes.NewReader(src)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestPalettedPNG(t *testing.T) {
	// paletted PNG
	pal := image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.Black, color.White})
	var b bytes.Buffer
	png.Encode(&b, pal)
	if _, err := sanitize(t, New(Limits{}), b.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func FuzzSanitize(f *testing.F) {
	f.Add(jpegBytes(f, 8, 8))
	f.Add(pngBytes(f, 8, 8))
	f.Add(insertAfterSOI(jpegBytes(f, 8, 8), segment(0xE1, exifWithGPS(3))))
	f.Add([]byte("<svg/>"))
	s := New(Limits{MaxInputBytes: 1 << 16, MaxWidth: 256, MaxHeight: 256, MaxPixels: 65536, MaxConcurrent: 1})
	f.Fuzz(func(t *testing.T, in []byte) {
		r, err := s.Sanitize(context.Background(), bytes.NewReader(in))
		if err != nil {
			return
		}
		if bytes.Contains(r.Full.Data, []byte("Exif")) {
			t.Fatal("exif in output")
		}
		if r.Full.Width > 256 || r.Full.Height > 256 {
			t.Fatal("size")
		}
	})
}
