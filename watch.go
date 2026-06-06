package evdev

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// DeviceAction reports whether a device node appeared or disappeared.
type DeviceAction int

const (
	DeviceAdded DeviceAction = iota
	DeviceRemoved
)

// DeviceEvent is a hotplug notification for a single evdev node.
type DeviceEvent struct {
	Action DeviceAction
	Path   string // e.g. "/dev/input/event7"
}

// watchMask covers nodes appearing (create / moved-in) and disappearing
// (delete / moved-out) under the watched directory.
const watchMask = unix.IN_CREATE | unix.IN_DELETE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM

// Watcher reports evdev device nodes appearing and disappearing under
// /dev/input, so a long-running program can pick up hotplugged devices (e.g. a
// swapped mouse) without restarting.
//
// A Watcher reports only changes after it is created, not the devices already
// present. To avoid missing a device that appears during startup, create the
// Watcher first, then enumerate with ListDevices; any node that races in will
// also arrive on Events.
//
// Events is closed when the Watcher stops (via Close or on a fatal error); after
// it closes, check Errors. Open the reported paths with Open, tolerating a
// transient permission error while udev finishes applying access rules.
type Watcher struct {
	dir   string
	ifd   int
	wakeR int
	wakeW int

	events chan DeviceEvent
	errs   chan error
	quit   chan struct{}
	done   chan struct{}

	closeOnce sync.Once
	closeErr  error
}

// NewWatcher starts watching /dev/input for device nodes being added or removed.
// The caller must Close the returned Watcher.
func NewWatcher() (*Watcher, error) { return newWatcher(devDir) }

func newWatcher(dir string) (*Watcher, error) {
	ifd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, fmt.Errorf("evdev: inotify init: %w", err)
	}
	if _, err := unix.InotifyAddWatch(ifd, dir, watchMask); err != nil {
		unix.Close(ifd)
		return nil, fmt.Errorf("evdev: watch %s: %w", dir, err)
	}
	// Self-pipe used to interrupt the poll loop on Close.
	var p [2]int
	if err := unix.Pipe2(p[:], unix.O_CLOEXEC|unix.O_NONBLOCK); err != nil {
		unix.Close(ifd)
		return nil, fmt.Errorf("evdev: watcher pipe: %w", err)
	}
	w := &Watcher{
		dir:    dir,
		ifd:    ifd,
		wakeR:  p[0],
		wakeW:  p[1],
		events: make(chan DeviceEvent),
		errs:   make(chan error, 1),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

// Events returns the channel of hotplug notifications. It is closed when the
// Watcher stops.
func (w *Watcher) Events() <-chan DeviceEvent { return w.events }

// Errors returns the channel carrying a fatal watch error, if any. It receives
// at most one error, sent before Events is closed.
func (w *Watcher) Errors() <-chan error { return w.errs }

// Close stops the Watcher and releases its file descriptors. It is safe to call
// more than once.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		close(w.quit)
		unix.Write(w.wakeW, []byte{0}) // wake the poll loop
		<-w.done                       // wait for it to exit before closing fds
		w.closeErr = firstErr(unix.Close(w.ifd), unix.Close(w.wakeR), unix.Close(w.wakeW))
	})
	return w.closeErr
}

func (w *Watcher) loop() {
	defer close(w.done)
	defer close(w.events)

	fds := []unix.PollFd{
		{Fd: int32(w.ifd), Events: unix.POLLIN},
		{Fd: int32(w.wakeR), Events: unix.POLLIN},
	}
	buf := make([]byte, 16*unix.SizeofInotifyEvent+4096)
	for {
		if _, err := unix.Poll(fds, -1); err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			w.fail(err)
			return
		}
		if fds[1].Revents&unix.POLLIN != 0 {
			return // Close requested
		}
		n, err := unix.Read(w.ifd, buf)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			w.fail(err)
			return
		}
		if !w.emit(buf[:n]) {
			return
		}
	}
}

// emit parses a batch of inotify records and forwards the ones for evdev nodes.
// It returns false if Close was requested mid-send. The inotify_event header is
// fixed-size (wd, mask, cookie, len as uint32s) followed by a NUL-padded name;
// fields are host-endian, decoded here without unsafe.
func (w *Watcher) emit(buf []byte) bool {
	for len(buf) >= unix.SizeofInotifyEvent {
		mask := binary.NativeEndian.Uint32(buf[4:8])
		nameLen := int(binary.NativeEndian.Uint32(buf[12:16]))
		end := unix.SizeofInotifyEvent + nameLen
		if end > len(buf) {
			break
		}
		name := string(bytes.TrimRight(buf[unix.SizeofInotifyEvent:end], "\x00"))
		buf = buf[end:]

		if !strings.HasPrefix(name, "event") {
			continue
		}
		var action DeviceAction
		switch {
		case mask&(unix.IN_CREATE|unix.IN_MOVED_TO) != 0:
			action = DeviceAdded
		case mask&(unix.IN_DELETE|unix.IN_MOVED_FROM) != 0:
			action = DeviceRemoved
		default:
			continue
		}
		select {
		case w.events <- DeviceEvent{Action: action, Path: filepath.Join(w.dir, name)}:
		case <-w.quit:
			return false
		}
	}
	return true
}

// fail delivers the first fatal error without blocking; later errors are dropped
// since the loop stops after the first.
func (w *Watcher) fail(err error) {
	select {
	case w.errs <- fmt.Errorf("evdev: watch %s: %w", w.dir, err):
	default:
	}
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
