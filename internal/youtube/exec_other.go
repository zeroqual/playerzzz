//go:build !windows

package youtube

import "os/exec"

func hideConsole(*exec.Cmd) {}
