//go:build windows

package shellexecutor

import (
	"os/exec"
	"strconv"
)

// taskkill /T kills the whole process tree.
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		if err != nil {
			return cmd.Process.Kill() // fall back to the direct child
		}
		return nil
	}
}
