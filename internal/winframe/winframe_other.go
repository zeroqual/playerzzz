//go:build !windows

package winframe

func Supported() bool { return false }

func Enable(uintptr) error     { return nil }
func SetMetrics(int32, int32)  {}
func Minimize(uintptr)         {}
func ToggleMaximize(uintptr)   {}
func IsMaximized(uintptr) bool { return false }
