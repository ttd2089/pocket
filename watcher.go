package main

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
)

// The Watcher interface the interface the application consumes for file
// watching. The standard implementation is a wrapper for fsnotify.

// The type to represent the kinds of file system events.
type EventType uint32

// The kinds of file system events we're interested in.
const (
	Create EventType = 1 << iota
	Write
	Remove
	Rename
)

// A set with the all the defined event types.
var eventTypes []EventType = []EventType{
	Create,
	Write,
	Remove,
	Rename,
}

// A file system event.
type FsEvent struct {

	// The path to the entry the event occured on.
	Path string

	// The type of the event that occurred.
	Type EventType
}

// The event type a Watcher raises.
type WatcherEvent struct {

	// The file system event. If Error is not nil then the meaning of Event is
	// undefined.
	Event FsEvent

	// The error (possibly nil).
	Error error
}

// The interface for file watching.
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

// An implementation of Watcher using fsnotify.
type watcherImpl struct {
	fsWatcher *fsnotify.Watcher
	events    chan WatcherEvent
}

// Tells the watcher to begin watching the given file or directory (not recursive).
func (w *watcherImpl) Watch(path string) error {
	if err := w.fsWatcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watcher: %s", err.Error())
	}
	return nil
}

// Stops watching the given file or directory.
func (w *watcherImpl) Unwatch(path string) error {
	if err := w.fsWatcher.Remove(path); err != nil {
		return fmt.Errorf("failed to remove watcher: %s", err.Error())
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
		log.Printf("Error closing fsnotify.Watcher: %+v", err)
	}
}

// Creates a new Watcher.
func NewWatcher() (Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &watcherImpl{
		fsWatcher: fsWatcher,
		events:    make(chan WatcherEvent),
	}
	go w.start()
	return w, nil
}

// Starts consuming the fsnotify events and maps them to WatcherEvents.
func (w *watcherImpl) start() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				break
			}
			if isWatchedEvent(event) {
				w.events <- newEvent(event)
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				break
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
func newEvent(event fsnotify.Event) WatcherEvent {
	return WatcherEvent{
		Event: FsEvent{
			Path: event.Name,
			Type: EventType(event.Op),
		},
		Error: nil,
	}
}
