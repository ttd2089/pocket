package main

import (
	"fmt"
	"os/exec"
	"path"
)

func newCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

// Stops the given command process and waits for it to complete.
func stopProcess(proc *exec.Cmd) error {

	taskkill := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(proc.Process.Pid))
	if err := taskkill.Run(); err != nil {
		return fmt.Errorf("error invoking taskkill on %s: %s", path.Base(proc.Path), err.Error())
	}

	// This will error with 'signal: killed' or 'exit status 1' if we killed
	// process so just ignore it.
	_ = proc.Wait()
	return nil
}
