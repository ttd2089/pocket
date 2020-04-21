package main

import (
	"log"
	"os"
	"path/filepath"
)

type Handler func(WatcherEvent)

func WatchDir(dir string, handle Handler) error {
	watcher, err := NewWatcher()
	if err != nil {
		return err
	}
	dw := dirWatcher{
		walkDirs,
		isDir,
		watcher,
	}
	return dw.watch(dir, handle)
}

type dirWatcher struct {
	walkDirs func(string, func(string) error) error
	isDir    func(string) bool
	watcher  Watcher
}

func (dw *dirWatcher) watch(dir string, handle Handler) error {
	if err := dw.watchDir(dir); err != nil {
		return err
	}
	events := dw.watcher.Events()
	for {
		event, ok := <-events
		if !ok {
			return nil
		}
		log.Printf("%+v", event)
		if event.Error == nil && event.Event.Type == Create && dw.isDir(event.Event.Path) {
			if err := dw.watchDir(event.Event.Path); err != nil {
				return err
			}
		}
		handle(event)
	}
	return nil
}

func (dw *dirWatcher) watchDir(dir string) error {
	return dw.walkDirs(dir, func(path string) error {
		err := dw.watcher.Watch(path)
		if err == nil {
			log.Printf("watchDir: watching %s\n", path)
		}
		return err
	})
}

func walkDirs(path string, walkFn func(string) error) error {
	return filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return walkFn(p)
		}
		return nil
	})
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		// This shouldn't be getting called for non-existent files, maybe log
		// the error?
		return false
	}
	return fi.IsDir()
}
