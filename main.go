package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pborman/getopt/v2"
)

func main() {

	cli := getopt.New()
	cli.SetProgram("pocket")
	cli.SetParameters("<cmd> [<cmd-args>]")
	usage := func() {
		cli.PrintUsage(os.Stderr)
		fmt.Fprintf(os.Stderr, "<cmd>               the command to run on file changes\n")
		fmt.Fprintf(os.Stderr, "<cmd-args>          the arguments for <cmd>\n")
	}
	cli.SetUsage(usage)

	chdirOpt := cli.StringLong("chdir", 'C', "", "the directory to run in", "<dir>")
	helpFlag := cli.BoolLong("help", 'h', "display help")
	logFlag := cli.BoolLong("log", 'L', "write application logs to stderr")

	cli.Parse(os.Args)
	args := cli.Args()
	if *helpFlag || len(args) == 0 {
		usage()
		return
	}

	if *chdirOpt != "" {
		if err := os.Chdir(*chdirOpt); err != nil {
			die(fmt.Sprintf("failed to cd to %s: %v", *chdirOpt, err))
		}
	}

	if *logFlag {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	run(args[0], args[1:]...)
}

// Runs the command and re-starts it on file changes.
func run(cmd string, args ...string) {

	// TODO: support other filter options?
	if !GitIgnoreSupported {
		die("error: gitignore filter not supported")
	}

	filter := GitIgnored

	watcher, err := NewWatcher()
	if err != nil {
		die(err.Error())
	}

	proc, err := startProcess(cmd, args)
	if err != nil {
		die(err.Error())
	}

	handle := func(e WatcherEvent) error {
		if e.Error != nil {
			logger.Printf("watcher error: %v", e.Error)
			return nil
		}
		skip, err := filter(e.Event.Path)
		if err != nil {
			logger.Printf("filter error: %v", err)
			return nil
		}
		if skip {
			logger.Printf("ignoring event: %v", e.Event)
			return nil
		}
		if err = stopProcess(proc); err != nil {
			return err
		}
		if proc, err = startProcess(cmd, args); err != nil {
			return err
		}
		return nil
	}

	if err := Watch(".", watcher, handle); err != nil {
		die(err.Error())
	}
}

// Write the given message to stderr and exit the process. This message
// written whether logging is enabled or not.
func die(message string) {
	os.Stderr.WriteString(message)
	os.Exit(1)
}

// Starts a process using the given cmd and args.
func startProcess(cmd string, args []string) (*exec.Cmd, error) {
	proc := exec.Command(cmd, args...)
	proc.Stderr = os.Stderr
	proc.Stdout = os.Stdout
	if err := proc.Start(); err != nil {
		tokens := append([]string{cmd}, args...)
		for i, v := range tokens {
			v = strings.ReplaceAll(v, "\"", "\\\"")
			if strings.Contains(v, " ") {
				v = fmt.Sprintf("\"%s\"", v)
			}
			tokens[i] = v
		}
		return nil, fmt.Errorf("failed to run '%s': %v", strings.Join(tokens, " "), err)
	}
	return proc, nil
}

// Stops the given command process and waits for it to complete.
func stopProcess(proc *exec.Cmd) error {
	if err := proc.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop %s: %v", path.Base(proc.Path), err)
	}
	// This error because we killed the process.
	_ = proc.Wait()
	return nil
}
