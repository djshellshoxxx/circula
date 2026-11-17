package cli

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
)

// syntheticGeotaggedJPEG builds a 60x20 JPEG with EXIF orientation 6 and a
// GPS-looking marker, entirely in memory.
func syntheticGeotaggedJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 60, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 60; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 4), uint8(y * 12), 80, 255})
		}
	}
	var j bytes.Buffer
	jpeg.Encode(&j, img, nil)
	var t bytes.Buffer
	t.WriteString("MM")
	binary.Write(&t, binary.BigEndian, uint16(42))
	binary.Write(&t, binary.BigEndian, uint32(8))
	binary.Write(&t, binary.BigEndian, uint16(1))
	binary.Write(&t, binary.BigEndian, uint16(0x0112))
	binary.Write(&t, binary.BigEndian, uint16(3))
	binary.Write(&t, binary.BigEndian, uint32(1))
	binary.Write(&t, binary.BigEndian, uint16(6))
	binary.Write(&t, binary.BigEndian, uint16(0))
	binary.Write(&t, binary.BigEndian, uint32(0))
	t.WriteString("GPSLatitude=49.28N;HOME")
	payload := append([]byte("Exif\x00\x00"), t.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, j.Bytes()[:2]...)
	out = append(out, seg...)
	return append(out, j.Bytes()[2:]...)
}
