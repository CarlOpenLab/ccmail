// Package icon generates the app icon (rounded-square gradient with a
// white four-point sparkle) as PNG bytes, for the window and the tray.
package icon

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
)

type point struct{ x, y float64 }

// starPolygon mirrors the CSS brand glyph's clip-path star.
var starPolygon = []point{
	{0.50, 0.00}, {0.62, 0.38}, {1.00, 0.50}, {0.62, 0.62},
	{0.50, 1.00}, {0.38, 0.62}, {0.00, 0.50}, {0.38, 0.38},
}

func pointInPolygon(pt point, poly []point) bool {
	inside := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		if (poly[i].y > pt.y) != (poly[j].y > pt.y) &&
			pt.x < (poly[j].x-poly[i].x)*(pt.y-poly[i].y)/(poly[j].y-poly[i].y)+poly[i].x {
			inside = !inside
		}
		j = i
	}
	return inside
}

func chunk(typ string, data []byte) []byte {
	out := make([]byte, 0, len(data)+12)
	var lenb [4]byte
	binary.BigEndian.PutUint32(lenb[:], uint32(len(data)))
	out = append(out, lenb[:]...)
	out = append(out, typ...)
	out = append(out, data...)
	var crcb [4]byte
	binary.BigEndian.PutUint32(crcb[:], crc32.ChecksumIEEE(append([]byte(typ), data...)))
	return append(out, crcb[:]...)
}

// Generate renders the icon at the given size (RGBA PNG).
func Generate(size int) []byte {
	if size < 8 {
		size = 256
	}
	radius := float64(size) * 0.22
	starSize := float64(size) * 0.52
	starOff := (float64(size) - starSize) / 2

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(size))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(size))
	ihdr[8] = 8
	ihdr[9] = 6

	raw := make([]byte, 0, (size*4+1)*size)
	for y := 0; y < size; y++ {
		raw = append(raw, 0)
		for x := 0; x < size; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			// rounded-rect coverage via distance to corner centers
			alpha := 255.0
			cx := radius
			if fx > float64(size)-radius {
				cx = float64(size) - radius
			}
			cy := radius
			if fy > float64(size)-radius {
				cy = float64(size) - radius
			}
			dx, dy := fx-cx, fy-cy
			if dx > 0 && dy > 0 {
				d := dx*dx + dy*dy
				if d > radius*radius {
					alpha = 0
				}
			}
			// vertical brand gradient #4f7df8 -> #2f5bd8
			t := fy / float64(size)
			r := 79 + (47-79)*t
			g := 125 + (91-125)*t
			b := 248 + (216-248)*t
			// white sparkle star
			if pointInPolygon(point{(fx - starOff) / starSize, (fy - starOff) / starSize}, starPolygon) {
				r, g, b = 255, 255, 255
			}
			raw = append(raw, byte(r), byte(g), byte(b), byte(alpha))
		}
	}

	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	out.Write(chunk("IHDR", ihdr))
	out.Write(chunk("IDAT", zbuf.Bytes()))
	out.Write(chunk("IEND", nil))
	return out.Bytes()
}
