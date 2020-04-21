package main

import (
	"log"
	"os"
)

func main() {
	if !GitIgnoreSupported {
		os.Stderr.WriteString("gitignore filter not supported  (git not on PATH)")
		os.Exit(1)
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
	})
	if err != nil {
		log.Fatal(err)
	}
}
