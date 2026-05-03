//go:build !windows

package app

type SplashWindow struct{}

func ShowSplashFromData(data []byte, ext string) (*SplashWindow, error) {
	return nil, nil
}

func ShowSplash(imagePath string) (*SplashWindow, error) {
	return nil, nil
}

func (sw *SplashWindow) Close() {
	if sw == nil {
		return
	}
}
