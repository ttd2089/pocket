package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// GitIgnoreSupported indicates whether the GitIgnored function is available.
var GitIgnoreSupported = false

func initGitignore() {
	if _, err := exec.LookPath("git"); err != nil {
		logger.Printf("gitignore not supported: %v\n", err)
	} else {
		GitIgnoreSupported = true
	}
}

// GitIgnored checks if the given path is gitignored.
func GitIgnored(path string) (bool, error) {
	cmd := exec.Command("git", "check-ignore", "-q", path)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	// Run() returns the "exit status 1" error when the file isn't ignored so
	// that's not an error for us.
	if err := cmd.Run(); err != nil && err.Error() != "exit status 1" {
		logger.Printf("gitignore error: %v\n", err)
		return false, err
	}
	stderr := stderrBuf.String()
	if stderr != "" {
		err := errors.New(stderr)
		logger.Printf("gitignore error: %v\n", err)
		return false, err
	}
	stdout := stdoutBuf.String()
	if stdout != "" {
		err := errors.New(stdout)
		logger.Printf("gitignore error: %v\n", err)
		return false, err
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode == 0 {
		return true, nil
	}
	// git does not consider the .git folder and contents to be ignored.
	if exitCode == 1 && strings.HasPrefix(path, ".git") {
		return true, nil
	}
	return false, nil
}
