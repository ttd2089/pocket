//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os/exec"
	"path"
	"syscall"
	"time"
)

func newCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// Stops the given command process and waits for it to complete.
func stopProcess(proc *exec.Cmd) error {

	pid := proc.Process.Pid

	if err := syscall.Kill(-pid, syscall.SIGINT); err != nil {
		logger.Printf("failed to send SIGINT to %s: %s", path.Base(proc.Path), err.Error())
	} else {
		// If the sigint worked then give it a moment
		time.Sleep(200 * time.Millisecond)
	}

	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to kill %s: %s", path.Base(proc.Path), err.Error())
	}

	// This will error with 'signal: killed' or 'exit status 1' if we killed
	// process so just ignore it.
	_ = proc.Wait()
	return nil
}
