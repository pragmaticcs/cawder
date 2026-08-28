//go:build windows

package tools

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setPlatformAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	return exec.Command(
		"taskkill",
		"/F",
		"/T",
		"/PID",
		strconv.Itoa(cmd.Process.Pid),
	).Run()
}
