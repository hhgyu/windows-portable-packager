//go:build !windows

package app

import "os"

func init() {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}
	SetLocale(lang)
}
