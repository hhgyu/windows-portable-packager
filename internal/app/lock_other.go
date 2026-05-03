//go:build !windows

package app

func detectLockedFiles(_ string) ([]string, error) {
	return nil, nil
}
