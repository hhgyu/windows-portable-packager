package app

import "sync/atomic"

var lockDetectLenient atomic.Bool

// ConfigureLockDetect toggles isFileLocked between strict (share=0) and
// lenient (DELETE + share-all) probes. Strict is the default and catches
// running EXEs as well as data files held by the app. Lenient restores
// the pre-fix behaviour for callers that hit false positives from
// indexers or AV scanning data files. The launcher calls this once per
// run after reading the package manifest.
func ConfigureLockDetect(lenient bool) {
	lockDetectLenient.Store(lenient)
}

func isLockDetectLenient() bool {
	return lockDetectLenient.Load()
}
