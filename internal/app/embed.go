package app

var embeddedPackage []byte
var embeddedSplash []byte
var embeddedSplashExt string

func SetEmbeddedPackage(data []byte) {
	embeddedPackage = data
}

func HasEmbeddedPackage() bool {
	return len(embeddedPackage) > 11 && string(embeddedPackage[:11]) != "PLACEHOLDER"
}

func SetEmbeddedSplash(data []byte, ext string) {
	embeddedSplash = data
	embeddedSplashExt = ext
}

func HasEmbeddedSplash() bool {
	return len(embeddedSplash) > 0
}

func GetEmbeddedSplash() ([]byte, string) {
	return embeddedSplash, embeddedSplashExt
}
