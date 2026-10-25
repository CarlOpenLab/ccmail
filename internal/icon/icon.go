// Package icon renders the ccmail app icon: a rounded-square brand gradient
// with a white envelope outline, used for the window and the tray.
package icon

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"math"
)

// Generate renders the icon at the given size (RGBA PNG).
func Generate(size int) []byte {
	if size < 8 {
		size = 256
	}
	radius := float64(size) * 0.23
	stroke := float64(size) * 0.055
	const samples = 3

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(size))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(size))
	ihdr[8] = 8
	ihdr[9] = 6

	raw := make([]byte, 0, (size*4+1)*size)
	for y := range size {
		raw = append(raw, 0)
		for x := range size {
			var cr, cg, cb, ca float64
			for sy := range samples {
				for sx := range samples {
					fx := float64(x) + (float64(sx)+0.5)/samples
					fy := float64(y) + (float64(sy)+0.5)/samples
					half := float64(size) / 2
					if !insideRounded(fx, fy, half, half, half, half, radius) {
						continue
					}
					r, g, b := gradient(fy / float64(size))
					if envelopeInk(fx, fy, float64(size), stroke) {
						r, g, b = 255, 255, 255
					}
					cr += r
					cg += g
					cb += b
					ca++
				}
			}
			if ca == 0 {
				raw = append(raw, 0, 0, 0, 0)
				continue
			}
			raw = append(raw, byte(cr/ca), byte(cg/ca), byte(cb/ca), byte(ca/(samples*samples)*255))
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

// gradient is the vertical graphite wash (#444a54 → #1e2126), matching the
// app's neutral chrome.
func gradient(t float64) (float64, float64, float64) {
	return 68 + (30-68)*t, 74 + (33-74)*t, 84 + (38-84)*t
}

// envelopeInk reports whether the point falls on the white envelope outline:
// a rounded rectangle with a downward V flap.
func envelopeInk(x, y, size, stroke float64) bool {
	cx, cy := size*0.5, size*0.51
	hw, hh := size*0.29, size*0.2
	corner := size * 0.055
	if math.Abs(roundedSDF(x, y, cx, cy, hw, hh, corner)) <= stroke/2 {
		return true
	}
	ax, ay := size*0.25, size*0.355
	bx, by := size*0.5, size*0.53
	dx, dy := size*0.75, size*0.355
	return segmentDist(x, y, ax, ay, bx, by) <= stroke/2 ||
		segmentDist(x, y, bx, by, dx, dy) <= stroke/2
}

func roundedSDF(px, py, cx, cy, hw, hh, r float64) float64 {
	qx := math.Abs(px-cx) - (hw - r)
	qy := math.Abs(py-cy) - (hh - r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func insideRounded(px, py, cx, cy, hw, hh, r float64) bool {
	return roundedSDF(px, py, cx, cy, hw, hh, r) <= 0
}

func segmentDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	length := dx*dx + dy*dy
	if length == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / length
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
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
