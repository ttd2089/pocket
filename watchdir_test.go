package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/go-test/deep"
)

func Test_WatchDir(t *testing.T) {

	t.Run("Returns error when Watcher#Watch returns error", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		watcher := &testWatcher{
			watch: func(string) error { return expectedErr },
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				return walkFn(dir)
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
		}
		close(watcher.events)
		err := dw.watch("/foo/", func(e WatcherEvent) error {
			t.Error("unxpected call to handle()")
			return nil
		})
		if err != expectedErr {
			t.Errorf("watchDir(); expected %+v, got %+v", expectedErr, err)
		}
	})

	t.Run("Calls handle() with the events from Watcher#Events", func(t *testing.T) {
		watcher := &testWatcher{
			watch: func(string) error { return nil },
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				return walkFn(dir)
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
		}
		expectedEvents := []WatcherEvent{
			{
				Event: FsEvent{Path: "/foo/bar/baz", Type: Create},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/bar/baz", Type: Remove},
				Error: nil,
			},
			{
				Event: FsEvent{}, Error: errors.New("some error"),
			},
		}
		go func() {
			for _, event := range expectedEvents {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		actualEvents := []WatcherEvent{}
		if err := dw.watch("/foo/", func(e WatcherEvent) error {
			actualEvents = append(actualEvents, e)
			return nil
		}); err != nil {
			t.Errorf("unexpected error from watch(): %+v", err)
		}
		if len(deep.Equal(expectedEvents, actualEvents)) != 0 {
			t.Errorf("handle(); expected %+v, got %+v", expectedEvents, actualEvents)
		}
	})

	t.Run("Returns nil after handling events successfully", func(t *testing.T) {
		watcher := &testWatcher{
			watch: func(string) error { return nil },
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				return walkFn(dir)
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
		}
		expectedEvents := []WatcherEvent{
			{Event: FsEvent{}, Error: nil},
		}
		go func() {
			for _, event := range expectedEvents {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		if err := dw.watch("/foo/", func(_ WatcherEvent) error { return nil }); err != nil {
			t.Errorf("watchDir(); expected nil, got %+v", err)
		}
	})

	t.Run("Returns error when handle returns error", func(t *testing.T) {
		watcher := &testWatcher{
			watch: func(string) error { return nil },
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				return walkFn(dir)
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
		}
		expectedError := errors.New("some error")
		events := []WatcherEvent{
			{Event: FsEvent{}, Error: nil},
			{Event: FsEvent{}, Error: nil},
			{Event: FsEvent{}, Error: expectedError},
		}
		go func() {
			for _, event := range events {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		handle := func(e WatcherEvent) error {
			return e.Error
		}
		if err := dw.watch("/foo/", handle); err != expectedError {
			t.Errorf("watchDir(); expected %+v, got %+v", expectedError, err)
		}
	})

	t.Run("Watches all sub-directories in the target directory", func(t *testing.T) {
		expectedWatched := []string{"/foo/", "/foo/bar/", "/foo/baz/"}
		actualWatched := []string{}
		watcher := &testWatcher{
			watch: func(dir string) error {
				actualWatched = append(actualWatched, dir)
				return nil
			},
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				for _, path := range expectedWatched {
					if err := walkFn(path); err != nil {
						t.Errorf("unexpected error from walkFn(): %+v", err)
					}
				}
				return nil
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
		}
		close(watcher.events)
		if err := dw.watch("/foo/", func(e WatcherEvent) error { return nil }); err != nil {
			t.Errorf("unexpected error from watch(): %+v", err)
		}
		if !reflect.DeepEqual(expectedWatched, actualWatched) {
			t.Errorf("watch(); expected %+v, got %+v", expectedWatched, actualWatched)
		}
	})

	t.Run("Watches newly created directories and their sub-directories", func(t *testing.T) {
		expectedWatched := []string{"/foo/", "/foo/bar/", "/foo/bar/baz/"}
		actualWatched := []string{}
		watcher := &testWatcher{
			watch: func(dir string) error {
				actualWatched = append(actualWatched, dir)
				return nil
			},
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				// When we add the initial directory it's all we'll find, then
				// when the event is raised for /foo/bar/ CREATE we'll find it
				// and its subdir /foo/bar/baz/.
				if dir == "/foo/" {
					return walkFn(dir)
				}
				if dir == "/foo/bar/" {
					if err := walkFn("/foo/bar/"); err != nil {
						t.Errorf("unexpected error from walkFn(): %+v", err)
					}
					return walkFn("/foo/bar/baz/")
				}
				t.Errorf("unexpected call to walkDirs('%s')", dir)
				return nil
			},
			isDir: func(path string) bool {
				// We will raise a few create events but only /foo/bar/ is a
				// new directory.
				return path == "/foo/bar/"
			},
			watcher: watcher,
		}
		events := []WatcherEvent{
			{
				Event: FsEvent{Path: "/foo/file.txt", Type: Create},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/file.txt", Type: Write},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/file.txt", Type: Rename},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/stuff.txt", Type: Create},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/stuff.txt", Type: Remove},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/bar/", Type: Create},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/bar/", Type: Remove},
				Error: nil,
			},
		}
		go func() {
			for _, event := range events {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		if err := dw.watch("/foo/", func(e WatcherEvent) error { return nil }); err != nil {
			t.Errorf("unexpected error from watch(): %+v", err)
		}
		if !reflect.DeepEqual(expectedWatched, actualWatched) {
			t.Errorf("watch(); expected %+v, got %+v", expectedWatched, actualWatched)
		}
	})

	t.Run("Debounced events trigger new directory watches", func(t *testing.T) {
		expectedWatched := []string{"/foo/", "/foo/bar/", "/foo/bar/baz/"}
		actualWatched := []string{}
		watcher := &testWatcher{
			watch: func(dir string) error {
				actualWatched = append(actualWatched, dir)
				return nil
			},
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				// When we add the initial directory it's all we'll find, then
				// when the event is raised for /foo/bar/ CREATE we'll find it
				// and its subdir /foo/bar/baz/.
				if dir == "/foo/" {
					return walkFn(dir)
				}
				if dir == "/foo/bar/" {
					if err := walkFn("/foo/bar/"); err != nil {
						t.Errorf("unexpected error from walkFn(): %+v", err)
					}
					return walkFn("/foo/bar/baz/")
				}
				t.Errorf("unexpected call to walkDirs('%s')", dir)
				return nil
			},
			isDir: func(path string) bool {
				// We will raise a few create events but only /foo/bar/ is a
				// new directory.
				return path == "/foo/bar/"
			},
			watcher: watcher,
			// Make sure the directory add is debounced after the file remove.
			debounceCount:    2,
			debounceInterval: time.Minute,
		}
		events := []WatcherEvent{
			{
				Event: FsEvent{Path: "/foo/stuff.txt", Type: Remove},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/bar/", Type: Create},
				Error: nil,
			},
		}
		go func() {
			for _, event := range events {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		if err := dw.watch("/foo/", func(e WatcherEvent) error { return nil }); err != nil {
			t.Errorf("unexpected error from watch(): %+v", err)
		}
		if !reflect.DeepEqual(expectedWatched, actualWatched) {
			t.Errorf("watch(); expected %+v, got %+v", expectedWatched, actualWatched)
		}
	})

	t.Run("Debounced events do not trigger calls to handle", func(t *testing.T) {

		watcher := &testWatcher{
			watch: func(string) error { return nil },
			unwatch: func(string) error {
				t.Error("unexpected call to Unwatch()")
				return nil
			},
			events: make(chan WatcherEvent),
		}
		dw := dirWatcher{
			walkDirs: func(dir string, walkFn func(string) error) error {
				return walkFn(dir)
			},
			isDir:   func(_ string) bool { return false },
			watcher: watcher,
			// Make sure the directory add is debounced after the file remove.
			debounceCount:    2,
			debounceInterval: time.Minute,
		}
		events := []WatcherEvent{
			{
				Event: FsEvent{Path: "/foo/bar/baz", Type: Create},
				Error: nil,
			},
			{
				Event: FsEvent{Path: "/foo/bar/baz", Type: Remove},
				Error: nil,
			},
			{
				Event: FsEvent{}, Error: errors.New("some error"),
			},
		}
		go func() {
			for _, event := range events {
				watcher.events <- event
			}
			close(watcher.events)
		}()
		actualEvents := []WatcherEvent{}
		if err := dw.watch("/foo/", func(e WatcherEvent) error {
			actualEvents = append(actualEvents, e)
			return nil
		}); err != nil {
			t.Errorf("unexpected error from watch(): %+v", err)
		}
		expectedEvents := []WatcherEvent{events[0]}
		if len(deep.Equal(expectedEvents, actualEvents)) != 0 {
			t.Errorf("handle(); expected %+v, got %+v", expectedEvents, actualEvents)
		}
	})
}

type testWatcher struct {
	watch   func(string) error
	unwatch func(string) error
	events  chan WatcherEvent
}

func (w *testWatcher) Watch(path string) error {
	return w.watch(path)
}

func (w *testWatcher) Unwatch(path string) error {
	return w.unwatch(path)
}

func (w *testWatcher) Events() <-chan WatcherEvent {
	return w.events
}

func (w *testWatcher) Stop() {
	close(w.events)
}
