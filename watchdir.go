package main

import (
	"os"
	"path/filepath"
	"time"
)

const debounceCount int = 15
const debounceInterval time.Duration = 1 * time.Second

// A Handler is a function that handles events for WatchDir.
type Handler func(WatcherEvent) error

// Watch watches a directory for file system changes until the watcher is
// stopped or fails to watch a sub-directory, or the handler returns an error.
func Watch(dir string, watcher Watcher, handle Handler) error {
	dw := dirWatcher{
		walkDirs,
		isDir,
		watcher,
		debounceCount,
		debounceInterval,
	}
	return dw.watch(dir, handle)
}

// The context for watching a directory.
type dirWatcher struct {

	// A function that walks the directory tree of a given directory path and
	// calls a given function on each directory path.
	walkDirs func(string, func(string) error) error

	// A function that tests whether a given path refers to a directory.
	isDir func(string) bool

	// The Watcher to use to detect file system changes.
	watcher Watcher

	// The max number of events to debounce.
	debounceCount int

	// The interval within which subsequent events must occur to trigger
	// debouncing.
	debounceInterval time.Duration
}

// Implements WatchDir using the target dirWatcher.
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

		dw.processEvent(event)

		// Debounce the event queue a bit. Our command will reflect the state
		// of the system when it runs so we don't actually care about the
		// of the events other than whether one should trigger a re-run.
	DEBOUNCE:
		for i := 0; i < dw.debounceCount; i++ {
			select {
			case e, _ := <-events:
				if err := dw.processEvent(e); err != nil {
					return err
				}
				break
			case _ = <-time.After(dw.debounceInterval):
				break DEBOUNCE
			}
		}

		if err := handle(event); err != nil {
			return err
		}
	}
}

// Adds watchers to the given directory and all of its subdirectories.
func (dw *dirWatcher) watchDir(dir string) error {
	return dw.walkDirs(dir, func(path string) error {
		return dw.watcher.Watch(path)
	})
}

// Encapsulates the dirWatcher's own handling of the event. This exists
// primarily to separate the dirWatcher's handling from the Handler since the
// latter is debounced and the former is not.
func (dw *dirWatcher) processEvent(event WatcherEvent) error {
	if event.Error != nil {
		logger.Printf("watchdir error event: %v\n", event.Error)
		return nil
	}

	logger.Printf("watchdir fs event: %v\n", event.Event)

	path := event.Event.Path
	if event.Event.Type == Create && dw.isDir(path) {
		if err := dw.watchDir(path); err != nil {
			logger.Printf("watchdir watch error: %s %v\n", path, err)
			return err
		}
	}

	return nil
}

// Walks the real file system calling walkFn on the directory entries it finds.
func walkDirs(path string, walkFn func(string) error) error {
	return filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			logger.Printf("watchdir walk error: %v\n", err)
			return err
		}
		if fi.IsDir() {
			return walkFn(p)
		}
		return nil
	})
}

// Tests whether the given path is a directory in the real file system.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		logger.Printf("watchdir stat error: %s %v\n", path, err)
		return false
	}
	return fi.IsDir()
}
