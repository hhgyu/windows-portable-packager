//go:build windows

package app

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	user32            = windows.NewLazySystemDLL("user32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procMessageBoxW   = user32.NewProc("MessageBoxW")
)

const (
	attachParentProcess = ^uintptr(0)
	mbOk                = 0x00000000
	mbOkCancel          = 0x00000001
	mbRetryCancel       = 0x00000005
	mbIconWarning       = 0x00000030
	mbIconError         = 0x00000010
	mbIconInfo          = 0x00000040
	idOk                = 1
	idCancel            = 2
	idRetry             = 4
)

var isTerminal bool

func init() {
	ret, _, _ := procAttachConsole.Call(attachParentProcess)
	if ret == 0 {
		return
	}
	isTerminal = true
	reattachStdHandles()
}

func reattachStdHandles() {
	if h, err := openConsoleFile("CONOUT$", windows.GENERIC_WRITE); err == nil {
		os.Stdout = os.NewFile(uintptr(h), "stdout")
	}
	if h, err := openConsoleFile("CONOUT$", windows.GENERIC_WRITE); err == nil {
		os.Stderr = os.NewFile(uintptr(h), "stderr")
	}
	if h, err := openConsoleFile("CONIN$", windows.GENERIC_READ); err == nil {
		os.Stdin = os.NewFile(uintptr(h), "stdin")
	}
}

func openConsoleFile(name string, access uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(
		p,
		access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
}

func IsTerminal() bool {
	return isTerminal
}

func messageBox(title, message string, flags uintptr) int {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	msgPtr, _ := windows.UTF16PtrFromString(message)
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	return int(ret)
}

func ShowRetryDialog(title, message string) bool {
	ret := messageBox(title, message, mbRetryCancel|mbIconWarning)
	return ret == idRetry
}

func ShowErrorDialog(title, message string) {
	if isTerminal {
		return
	}
	messageBox(title, message, mbOk|mbIconError)
}

func ShowInfoDialog(title, message string) {
	if isTerminal {
		return
	}
	messageBox(title, message, mbOk|mbIconInfo)
}
