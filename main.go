package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {

	watcher, err := NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	events := watcher.Events()
	go func() {
		for {
			event, ok := <-events
			if !ok {
				return
			}
			log.Printf("%+v", event)
		}
		done <- struct{}{}
	}()

	addWatcher := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return watcher.Watch(path)
	}

	if err := filepath.Walk(".", addWatcher); err != nil {
		log.Fatal(err)
	}

	<-done
}
