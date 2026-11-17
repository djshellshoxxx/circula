// Package testkit holds synthetic fixtures shared by this module's tests.
package testkit

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"sync"
	"testing"
	"time"
)

// Clock is a settable clock safe for concurrent use.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

// NewClock starts at a fixed instant.
func NewClock() *Clock { return &Clock{t: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)} }

// Now returns the current fake time.
func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

// Advance moves time forward.
func (c *Clock) Advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// JPEG returns a synthetic gradient JPEG. seed varies the pixels so that
// different seeds hash differently.
func JPEG(tb testing.TB, w, h, seed int) []byte {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8((x*7 + seed*31) % 256), uint8((y*5 + seed*17) % 256), uint8(seed * 13 % 256), 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

// PNG returns a synthetic PNG.
func PNG(tb testing.TB, w, h int) []byte {
	tb.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{10, uint8(x), uint8(y), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

// WithGPSEXIF inserts an APP1 Exif segment with a GPS marker string and
// the given orientation right after SOI.
func WithGPSEXIF(jpg []byte, orientation uint16) []byte {
	var t bytes.Buffer
	t.WriteString("MM")
	binary.Write(&t, binary.BigEndian, uint16(42))
	binary.Write(&t, binary.BigEndian, uint32(8))
	binary.Write(&t, binary.BigEndian, uint16(1))
	binary.Write(&t, binary.BigEndian, uint16(0x0112))
	binary.Write(&t, binary.BigEndian, uint16(3))
	binary.Write(&t, binary.BigEndian, uint32(1))
	binary.Write(&t, binary.BigEndian, orientation)
	binary.Write(&t, binary.BigEndian, uint16(0))
	binary.Write(&t, binary.BigEndian, uint32(0))
	t.WriteString("GPSLatitude=49.2827N;GPSLongitude=123.1207W;HOME")
	payload := append([]byte("Exif\x00\x00"), t.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}
