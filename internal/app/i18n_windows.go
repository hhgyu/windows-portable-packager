//go:build windows

package app

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func init() {
	lang := detectSystemLocale()
	SetLocale(lang)
}

func detectSystemLocale() string {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultLocaleName")

	buf := make([]uint16, 85)
	ret, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return "en"
	}
	return windows.UTF16ToString(buf)
}
