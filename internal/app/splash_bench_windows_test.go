//go:build windows

package app

import (
	"bytes"
	"image"
	"image/color"
	"math/rand"
	"testing"

	"github.com/kettek/apng"
)

func makeRandomRGBA(w, h int) []byte {
	buf := make([]byte, w*h*4)
	rand.New(rand.NewSource(1)).Read(buf)
	return buf
}

// makeBenchAPNG synthesises an APNG with nFrames distinct frames at wxh.
// Each frame is a solid colour shifted per index; this reproduces the cost
// shape of decoding + draw.Draw + premultiply without depending on any
// external file.
func makeBenchAPNG(tb testing.TB, w, h, nFrames int) []byte {
	tb.Helper()
	a := apng.APNG{Frames: make([]apng.Frame, nFrames)}
	for i := range a.Frames {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		c := color.RGBA{
			R: uint8((i * 7) & 0xff),
			G: uint8((i * 11) & 0xff),
			B: uint8((i * 13) & 0xff),
			A: 0xff,
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetRGBA(x, y, c)
			}
		}
		a.Frames[i].Image = img
		a.Frames[i].DelayNumerator = 1
		a.Frames[i].DelayDenominator = 30
	}
	var buf bytes.Buffer
	if err := apng.Encode(&buf, a); err != nil {
		tb.Fatalf("encode bench apng: %v", err)
	}
	return buf.Bytes()
}

func BenchmarkLoadAPNGFrames(b *testing.B) {
	data := makeBenchAPNG(b, 560, 400, 120)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		frames, err := loadAPNGFrames(data)
		if err != nil {
			b.Fatal(err)
		}
		for _, f := range frames {
			if f.hbmp != 0 {
				procDeleteObject.Call(f.hbmp)
			}
		}
	}
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
