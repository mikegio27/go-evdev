package evdev

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWatcher drives the inotify watcher against a temp directory, so it needs
// no real hardware: creating/removing files named like device nodes stands in
// for hotplug.
func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	w, err := newWatcher(dir)
	if err != nil {
		t.Fatalf("newWatcher: %v", err)
	}
	defer w.Close()

	path := filepath.Join(dir, "event5")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if ev := waitEvent(t, w); ev.Action != DeviceAdded || ev.Path != path {
		t.Errorf("add: got %+v, want {DeviceAdded %s}", ev, path)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if ev := waitEvent(t, w); ev.Action != DeviceRemoved || ev.Path != path {
		t.Errorf("remove: got %+v, want {DeviceRemoved %s}", ev, path)
	}

	// Non-evdev nodes (e.g. mouse0, js0) must be ignored.
	if err := os.WriteFile(filepath.Join(dir, "mouse0"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-w.Events():
		t.Errorf("unexpected event for non-evdev node: %+v", ev)
	case err := <-w.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

func waitEvent(t *testing.T, w *Watcher) DeviceEvent {
	t.Helper()
	select {
	case ev := <-w.Events():
		return ev
	case err := <-w.Errors():
		t.Fatalf("watch error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for device event")
	}
	return DeviceEvent{}
}

// TestWatcherClose verifies Close is idempotent and stops the watcher cleanly.
func TestWatcherClose(t *testing.T) {
	w, err := newWatcher(t.TempDir())
	if err != nil {
		t.Fatalf("newWatcher: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if _, ok := <-w.Events(); ok {
		t.Error("Events channel should be closed after Close")
	}
}
