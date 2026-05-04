//go:build !windows

package app

type SingletonHandle struct{}

func AcquireSingleton(_ string) (*SingletonHandle, bool, error) {
	return &SingletonHandle{}, true, nil
}

func (s *SingletonHandle) Release() {}
