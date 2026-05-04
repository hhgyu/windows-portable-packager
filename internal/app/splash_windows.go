//go:build windows

package app

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/kettek/apng"
	"golang.org/x/sys/windows"
)

var (
	gdiplusDLL = windows.NewLazySystemDLL("gdiplus.dll")
	user32DLL  = windows.NewLazySystemDLL("user32.dll")
	gdi32DLL   = windows.NewLazySystemDLL("gdi32.dll")
	kernel32DLL = windows.NewLazySystemDLL("kernel32.dll")

	procGdiplusStartup2        = gdiplusDLL.NewProc("GdiplusStartup")
	procGdiplusShutdown2       = gdiplusDLL.NewProc("GdiplusShutdown")
	procGdipCreateBitmapFromStream = gdiplusDLL.NewProc("GdipCreateBitmapFromStream")
	procGdipDisposeImage2      = gdiplusDLL.NewProc("GdipDisposeImage")
	procGdipGetImageWidth2     = gdiplusDLL.NewProc("GdipGetImageWidth")
	procGdipGetImageHeight2    = gdiplusDLL.NewProc("GdipGetImageHeight")
	procGdipCreateFromHDC2     = gdiplusDLL.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics2    = gdiplusDLL.NewProc("GdipDeleteGraphics")
	procGdipDrawImageRectI     = gdiplusDLL.NewProc("GdipDrawImageRectI")

	procCreateStreamOnHGlobal = windows.NewLazySystemDLL("ole32.dll").NewProc("CreateStreamOnHGlobal")
	procGlobalAlloc           = kernel32DLL.NewProc("GlobalAlloc")
	procGlobalLock            = kernel32DLL.NewProc("GlobalLock")
	procGlobalUnlock          = kernel32DLL.NewProc("GlobalUnlock")

	procCreateWindowExW2    = user32DLL.NewProc("CreateWindowExW")
	procDefWindowProcW2     = user32DLL.NewProc("DefWindowProcW")
	procDestroyWindow2      = user32DLL.NewProc("DestroyWindow")
	procDispatchMessageW2   = user32DLL.NewProc("DispatchMessageW")
	procGetDC2              = user32DLL.NewProc("GetDC")
	procGetMessageW2        = user32DLL.NewProc("GetMessageW")
	procGetSystemMetrics2   = user32DLL.NewProc("GetSystemMetrics")
	procLoadCursorW2        = user32DLL.NewProc("LoadCursorW")
	procPostQuitMessage2    = user32DLL.NewProc("PostQuitMessage")
	procRegisterClassExW2   = user32DLL.NewProc("RegisterClassExW")
	procReleaseDC2          = user32DLL.NewProc("ReleaseDC")
	procSetTimer2           = user32DLL.NewProc("SetTimer")
	procKillTimer2          = user32DLL.NewProc("KillTimer")
	procShowWindow2         = user32DLL.NewProc("ShowWindow")
	procTranslateMessage2   = user32DLL.NewProc("TranslateMessage")
	procUpdateLayeredWindow = user32DLL.NewProc("UpdateLayeredWindow")
	procGetModuleHandleW2   = kernel32DLL.NewProc("GetModuleHandleW")

	procCreateCompatibleDC = gdi32DLL.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32DLL.NewProc("CreateDIBSection")
	procSelectObject       = gdi32DLL.NewProc("SelectObject")
	procDeleteObject       = gdi32DLL.NewProc("DeleteObject")
	procDeleteDC           = gdi32DLL.NewProc("DeleteDC")
)

const (
	wmDestroy2   = 0x0002
	wmTimer2     = 0x0113
	wsPopup2     = 0x80000000
	wsExLayered  = 0x00080000
	swShow2      = 5
	smCxScreen2  = 0
	smCyScreen2  = 1
	timerID2     = 1
	gmemMoveable = 0x0002
	ulwAlpha     = 0x00000002
	acSrcOver    = 0x00
	acSrcAlpha   = 0x01
)

// blendFunction matches Win32 BLENDFUNCTION used by UpdateLayeredWindow.
// Layout is fixed by the API and must not be reordered.
type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type pointL struct {
	X int32
	Y int32
}

type sizeL struct {
	CX int32
	CY int32
}

type wndClassExW2 struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msg2 struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type bitmapInfo struct {
	bmiHeader bitmapInfoHeader
	bmiColors [1]uint32
}

type gdiplusStartupInput2 struct {
	gdiplusVersion           uint32
	debugEventCallback       uintptr
	suppressBackgroundThread int32
	suppressExternalCodecs   int32
}

type splashFrame struct {
	hbmp   uintptr
	width  int
	height int
	delay  time.Duration
}

type splashData struct {
	frames  []splashFrame
	current int
	mu      sync.Mutex
	hwnd    uintptr
	done    chan struct{}
}

var (
	gdipToken2  uintptr
	globalSplash2 *splashData
)

func gdipInit2() bool {
	input := gdiplusStartupInput2{gdiplusVersion: 1}
	ret, _, _ := procGdiplusStartup2.Call(
		uintptr(unsafe.Pointer(&gdipToken2)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	return ret == 0
}

// rgbaToBGRA swaps the R and B channels of a Go image.RGBA buffer (which is
// already alpha-premultiplied per the image package contract) so it matches
// the premultiplied BGRA layout UpdateLayeredWindow with AC_SRC_ALPHA expects.
// No per-pixel multiply: doing one here would double-premultiply alpha and
// crush translucent pixels toward black.
func rgbaToBGRA(dst, src []byte) {
	n := len(dst)
	if len(src) < n {
		n = len(src)
	}
	for i := 0; i+3 < n; i += 4 {
		dst[i] = src[i+2]
		dst[i+1] = src[i+1]
		dst[i+2] = src[i]
		dst[i+3] = src[i+3]
	}
}

func imageToBitmap(img image.Image) (uintptr, int, int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	bi := bitmapInfo{
		bmiHeader: bitmapInfoHeader{
			biSize:      uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			biWidth:     int32(w),
			biHeight:    -int32(h),
			biPlanes:    1,
			biBitCount:  32,
		},
	}

	screenDC, _, _ := procGetDC2.Call(0)
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)

	var bits unsafe.Pointer
	hbmp, _, _ := procCreateDIBSection.Call(
		memDC,
		uintptr(unsafe.Pointer(&bi)),
		0,
		uintptr(unsafe.Pointer(&bits)),
		0, 0,
	)

	if bits != nil && hbmp != 0 {
		dst := unsafe.Slice((*byte)(bits), w*h*4)
		rgbaToBGRA(dst, rgba.Pix)
	}

	procDeleteDC.Call(memDC)
	procReleaseDC2.Call(0, screenDC)
	return hbmp, w, h
}

func loadSplashFrames(imagePath string) ([]splashFrame, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(imagePath))
	return loadSplashFramesFromData(data, ext)
}

func loadSplashFramesFromData(data []byte, ext string) ([]splashFrame, error) {
	if ext == ".gif" {
		return loadGIFFrames(data)
	}

	if ext == ".png" || ext == ".apng" {
		if frames, err := loadAPNGFrames(data); err == nil && len(frames) > 1 {
			return frames, nil
		}
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	hbmp, w, h := imageToBitmap(img)
	return []splashFrame{{hbmp: hbmp, width: w, height: h, delay: 0}}, nil
}

func loadGIFFrames(data []byte) ([]splashFrame, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var frames []splashFrame
	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := image.NewRGBA(bounds)

	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		snapshot := image.NewRGBA(bounds)
		draw.Draw(snapshot, bounds, canvas, bounds.Min, draw.Src)

		hbmp, w, h := imageToBitmap(snapshot)
		delay := time.Duration(g.Delay[i]) * 10 * time.Millisecond
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		frames = append(frames, splashFrame{hbmp: hbmp, width: w, height: h, delay: delay})

		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalBackground {
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}
	}
	return frames, nil
}

func loadAPNGFrames(data []byte) ([]splashFrame, error) {
	a, err := apng.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if len(a.Frames) <= 1 {
		return nil, nil
	}

	// APNG canvas is the union of every frame's destination rect
	// (XOffset+Width, YOffset+Height). Frames are partial images placed
	// onto the canvas at their offsets, not standalone full-size images.
	canvasW, canvasH := 0, 0
	for _, f := range a.Frames {
		fb := f.Image.Bounds()
		if right := f.XOffset + fb.Dx(); right > canvasW {
			canvasW = right
		}
		if bottom := f.YOffset + fb.Dy(); bottom > canvasH {
			canvasH = bottom
		}
	}
	if canvasW == 0 || canvasH == 0 {
		return nil, nil
	}

	canvasBounds := image.Rect(0, 0, canvasW, canvasH)
	canvas := image.NewRGBA(canvasBounds)
	var prevSnapshot *image.RGBA

	type pendingFrame struct {
		snapshot *image.RGBA
		delay    time.Duration
	}
	var pending []pendingFrame

	for i, frame := range a.Frames {
		// IsDefault is the static fallback PNG and per spec is not part of
		// the animation loop, so skip it when present on frame 0.
		if i == 0 && frame.IsDefault {
			continue
		}

		fb := frame.Image.Bounds()
		target := image.Rect(
			frame.XOffset, frame.YOffset,
			frame.XOffset+fb.Dx(), frame.YOffset+fb.Dy(),
		)

		// DISPOSE_OP_PREVIOUS requires the pre-render canvas to be restored
		// before the next frame draws; snapshot it while it is still intact.
		if frame.DisposeOp == apng.DISPOSE_OP_PREVIOUS {
			prevSnapshot = image.NewRGBA(canvasBounds)
			draw.Draw(prevSnapshot, canvasBounds, canvas, image.Point{}, draw.Src)
		}

		// BLEND_OP_SOURCE replaces destination pixels (alpha included);
		// BLEND_OP_OVER alpha-blends onto the existing canvas.
		blendOp := draw.Over
		if frame.BlendOp == apng.BLEND_OP_SOURCE {
			blendOp = draw.Src
		}
		draw.Draw(canvas, target, frame.Image, fb.Min, blendOp)

		snapshot := image.NewRGBA(canvasBounds)
		draw.Draw(snapshot, canvasBounds, canvas, image.Point{}, draw.Src)

		var delay time.Duration
		if frame.DelayDenominator > 0 {
			delay = time.Duration(float64(frame.DelayNumerator)/float64(frame.DelayDenominator)*1000) * time.Millisecond
		} else {
			delay = 100 * time.Millisecond
		}
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}

		pending = append(pending, pendingFrame{snapshot: snapshot, delay: delay})

		// DisposeOp determines the canvas state seen by the next frame.
		switch frame.DisposeOp {
		case apng.DISPOSE_OP_BACKGROUND:
			draw.Draw(canvas, target, image.Transparent, image.Point{}, draw.Src)
		case apng.DISPOSE_OP_PREVIOUS:
			if prevSnapshot != nil {
				draw.Draw(canvas, canvasBounds, prevSnapshot, image.Point{}, draw.Src)
			}
		}
	}

	// imageToBitmap performs CPU premultiply + GDI CreateDIBSection per frame.
	// GDI object creation is thread-safe so long as each goroutine works on
	// its own DC, which the function already does — fan out across NumCPU
	// workers for ~Ncore speedup on this stage.
	frames := make([]splashFrame, len(pending))
	workers := runtime.NumCPU()
	if workers > len(pending) {
		workers = len(pending)
	}
	var wg sync.WaitGroup
	jobs := make(chan int, len(pending))
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				p := pending[idx]
				hbmp, fw, fh := imageToBitmap(p.snapshot)
				frames[idx] = splashFrame{hbmp: hbmp, width: fw, height: fh, delay: p.delay}
			}
		}()
	}
	for i := range pending {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return frames, nil
}

// updateLayeredFrame pushes a single splashFrame to the layered window via
// UpdateLayeredWindow. The bitmap MUST already hold premultiplied BGRA data
// because we pass AC_SRC_ALPHA in the BLENDFUNCTION.
func updateLayeredFrame(hwnd uintptr, frame splashFrame) {
	if frame.hbmp == 0 {
		return
	}
	screenDC, _, _ := procGetDC2.Call(0)
	if screenDC == 0 {
		return
	}
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		procReleaseDC2.Call(0, screenDC)
		return
	}
	oldBmp, _, _ := procSelectObject.Call(memDC, frame.hbmp)

	sz := sizeL{CX: int32(frame.width), CY: int32(frame.height)}
	srcPt := pointL{X: 0, Y: 0}
	blend := blendFunction{
		BlendOp:             acSrcOver,
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         acSrcAlpha,
	}

	procUpdateLayeredWindow.Call(
		hwnd,
		screenDC,
		0,
		uintptr(unsafe.Pointer(&sz)),
		memDC,
		uintptr(unsafe.Pointer(&srcPt)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)

	procSelectObject.Call(memDC, oldBmp)
	procDeleteDC.Call(memDC)
	procReleaseDC2.Call(0, screenDC)
}

func splashWndProc2(hwnd, msgID, wParam, lParam uintptr) uintptr {
	switch uint32(msgID) {
	case wmTimer2:
		if globalSplash2 != nil {
			globalSplash2.mu.Lock()
			frames := globalSplash2.frames
			if len(frames) > 1 {
				globalSplash2.current = (globalSplash2.current + 1) % len(frames)
				idx := globalSplash2.current
				next := frames[idx]
				delay := next.delay
				globalSplash2.mu.Unlock()
				procKillTimer2.Call(hwnd, timerID2)
				procSetTimer2.Call(hwnd, timerID2, uintptr(delay.Milliseconds()), 0)
				updateLayeredFrame(hwnd, next)
			} else {
				globalSplash2.mu.Unlock()
			}
		}
		return 0

	case wmDestroy2:
		procKillTimer2.Call(hwnd, timerID2)
		procPostQuitMessage2.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW2.Call(hwnd, msgID, wParam, lParam)
	return ret
}

type SplashWindow struct {
	done       chan struct{}
	wg         sync.WaitGroup
	startedAt  time.Time
	minVisible time.Duration
}

// SetMinVisible asks Close to keep the splash on screen for at least d after
// it was first shown. ForceClose still bypasses this. Safe to call once after
// ShowSplashFromData / ShowSplash; calling more than once or after Close has
// no effect.
func (sw *SplashWindow) SetMinVisible(d time.Duration) {
	if sw == nil || d <= 0 {
		return
	}
	sw.minVisible = d
}

func ShowSplashFromData(data []byte, ext string) (*SplashWindow, error) {
	if isAnimatedSplashExt(ext) {
		if first, ok := loadFirstFrameFast(data, ext); ok {
			sw, err := showSplashFrames([]splashFrame{first})
			if err != nil {
				if first.hbmp != 0 {
					procDeleteObject.Call(first.hbmp)
				}
				return sw, err
			}
			go loadRemainingFramesAsync(data, ext)
			return sw, nil
		}
	}

	frames, err := loadSplashFramesFromData(data, ext)
	if err != nil || len(frames) == 0 {
		return nil, err
	}
	return showSplashFrames(frames)
}

func isAnimatedSplashExt(ext string) bool {
	e := strings.ToLower(ext)
	return e == ".gif" || e == ".apng" || e == ".png"
}

// loadFirstFrameFast composes only the FIRST animation frame onto the full
// canvas, so the splash window can appear before the remaining frames finish
// decoding. APNG default frame (IHDR+IDAT) is NOT used because it can be a
// fallback static image at a different position than the animation's frame[0]
// (e.g. progress-bar overlays whose XOffset/YOffset differ from the IHDR).
func loadFirstFrameFast(data []byte, ext string) (splashFrame, bool) {
	switch strings.ToLower(ext) {
	case ".gif":
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil || len(g.Image) == 0 {
			return splashFrame{}, false
		}
		bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
		canvas := image.NewRGBA(bounds)
		fb := g.Image[0].Bounds()
		draw.Draw(canvas, fb, g.Image[0], fb.Min, draw.Over)
		hbmp, w, h := imageToBitmap(canvas)
		delay := time.Duration(g.Delay[0]) * 10 * time.Millisecond
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		return splashFrame{hbmp: hbmp, width: w, height: h, delay: delay}, true

	case ".png", ".apng":
		a, err := apng.DecodeAll(bytes.NewReader(data))
		if err != nil || len(a.Frames) == 0 {
			return splashFrame{}, false
		}
		canvasW, canvasH := 0, 0
		for _, f := range a.Frames {
			fb := f.Image.Bounds()
			if right := f.XOffset + fb.Dx(); right > canvasW {
				canvasW = right
			}
			if bottom := f.YOffset + fb.Dy(); bottom > canvasH {
				canvasH = bottom
			}
		}
		if canvasW == 0 || canvasH == 0 {
			return splashFrame{}, false
		}
		canvasBounds := image.Rect(0, 0, canvasW, canvasH)
		canvas := image.NewRGBA(canvasBounds)
		var first *apng.Frame
		for i := range a.Frames {
			if i == 0 && a.Frames[i].IsDefault {
				continue
			}
			first = &a.Frames[i]
			break
		}
		if first == nil {
			return splashFrame{}, false
		}
		fb := first.Image.Bounds()
		target := image.Rect(first.XOffset, first.YOffset, first.XOffset+fb.Dx(), first.YOffset+fb.Dy())
		blendOp := draw.Over
		if first.BlendOp == apng.BLEND_OP_SOURCE {
			blendOp = draw.Src
		}
		draw.Draw(canvas, target, first.Image, fb.Min, blendOp)
		hbmp, w, h := imageToBitmap(canvas)
		var delay time.Duration
		if first.DelayDenominator > 0 {
			delay = time.Duration(float64(first.DelayNumerator)/float64(first.DelayDenominator)*1000) * time.Millisecond
		} else {
			delay = 100 * time.Millisecond
		}
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		return splashFrame{hbmp: hbmp, width: w, height: h, delay: delay}, true
	}
	return splashFrame{}, false
}

// loadRemainingFramesAsync runs on a background goroutine after the splash is
// already on screen with its first frame. It performs the full APNG/GIF decode
// + per-frame premultiply, then atomically swaps globalSplash2.frames so the
// timer can advance through the real animation. Old frames (the placeholder
// first frame) are released after the swap.
func loadRemainingFramesAsync(data []byte, ext string) {
	frames, err := loadSplashFramesFromData(data, ext)
	if err != nil || len(frames) == 0 {
		return
	}

	g := globalSplash2
	if g == nil {
		for _, f := range frames {
			if f.hbmp != 0 {
				procDeleteObject.Call(f.hbmp)
			}
		}
		return
	}

	g.mu.Lock()
	old := g.frames
	g.frames = frames
	g.current = 0
	hwnd := g.hwnd
	g.mu.Unlock()

	for _, f := range old {
		if f.hbmp != 0 {
			procDeleteObject.Call(f.hbmp)
		}
	}

	if hwnd != 0 && len(frames) > 1 {
		updateLayeredFrame(hwnd, frames[0])
		delay := frames[0].delay
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		procSetTimer2.Call(hwnd, timerID2, uintptr(delay.Milliseconds()), 0)
	}
}

func showSplashFrames(frames []splashFrame) (*SplashWindow, error) {
	sw := &SplashWindow{done: make(chan struct{}), startedAt: time.Now()}
	globalSplash2 = &splashData{frames: frames, done: sw.done}

	ready := make(chan struct{})

	sw.wg.Add(1)
	go func() {
		defer sw.wg.Done()
		defer func() {
			// Read the current frames (may have been swapped by
			// loadRemainingFramesAsync after the splash was created)
			// and free those, not the originally captured slice.
			if g := globalSplash2; g != nil {
				g.mu.Lock()
				cur := g.frames
				g.frames = nil
				g.mu.Unlock()
				for _, f := range cur {
					if f.hbmp != 0 {
						procDeleteObject.Call(f.hbmp)
					}
				}
			}
			globalSplash2 = nil
		}()

		hInst, _, _ := procGetModuleHandleW2.Call(0)
		cursor, _, _ := procLoadCursorW2.Call(0, 32512)

		className, _ := windows.UTF16PtrFromString("WPPSplash2")
		wc := wndClassExW2{
			cbSize:        uint32(unsafe.Sizeof(wndClassExW2{})),
			lpfnWndProc:   windows.NewCallback(splashWndProc2),
			hInstance:     hInst,
			hCursor:       cursor,
			lpszClassName: className,
		}
		procRegisterClassExW2.Call(uintptr(unsafe.Pointer(&wc)))

		f0 := frames[0]
		screenW, _, _ := procGetSystemMetrics2.Call(smCxScreen2)
		screenH, _, _ := procGetSystemMetrics2.Call(smCyScreen2)
		x := (int32(screenW) - int32(f0.width)) / 2
		y := (int32(screenH) - int32(f0.height)) / 2

		title, _ := windows.UTF16PtrFromString("")
		hwnd, _, _ := procCreateWindowExW2.Call(
			wsExLayered,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(title)),
			wsPopup2,
			uintptr(x), uintptr(y),
			uintptr(f0.width), uintptr(f0.height),
			0, 0, hInst, 0,
		)

		globalSplash2.hwnd = hwnd
		updateLayeredFrame(hwnd, f0)
		procShowWindow2.Call(hwnd, swShow2)

		if len(frames) > 1 {
			procSetTimer2.Call(hwnd, timerID2, uintptr(frames[0].delay.Milliseconds()), 0)
		}

		close(ready)

		var m msg2
		for {
			select {
			case <-sw.done:
				procDestroyWindow2.Call(hwnd)
			default:
			}
			ret, _, _ := procGetMessageW2.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if ret == 0 || int32(ret) == -1 {
				return
			}
			procTranslateMessage2.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW2.Call(uintptr(unsafe.Pointer(&m)))
		}
	}()

	<-ready
	time.Sleep(50 * time.Millisecond)
	return sw, nil
}

func ShowSplash(imagePath string) (*SplashWindow, error) {
	if imagePath == "" {
		return nil, nil
	}
	frames, err := loadSplashFrames(imagePath)
	if err != nil || len(frames) == 0 {
		return nil, err
	}
	return showSplashFrames(frames)
}

// Close honours SetMinVisible: if the splash has been on screen for less than
// minVisible, it sleeps the remainder before tearing the window down. Use
// this on the success path after the child has spawned. Use ForceClose on
// error paths so a failing launcher does not strand the user staring at a
// splash that will not advance.
func (sw *SplashWindow) Close() {
	if sw == nil {
		return
	}
	if sw.minVisible > 0 && !sw.startedAt.IsZero() {
		if remaining := sw.minVisible - time.Since(sw.startedAt); remaining > 0 {
			time.Sleep(remaining)
		}
	}
	sw.tearDown()
}

// ForceClose tears the splash window down immediately, bypassing minVisible.
// Reserved for error/abort paths.
func (sw *SplashWindow) ForceClose() {
	if sw == nil {
		return
	}
	sw.tearDown()
}

func (sw *SplashWindow) tearDown() {
	select {
	case <-sw.done:
	default:
		close(sw.done)
	}
	sw.wg.Wait()
}
