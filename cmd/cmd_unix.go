//go:build !windows

package cmd

import (
	"os/exec"
)

func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
