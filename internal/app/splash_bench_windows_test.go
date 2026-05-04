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

func BenchmarkRGBAToBGRA(b *testing.B) {
	const w, h = 560, 400
	src := makeRandomRGBA(w, h)
	dst := make([]byte, w*h*4)
	b.SetBytes(int64(w * h * 4))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rgbaToBGRA(dst, src)
	}
}

// BenchmarkLoadFirstFrameFast measures the latency of the async fast path
// taken by ShowSplashFromData before background loading kicks in. This is
// what the user actually waits to see on screen.
func BenchmarkLoadFirstFrameFast(b *testing.B) {
	data := makeBenchAPNG(b, 560, 400, 120)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f, ok := loadFirstFrameFast(data, ".apng")
		if !ok {
			b.Fatal("first frame decode failed")
		}
		if f.hbmp != 0 {
			procDeleteObject.Call(f.hbmp)
		}
	}
}

// TestRGBAToBGRASwap pins rgbaToBGRA to a pure R<->B swap with no alpha math.
// image.RGBA is already premultiplied, so any per-pixel multiply here would
// double-premultiply and crush translucent pixels toward black.
func TestRGBAToBGRASwap(t *testing.T) {
	src := makeRandomRGBA(64, 64)
	expected := make([]byte, len(src))
	for i := 0; i+3 < len(src); i += 4 {
		expected[i] = src[i+2]
		expected[i+1] = src[i+1]
		expected[i+2] = src[i]
		expected[i+3] = src[i+3]
	}
	actual := make([]byte, len(src))
	rgbaToBGRA(actual, src)
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("byte %d: actual=%d expected=%d", i, actual[i], expected[i])
		}
	}
}
