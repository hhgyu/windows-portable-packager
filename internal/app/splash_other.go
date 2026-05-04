//go:build !windows

package app

import "time"

type SplashWindow struct{}

func ShowSplashFromData(data []byte, ext string) (*SplashWindow, error) {
	return nil, nil
}

func ShowSplash(imagePath string) (*SplashWindow, error) {
	return nil, nil
}

func (sw *SplashWindow) SetMinVisible(_ time.Duration) {}

func (sw *SplashWindow) Close() {
	if sw == nil {
		return
	}
}

func (sw *SplashWindow) ForceClose() {
	if sw == nil {
		return
	}
}
