package main

import (
	_ "embed"

	"github.com/hhgyu/windows-portable-packager/internal/app"
)

//go:embed embedded/app.kbpkg
var embeddedPackageData []byte

func init() {
	app.SetEmbeddedPackage(embeddedPackageData)
}
