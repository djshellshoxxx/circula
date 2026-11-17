package imgsan

import (
	"bytes"
	"encoding/binary"
)

type imgKind int

const (
	kindUnknown imgKind = iota
	kindJPEG
	kindPNG
)

var pngSig = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func sniff(b []byte) imgKind {
	switch {
	case len(b) >= 4 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return kindJPEG
	case len(b) >= 8 && bytes.Equal(b[:8], pngSig):
		return kindPNG
	}
	return kindUnknown
}

// suspicious signatures that must never appear inside metadata payloads.
var activeSigs = [][]byte{
	[]byte("PK\x03\x04"), []byte("PK\x05\x06"), []byte("<svg"), []byte("<script"),
	[]byte("<html"), []byte("<?php"), []byte("%PDF"), []byte("<?xml"), []byte("javascript:"),
	[]byte("\x7fELF"), []byte("<!DOCTYPE"),
}

func hasActive(p []byte) bool {
	lower := bytes.ToLower(p)
	for _, s := range activeSigs {
		if bytes.Contains(lower, bytes.ToLower(s)) {
			return true
		}
	}
	return false
}

const maxSegments = 20000

// walkJPEG verifies marker structure end to end, requires EOI with only zero
// padding after it, rejects active content in APPn/COM payloads and returns
// the EXIF orientation (1 if absent or invalid).
func walkJPEG(b []byte) (int, error) {
	orient := 1
	i := 2 // after SOI
	n := len(b)
	segs := 0
	for {
		segs++
		if segs > maxSegments {
			return 0, ErrCorrupt
		}
		// find next marker prefix
		if i >= n || b[i] != 0xFF {
			return 0, ErrCorrupt
		}
		for i < n && b[i] == 0xFF {
			i++
		}
		if i >= n {
			return 0, ErrCorrupt
		}
		m := b[i]
		i++
		switch {
		case m == 0xD9: // EOI
			for _, c := range b[i:] {
				if c != 0 {
					return 0, ErrTrailingData
				}
			}
			return orient, nil
		case m == 0x00 || m == 0x01 || (m >= 0xD0 && m <= 0xD8):
			continue
		}
		if i+2 > n {
			return 0, ErrCorrupt
		}
		l := int(binary.BigEndian.Uint16(b[i:]))
		if l < 2 || i+l > n {
			return 0, ErrCorrupt
		}
		payload := b[i+2 : i+l]
		if (m >= 0xE0 && m <= 0xEF) || m == 0xFE {
			if hasActive(payload) {
				return 0, ErrSuspiciousMetadata
			}
			if m == 0xE1 {
				if o := exifOrientation(payload); o > 0 {
					orient = o
				}
			}
		}
		i += l
		if m == 0xDA { // SOS: skip entropy-coded data to next real marker
			for i < n {
				if b[i] != 0xFF {
					i++
					continue
				}
				if i+1 >= n {
					return 0, ErrCorrupt
				}
				nx := b[i+1]
				if nx == 0x00 || (nx >= 0xD0 && nx <= 0xD7) {
					i += 2
					continue
				}
				if nx == 0xFF {
					i++
					continue
				}
				break
			}
		}
	}
}

// exifOrientation parses the TIFF header of an APP1 "Exif" payload and returns
// the orientation tag (1..8) or 0. All reads are bounds-checked.
func exifOrientation(p []byte) int {
	if len(p) < 14 || string(p[:6]) != "Exif\x00\x00" {
		return 0
	}
	t := p[6:]
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	if bo.Uint16(t[2:]) != 42 {
		return 0
	}
	off := int(bo.Uint32(t[4:]))
	if off < 8 || off+2 > len(t) {
		return 0
	}
	cnt := int(bo.Uint16(t[off:]))
	for k := 0; k < cnt && k < 256; k++ {
		e := off + 2 + k*12
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:]) == 0x0112 && bo.Uint16(t[e+2:]) == 3 && bo.Uint32(t[e+4:]) == 1 {
			v := int(bo.Uint16(t[e+8:]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// walkPNG verifies the chunk sequence (IHDR first, IEND last, no trailing
// bytes) and rejects active content in textual ancillary chunks.
func walkPNG(b []byte) error {
	i := 8
	n := len(b)
	first := true
	for chunks := 0; ; chunks++ {
		if chunks > maxSegments || i+12 > n {
			return ErrCorrupt
		}
		l := int64(binary.BigEndian.Uint32(b[i:]))
		typ := string(b[i+4 : i+8])
		if l > int64(n-i-12) {
			return ErrCorrupt
		}
		if first && typ != "IHDR" {
			return ErrCorrupt
		}
		first = false
		data := b[i+8 : i+8+int(l)]
		switch typ {
		case "tEXt", "zTXt", "iTXt", "eXIf", "iCCP", "sPLT":
			if hasActive(data) {
				return ErrSuspiciousMetadata
			}
		}
		i += 12 + int(l)
		if typ == "IEND" {
			if i != n {
				return ErrTrailingData
			}
			return nil
		}
	}
}
