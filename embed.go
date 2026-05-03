package main

import (
	_ "embed"
	"strings"

	"github.com/hhgyu/windows-portable-packager/internal/app"
)

//go:embed embedded/app.kbpkg
var embeddedPackageData []byte

//go:embed embedded/splash.dat
var embeddedSplashData []byte

//go:embed embedded/splash.ext
var embeddedSplashExtRaw []byte

func init() {
	app.SetEmbeddedPackage(embeddedPackageData)
	ext := strings.TrimSpace(string(embeddedSplashExtRaw))
	if ext != "" && !strings.HasPrefix(string(embeddedSplashData), "PLACEHOLDER") {
		app.SetEmbeddedSplash(embeddedSplashData, ext)
	}
}
