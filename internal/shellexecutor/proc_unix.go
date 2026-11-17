//go:build !windows

package shellexecutor

import (
	"os"
	"os/exec"
	"syscall"
)

// setupProcessGroup runs the command in its own process group, so on timeout
// the child and everything it spawned are killed together.
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
}
