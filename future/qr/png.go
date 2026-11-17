package qr

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"

	qrcode "github.com/yeqown/go-qrcode/v2"
)

// ECLevel is the QR error-correction level.
type ECLevel int

// Error-correction levels. Links are short, so High is the print default.
const (
	ECMedium ECLevel = iota + 1
	ECQuart
	ECHigh
)

const (
	quietModules = 4 // QR specification minimum quiet zone
	// MaxRenderContent bounds the text encoded into a QR code.
	MaxRenderContent = 512
	// MaxRenderPixels bounds either PNG dimension.
	MaxRenderPixels = 2048
)

// ErrRender is returned when content or size bounds are violated.
var ErrRender = errors.New("qr: cannot render")

// RenderPNG encodes content as a black-on-white QR PNG with a four-module quiet
// zone and an integer module scale, using the pinned yeqown/go-qrcode core
// encoder. The image is at most maxPixels wide and high. Only a link built by
// Link/AliasLink should be passed; the function additionally refuses control
// characters and oversize input so it can never be coerced into encoding
// arbitrary payloads.
func RenderPNG(content string, level ECLevel, maxPixels int) ([]byte, error) {
	if content == "" || len(content) > MaxRenderContent || maxPixels < 64 || maxPixels > MaxRenderPixels {
		return nil, ErrRender
	}
	if strings.IndexFunc(content, func(r rune) bool { return r < 0x20 || r == 0x7f || r > 0x7e }) >= 0 {
		return nil, ErrRender
	}
	var ec qrcode.EncodeOption
	switch level {
	case ECMedium:
		ec = qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionMedium)
	case ECQuart:
		ec = qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart)
	case ECHigh:
		ec = qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionHighest)
	default:
		return nil, ErrRender
	}
	q, err := qrcode.NewWith(content, ec)
	if err != nil {
		return nil, ErrRender
	}
	w := &pngWriter{maxPixels: maxPixels}
	if err := q.Save(w); err != nil {
		return nil, err
	}
	if w.err != nil {
		return nil, w.err
	}
	return w.out, nil
}

// pngWriter is an original qrcode.Writer: it draws the module matrix with
// image/png only (no upstream image writer, fonts or remote resources).
type pngWriter struct {
	maxPixels int
	out       []byte
	err       error
}

func (w *pngWriter) Close() error { return nil }

func (w *pngWriter) Write(mat qrcode.Matrix) error {
	n := mat.Width()
	total := n + 2*quietModules
	scale := w.maxPixels / total
	if n <= 0 || mat.Height() != n || scale < 1 {
		w.err = ErrRender
		return w.err
	}
	px := total * scale
	pal := color.Palette{color.White, color.Black}
	img := image.NewPaletted(image.Rect(0, 0, px, px), pal) // index 0 = white
	bits := mat.Bitmap()
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !bits[y][x] {
				continue
			}
			x0, y0 := (x+quietModules)*scale, (y+quietModules)*scale
			for dy := 0; dy < scale; dy++ {
				row := img.Pix[(y0+dy)*img.Stride+x0 : (y0+dy)*img.Stride+x0+scale]
				for i := range row {
					row[i] = 1
				}
			}
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		w.err = err
		return err
	}
	w.out = buf.Bytes()
	return nil
}
