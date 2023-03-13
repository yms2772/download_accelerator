//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func prepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
