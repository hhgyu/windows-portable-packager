//go:build windows

package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makePNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 128, B: 0, A: 255})
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestShowSplashNilOnEmptyPath(t *testing.T) {
	sw, err := ShowSplash("")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sw != nil {
		t.Error("expected nil SplashWindow for empty path")
	}
}

func TestShowSplashNilOnMissingFile(t *testing.T) {
	sw, err := ShowSplash("/nonexistent/splash.png")
	if sw != nil {
		t.Error("expected nil SplashWindow for missing file")
	}
	_ = err
}

func TestSplashCloseNilSafe(t *testing.T) {
	var sw *SplashWindow
	sw.Close()
}

func TestShowSplashFromDataNilOnEmpty(t *testing.T) {
	sw, _ := ShowSplashFromData(nil, ".png")
	if sw != nil {
		t.Error("expected nil SplashWindow for empty data")
	}
}

func TestLoadSplashFramesStaticPNG(t *testing.T) {
	tmp := t.TempDir()
	imgPath := makePNG(t, tmp, "test.png", 100, 80)

	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatal(err)
	}

	frames, err := loadSplashFramesFromData(data, ".png")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(frames))
	}
	if frames[0].width != 100 || frames[0].height != 80 {
		t.Errorf("expected 100x80, got %dx%d", frames[0].width, frames[0].height)
	}
}

func TestLoadSplashFramesJPEG(t *testing.T) {
	tmp := t.TempDir()
	imgPath := makePNG(t, tmp, "test.png", 50, 50)
	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatal(err)
	}

	frames, err := loadSplashFramesFromData(data, ".jpg")
	if err != nil {
		t.Fatalf("expected no error with PNG data as jpg ext fallback, got: %v", err)
	}
	if len(frames) == 0 {
		t.Error("expected at least 1 frame")
	}
}
