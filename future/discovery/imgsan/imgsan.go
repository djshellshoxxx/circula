// Package imgsan validates untrusted JPEG/PNG uploads and produces fresh,
// metadata-free re-encoded images plus a thumbnail.
//
// The original container bytes are never copied to the output. Everything is
// bounded before the full decode: input bytes, dimensions, pixel count,
// concurrency, and output size. Only JPEG and PNG are accepted; SVG, GIF,
// ZIP, WebP, TIFF, BMP and everything else is rejected by signature, not by
// the client-declared content type or file name.
package imgsan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
)

// Errors returned by Sanitize. They carry no uploaded content.
var (
	ErrEmpty              = errors.New("imgsan: empty upload")
	ErrTooLarge           = errors.New("imgsan: upload exceeds byte limit")
	ErrUnsupportedType    = errors.New("imgsan: only JPEG and PNG are accepted")
	ErrCorrupt            = errors.New("imgsan: image is malformed")
	ErrDimensions         = errors.New("imgsan: image dimensions exceed limit")
	ErrTrailingData       = errors.New("imgsan: data after end of image")
	ErrSuspiciousMetadata = errors.New("imgsan: metadata contains active or archive content")
	ErrOutputTooLarge     = errors.New("imgsan: sanitized image exceeds output limit")
	ErrBusy               = errors.New("imgsan: decoder busy")
)

// Limits are explicit resource budgets. Zero values are replaced by the
// provisional defaults; these defaults have not been derived from a
// production resource budget.
type Limits struct {
	MaxInputBytes  int   // raw upload bytes
	MaxWidth       int   // declared pixel width
	MaxHeight      int   // declared pixel height
	MaxPixels      int64 // width*height, computed without overflow
	FullMaxEdge    int   // longest edge of the stored full-size image
	ThumbMaxEdge   int   // longest edge of the thumbnail
	MaxOutputBytes int   // each of full and thumbnail after re-encode
	MaxConcurrent  int   // simultaneous decodes in this Sanitizer
	JPEGQuality    int   // 1..100
}

// DefaultLimits returns provisional defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxInputBytes:  5 << 20,
		MaxWidth:       8000,
		MaxHeight:      8000,
		MaxPixels:      16_000_000,
		FullMaxEdge:    2048,
		ThumbMaxEdge:   320,
		MaxOutputBytes: 3 << 20,
		MaxConcurrent:  2,
		JPEGQuality:    85,
	}
}

func (l Limits) withDefaults() Limits {
	d := DefaultLimits()
	if l.MaxInputBytes <= 0 {
		l.MaxInputBytes = d.MaxInputBytes
	}
	if l.MaxWidth <= 0 {
		l.MaxWidth = d.MaxWidth
	}
	if l.MaxHeight <= 0 {
		l.MaxHeight = d.MaxHeight
	}
	if l.MaxPixels <= 0 {
		l.MaxPixels = d.MaxPixels
	}
	if l.FullMaxEdge <= 0 {
		l.FullMaxEdge = d.FullMaxEdge
	}
	if l.ThumbMaxEdge <= 0 {
		l.ThumbMaxEdge = d.ThumbMaxEdge
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = d.MaxOutputBytes
	}
	if l.MaxConcurrent <= 0 {
		l.MaxConcurrent = d.MaxConcurrent
	}
	if l.JPEGQuality <= 0 || l.JPEGQuality > 100 {
		l.JPEGQuality = d.JPEGQuality
	}
	return l
}

// Variant is one re-encoded image.
type Variant struct {
	Data          []byte
	MIME          string
	Width, Height int
}

// Result is the sanitized output. SHA256 is the hex digest of Full.Data.
type Result struct {
	Full   Variant
	Thumb  Variant
	SHA256 string
}

// Sanitizer bounds concurrent decodes.
type Sanitizer struct {
	lim Limits
	sem chan struct{}
}

// New builds a Sanitizer. Zero Limits fields take defaults.
func New(l Limits) *Sanitizer {
	l = l.withDefaults()
	return &Sanitizer{lim: l, sem: make(chan struct{}, l.MaxConcurrent)}
}

// Limits returns the effective limits.
func (s *Sanitizer) Limits() Limits { return s.lim }

// Sanitize reads at most MaxInputBytes+1 bytes from r, validates structure
// and declared size, decodes, applies EXIF orientation (JPEG), and
// re-encodes a full image and a thumbnail. Waiting for a decode slot honors ctx.
func (s *Sanitizer) Sanitize(ctx context.Context, r io.Reader) (*Result, error) {
	raw, err := io.ReadAll(io.LimitReader(r, int64(s.lim.MaxInputBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrEmpty
	}
	if len(raw) > s.lim.MaxInputBytes {
		return nil, ErrTooLarge
	}
	kind := sniff(raw)
	if kind == kindUnknown {
		return nil, ErrUnsupportedType
	}
	// Structural strictness and declared dimensions are checked before any
	// pixel buffer exists and without taking a decode slot.
	var orient int
	var cfg image.Config
	switch kind {
	case kindJPEG:
		o, err := walkJPEG(raw)
		if err != nil {
			return nil, err
		}
		orient = o
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return nil, ErrCorrupt
		}
	case kindPNG:
		if err := walkPNG(raw); err != nil {
			return nil, err
		}
		cfg, err = png.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return nil, ErrCorrupt
		}
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrCorrupt
	}
	if cfg.Width > s.lim.MaxWidth || cfg.Height > s.lim.MaxHeight || int64(cfg.Width)*int64(cfg.Height) > s.lim.MaxPixels {
		return nil, ErrDimensions
	}

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return nil, ErrBusy
	}

	var src image.Image
	if kind == kindJPEG {
		src, err = jpeg.Decode(bytes.NewReader(raw))
	} else {
		src, err = png.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return nil, ErrCorrupt
	}
	base := toRGBA(src)
	src = nil
	if kind == kindJPEG && orient > 1 {
		base = orientImage(base, orient)
	}
	mime := "image/jpeg"
	if kind == kindPNG {
		mime = "image/png"
	}
	full, err := s.encode(fit(base, s.lim.FullMaxEdge), mime)
	if err != nil {
		return nil, err
	}
	thumb, err := s.encode(fit(base, s.lim.ThumbMaxEdge), mime)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(full.Data)
	return &Result{Full: full, Thumb: thumb, SHA256: hex.EncodeToString(sum[:])}, nil
}

func (s *Sanitizer) encode(img *image.RGBA, mime string) (Variant, error) {
	b := img.Bounds()
	v := Variant{MIME: mime, Width: b.Dx(), Height: b.Dy()}
	var buf bytes.Buffer
	if mime == "image/png" {
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return v, ErrCorrupt
		}
		if buf.Len() > s.lim.MaxOutputBytes {
			return v, ErrOutputTooLarge
		}
		v.Data = buf.Bytes()
		return v, nil
	}
	// JPEG output only comes from JPEG input, which is opaque; PNG keeps alpha.
	for q := s.lim.JPEGQuality; ; q -= 15 {
		buf.Reset()
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return v, ErrCorrupt
		}
		if buf.Len() <= s.lim.MaxOutputBytes {
			v.Data = buf.Bytes()
			return v, nil
		}
		if q-15 < 30 {
			return v, ErrOutputTooLarge
		}
	}
}

// toRGBA copies any decoded image (YCbCr, CMYK, paletted, 16-bit ...) into a
// fresh zero-origin RGBA so that no source memory or metadata survives.
func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	xdraw.Draw(dst, dst.Bounds(), src, b.Min, xdraw.Src)
	return dst
}

// fit scales down so the longest edge is at most max. Never upscales.
func fit(src *image.RGBA, max int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	var nw, nh int
	if w >= h {
		nw = max
		nh = int((int64(h)*int64(max) + int64(w)/2) / int64(w))
	} else {
		nh = max
		nw = int((int64(w)*int64(max) + int64(h)/2) / int64(h))
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst
}

// orientImage applies EXIF orientation 2..8 to a zero-origin RGBA.
func orientImage(src *image.RGBA, o int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	nw, nh := w, h
	if o >= 5 {
		nw, nh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			default:
				dx, dy = x, y
			}
			dst.SetRGBA(dx, dy, src.RGBAAt(x, y))
		}
	}
	return dst
}
