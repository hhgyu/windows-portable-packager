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

	procCreateWindowExW2   = user32DLL.NewProc("CreateWindowExW")
	procDefWindowProcW2    = user32DLL.NewProc("DefWindowProcW")
	procDestroyWindow2     = user32DLL.NewProc("DestroyWindow")
	procDispatchMessageW2  = user32DLL.NewProc("DispatchMessageW")
	procGetDC2             = user32DLL.NewProc("GetDC")
	procGetMessageW2       = user32DLL.NewProc("GetMessageW")
	procGetSystemMetrics2  = user32DLL.NewProc("GetSystemMetrics")
	procLoadCursorW2       = user32DLL.NewProc("LoadCursorW")
	procPostQuitMessage2   = user32DLL.NewProc("PostQuitMessage")
	procRegisterClassExW2  = user32DLL.NewProc("RegisterClassExW")
	procReleaseDC2         = user32DLL.NewProc("ReleaseDC")
	procSetTimer2          = user32DLL.NewProc("SetTimer")
	procKillTimer2         = user32DLL.NewProc("KillTimer")
	procShowWindow2        = user32DLL.NewProc("ShowWindow")
	procTranslateMessage2  = user32DLL.NewProc("TranslateMessage")
	procUpdateWindow2      = user32DLL.NewProc("UpdateWindow")
	procInvalidateRect2    = user32DLL.NewProc("InvalidateRect")
	procBeginPaint2        = user32DLL.NewProc("BeginPaint")
	procEndPaint2          = user32DLL.NewProc("EndPaint")
	procGetModuleHandleW2  = kernel32DLL.NewProc("GetModuleHandleW")

	procCreateCompatibleDC     = gdi32DLL.NewProc("CreateCompatibleDC")
	procCreateDIBSection       = gdi32DLL.NewProc("CreateDIBSection")
	procSelectObject           = gdi32DLL.NewProc("SelectObject")
	procDeleteObject           = gdi32DLL.NewProc("DeleteObject")
	procDeleteDC               = gdi32DLL.NewProc("DeleteDC")
	procBitBlt                 = gdi32DLL.NewProc("BitBlt")
)

const (
	wmDestroy2  = 0x0002
	wmPaint2    = 0x000F
	wmTimer2    = 0x0113
	wsPopup2    = 0x80000000
	wsVisible2  = 0x10000000
	swShow2     = 5
	smCxScreen2 = 0
	smCyScreen2 = 1
	timerID2    = 1
	gmemMoveable = 0x0002
	srccopy      = 0x00CC0020
)

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

type paintStruct2 struct {
	hdc         uintptr
	fErase      int32
	rcPaint     struct{ left, top, right, bottom int32 }
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
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
		src := rgba.Pix
		for i := 0; i < len(dst) && i < len(src); i += 4 {
			dst[i] = src[i+2]
			dst[i+1] = src[i+1]
			dst[i+2] = src[i]
			dst[i+3] = src[i+3]
		}
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

	var frames []splashFrame
	bounds := a.Frames[0].Image.Bounds()
	for _, b := range a.Frames {
		if b.Image.Bounds().Max.X > bounds.Max.X {
			bounds = b.Image.Bounds()
		}
	}

	canvas := image.NewRGBA(bounds)

	for _, frame := range a.Frames {
		fb := frame.Image.Bounds()
		draw.Draw(canvas, fb, frame.Image, fb.Min, draw.Over)

		snapshot := image.NewRGBA(bounds)
		draw.Draw(snapshot, bounds, canvas, bounds.Min, draw.Src)

		hbmp, w, h := imageToBitmap(snapshot)

		var delay time.Duration
		if frame.DelayDenominator > 0 {
			delay = time.Duration(float64(frame.DelayNumerator)/float64(frame.DelayDenominator)*1000) * time.Millisecond
		} else {
			delay = 100 * time.Millisecond
		}
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}

		frames = append(frames, splashFrame{hbmp: hbmp, width: w, height: h, delay: delay})

		if frame.DisposeOp == apng.DISPOSE_OP_BACKGROUND {
			draw.Draw(canvas, fb, image.Transparent, image.Point{}, draw.Src)
		}
	}
	return frames, nil
}

func splashWndProc2(hwnd, msgID, wParam, lParam uintptr) uintptr {
	switch uint32(msgID) {
	case wmPaint2:
		var ps paintStruct2
		hdc, _, _ := procBeginPaint2.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 && globalSplash2 != nil {
			globalSplash2.mu.Lock()
			idx := globalSplash2.current
			frames := globalSplash2.frames
			globalSplash2.mu.Unlock()

			if idx < len(frames) {
				f := frames[idx]
				memDC, _, _ := procCreateCompatibleDC.Call(hdc)
				procSelectObject.Call(memDC, f.hbmp)
				procBitBlt.Call(hdc, 0, 0, uintptr(f.width), uintptr(f.height), memDC, 0, 0, srccopy)
				procDeleteDC.Call(memDC)
			}
		}
		procEndPaint2.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case wmTimer2:
		if globalSplash2 != nil {
			globalSplash2.mu.Lock()
			frames := globalSplash2.frames
			if len(frames) > 1 {
				globalSplash2.current = (globalSplash2.current + 1) % len(frames)
				idx := globalSplash2.current
				delay := frames[idx].delay
				globalSplash2.mu.Unlock()
				procKillTimer2.Call(hwnd, timerID2)
				procSetTimer2.Call(hwnd, timerID2, uintptr(delay.Milliseconds()), 0)
				procInvalidateRect2.Call(hwnd, 0, 0)
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
	done chan struct{}
	wg   sync.WaitGroup
}

func ShowSplashFromData(data []byte, ext string) (*SplashWindow, error) {
	frames, err := loadSplashFramesFromData(data, ext)
	if err != nil || len(frames) == 0 {
		return nil, err
	}
	return showSplashFrames(frames)
}

func showSplashFrames(frames []splashFrame) (*SplashWindow, error) {
	sw := &SplashWindow{done: make(chan struct{})}
	globalSplash2 = &splashData{frames: frames, done: sw.done}

	ready := make(chan struct{})

	sw.wg.Add(1)
	go func() {
		defer sw.wg.Done()
		defer func() {
			for _, f := range frames {
				if f.hbmp != 0 {
					procDeleteObject.Call(f.hbmp)
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
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(title)),
			wsPopup2|wsVisible2,
			uintptr(x), uintptr(y),
			uintptr(f0.width), uintptr(f0.height),
			0, 0, hInst, 0,
		)

		globalSplash2.hwnd = hwnd
		procShowWindow2.Call(hwnd, swShow2)
		procUpdateWindow2.Call(hwnd)

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

func (sw *SplashWindow) Close() {
	if sw == nil {
		return
	}
	select {
	case <-sw.done:
	default:
		close(sw.done)
	}
	sw.wg.Wait()
}
