//go:build !windows

package main

import (
	"os/exec"
)

func prepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
