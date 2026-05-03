//go:build windows

package app

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	user32          = windows.NewLazySystemDLL("user32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procMessageBoxW   = user32.NewProc("MessageBoxW")
)

const (
	attachParentProcess = ^uintptr(0) // (DWORD)-1
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
	isTerminal = ret != 0

	if !isTerminal {
		isTerminal = isStdoutTerminal()
	}
}

func isStdoutTerminal() bool {
	var mode uint32
	handle := windows.Handle(os.Stdout.Fd())
	err := windows.GetConsoleMode(handle, &mode)
	return err == nil
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
	if isTerminal {
		return false
	}
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
