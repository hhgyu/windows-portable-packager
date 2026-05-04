//go:build windows

package app

import (
	"math/rand"
	"testing"
)

func makeRandomRGBA(w, h int) []byte {
	buf := make([]byte, w*h*4)
	rand.New(rand.NewSource(1)).Read(buf)
	return buf
}

func BenchmarkPremultiplyRGBAToBGRA(b *testing.B) {
	const w, h = 560, 400
	src := makeRandomRGBA(w, h)
	dst := make([]byte, w*h*4)
	b.SetBytes(int64(w * h * 4))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		premultiplyRGBAToBGRA(dst, src)
	}
}

// TestPremultiplyRGBAToBGRAExactness pins premultiplyRGBAToBGRA to the exact
// (c*a)/255 formula. The LUT is a performance optimisation, not a numeric
// approximation; any drift would silently shift colours under transparency.
func TestPremultiplyRGBAToBGRAExactness(t *testing.T) {
	src := makeRandomRGBA(64, 64)
	expected := make([]byte, len(src))
	for i := 0; i+3 < len(src); i += 4 {
		a := src[i+3]
		expected[i] = uint8(uint16(src[i+2]) * uint16(a) / 255)
		expected[i+1] = uint8(uint16(src[i+1]) * uint16(a) / 255)
		expected[i+2] = uint8(uint16(src[i]) * uint16(a) / 255)
		expected[i+3] = a
	}
	actual := make([]byte, len(src))
	premultiplyRGBAToBGRA(actual, src)
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("byte %d: actual=%d expected=%d", i, actual[i], expected[i])
		}
	}
}
