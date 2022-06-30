package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pborman/getopt/v2"
)

var productVersion = "0.0.0"

var logger *log.Logger

func init() {

	logger = log.New(ioutil.Discard, "", log.LstdFlags)

	initGitignore()
}

func main() {

	cli := getopt.New()
	cli.SetProgram("pocket")
	cli.SetParameters("<cmd> [<cmd-args>]")
	usage := func(w io.Writer) {
		cli.PrintUsage(w)
		fmt.Fprintf(w, "<cmd>               the command to run on file changes\n")
		fmt.Fprintf(w, "<cmd-args>          the arguments for <cmd>\n")
	}
	cli.SetUsage(func() { usage(os.Stderr) })

	chdirOpt := cli.StringLong("chdir", 'C', "", "the directory to run in", "<dir>")
	helpFlag := cli.BoolLong("help", 'h', "display help")
	logFlag := cli.BoolLong("log", 'L', "write application logs to stderr")
	versionFlag := cli.BoolLong("version", 'v', "display product version")

	cli.Parse(os.Args)

	args := cli.Args()

	if *helpFlag {
		usage(os.Stdout)
		return
	}

	if *versionFlag {
		fmt.Fprintf(os.Stdout, "pocket version %s", productVersion)
		return
	}

	if len(args) == 0 {
		usage(os.Stdout)
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

	filter := func(event FsEvent) (bool, error) {
		return GitIgnored(event.Path)
	}

	watcher, err := NewWatcher(filter)
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
	proc := newCommand(cmd, args...)
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
