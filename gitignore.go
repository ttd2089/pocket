package main

import (
	"bytes"
	"errors"
	"os/exec"
)

var GitIgnoreSupported bool

func init() {
	_, err := exec.LookPath("git")
	GitIgnoreSupported = err == nil
}

func GitIgnored(path string) (bool, error) {
	cmd := exec.Command("git", "check-ignore", "-q", path)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil && err.Error() != "exit status 1" {
		return false, err
	}
	stderr := stderrBuf.String()
	if stderr != "" {
		return false, errors.New(stderr)
	}
	stdout := stdoutBuf.String()
	if stderr != "" {
		return false, errors.New(stdout)
	}
	return cmd.ProcessState.ExitCode() == 0, nil
}
