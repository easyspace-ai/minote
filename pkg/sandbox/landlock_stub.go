//go:build !linux

package sandbox

import "errors"

// CheckLandlockAvailable reports whether the current kernel exposes Landlock.
func CheckLandlockAvailable() bool {
	return false
}

func probeLandlock(_ string) error {
	return errors.New("landlock is only supported on Linux")
}

func applyLandlock(_ string) error {
	return errors.New("landlock is only supported on Linux")
}
