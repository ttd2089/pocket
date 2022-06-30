package main

import (
	"bytes"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// The Watcher interface the interface the application consumes for file
// watching. The standard implementation is a wrapper for fsnotify.

// EventType represents the kinds of file system events.
type EventType uint32

// The kinds of file system events we're interested in.
const (
	Create EventType = 1 << iota
	Write
	Remove
	Rename
)

// A set with the all the defined event types.
var eventTypes = []EventType{
	Create,
	Write,
	Remove,
	Rename,
}

func (t EventType) String() string {
	var buffer bytes.Buffer

	if t&Create == Create {
		buffer.WriteString("|CREATE")
	}
	if t&Remove == Remove {
		buffer.WriteString("|REMOVE")
	}
	if t&Write == Write {
		buffer.WriteString("|WRITE")
	}
	if t&Rename == Rename {
		buffer.WriteString("|RENAME")
	}
	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:]
}

// An FsEvent represents a file system event.
type FsEvent struct {

	// The path to the entry the event occured on.
	Path string

	// The type of the event that occurred.
	Type EventType
}

// A WatcherEvent is raised by a Watcher when an event or error occurs.
type WatcherEvent struct {

	// The file system event. If Error is not nil then the meaning of Event is
	// undefined.
	Event FsEvent

	// The error (possibly nil).
	Error error
}

// A Watcher raises file system events for watched directories and files.
type Watcher interface {

	// Tells the watcher to begin watching the given file or directory (not recursive).
	Watch(path string) error

	// Stops watching the given file or directory.
	Unwatch(path string) error

	// The channel that the watcher communicates events and errors on.
	Events() <-chan WatcherEvent

	// Stops the Watcher.
	Stop()
}

// A WatchFilter is a function that indicates whether an FsEvent should be
// filtered out of the event stream emitted by a Watcher.
type WatchFilter func(event FsEvent) (bool, error)

// NewWatcher creates a new Watcher.
// If filter is nil then no events will be filtered out.
func NewWatcher(filter WatchFilter) (Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if filter == nil {
		filter = func(event FsEvent) (bool, error) {
			return false, nil
		}
	}
	w := &watcherImpl{
		fsWatcher: fsWatcher,
		events:    make(chan WatcherEvent, 1),
		filter:    filter,
	}
	go w.start()
	return w, nil
}

// An implementation of Watcher using fsnotify.
type watcherImpl struct {
	fsWatcher *fsnotify.Watcher
	events    chan WatcherEvent
	filter    WatchFilter
}

// Tells the watcher to begin watching the given file or directory (not recursive).
func (w *watcherImpl) Watch(path string) error {
	if err := w.fsWatcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watcher: %v", err)
	}
	return nil
}

// Stops watching the given file or directory.
func (w *watcherImpl) Unwatch(path string) error {
	if err := w.fsWatcher.Remove(path); err != nil {
		return fmt.Errorf("failed to remove watcher: %v", err)
	}
	return nil
}

// The channel that the watcher communicates events and errors on.
func (w *watcherImpl) Events() <-chan WatcherEvent {
	return w.events
}

// Stops the Watcher.
func (w *watcherImpl) Stop() {
	if err := w.fsWatcher.Close(); err != nil {
		logger.Printf("Error closing fsnotify.Watcher: %v\n", err)
	}
}

// Starts consuming the fsnotify events and maps them to WatcherEvents.
func (w *watcherImpl) start() {
	defer close(w.events)
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if !isWatchedEvent(event) {
				break
			}
			fsEvent := newEvent(event)
			skip, err := w.filter(fsEvent)
			if err != nil {
				logger.Printf(
					"watcher filter error: event=%v, error=%v\n", fsEvent, err)
			} else if skip {
				logger.Printf("watcher filter: ignoring event %v\n", fsEvent)
				break
			}
			w.events <- WatcherEvent{
				Event: fsEvent,
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.events <- WatcherEvent{
				Error: err,
			}
		}
	}
}

// Indicates whether an fsnotify.Event represents and event that we care about.
func isWatchedEvent(event fsnotify.Event) bool {
	for _, t := range eventTypes {
		if EventType(event.Op) == t {
			return true
		}
	}
	return false
}

// Maps an fsnotify.Event to a WatcherEvent.
func newEvent(event fsnotify.Event) FsEvent {
	return FsEvent{
		Path: event.Name,
		Type: EventType(event.Op),
	}
}
