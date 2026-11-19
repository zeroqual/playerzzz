//go:build !windows

package trash

import "os"

func Delete(path string) error { return os.Remove(path) }
