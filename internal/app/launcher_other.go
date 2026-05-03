//go:build !windows

package app

import "os/exec"

func setSysProcAttr(_ *exec.Cmd) {}
