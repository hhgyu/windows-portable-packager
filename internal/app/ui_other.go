//go:build !windows

package app

func IsTerminal() bool {
	return true
}

func ShowRetryDialog(title, message string) bool {
	return false
}

func ShowErrorDialog(title, message string) {}

func ShowInfoDialog(title, message string) {}
