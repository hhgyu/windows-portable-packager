package app

var embeddedPackage []byte

func SetEmbeddedPackage(data []byte) {
	embeddedPackage = data
}

func HasEmbeddedPackage() bool {
	return len(embeddedPackage) > 11 && string(embeddedPackage[:11]) != "PLACEHOLDER"
}
