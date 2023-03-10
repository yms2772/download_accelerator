//go:build windows

package cmd

import (
	"os/exec"
	"syscall"
)

func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
