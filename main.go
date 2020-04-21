package main

import (
	"log"
	"os"
	"os/exec"
)

func getCommand() *exec.Cmd {
	cmd := exec.Command("/bin/bash", "-c", "for i in {0001..9999}; do d=\"$(date)\"; printf \"%04d : %s\n\" $i \"$d\"; sleep 10; done")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

func main() {

	if !GitIgnoreSupported {
		os.Stderr.WriteString("gitignore filter not supported  (git not on PATH)")
		os.Exit(1)
	}

	cmd := getCommand()
	if err := cmd.Start(); err != nil {
		log.Fatalf("initial command: '%v'", err)
	}

	err := WatchDir(".", func(e WatcherEvent) {
		if e.Error != nil {
			log.Printf("watch error: '%v'", e.Error)
			return
		}
		ignored, err := GitIgnored(e.Event.Path)
		if err != nil {
			log.Printf("error: '%v'", err)
		}
		if ignored {
			log.Printf("ignoring event: '%v'", e.Event)
			return
		}
		log.Printf("event: '%v'", e.Event)
		// Worry about grandchild processes?
		if err := cmd.Process.Kill(); err != nil {
			log.Fatalf("kill: '%v'", err)
		}
		log.Println("waiting...")
		cmd.Wait()
		log.Println("waited...")
		cmd = getCommand()
		log.Println("starting...")
		if err := cmd.Start(); err != nil {

			log.Fatalf("re-start: '%v'", err.Error())
		}
		log.Println("started")
	})
	if err != nil {
		log.Fatal(err)
	}
}
