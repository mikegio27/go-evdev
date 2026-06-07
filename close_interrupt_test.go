package evdev

import (
	"os"
	"testing"
	"time"
)

// TestCloseInterruptsReadOne verifies that Close unblocks a ReadOne that is
// waiting for input. This is the property that lets a long-lived reader (e.g. a
// remapper engine) be cancelled cleanly: it holds only because Open is
// non-blocking (poller-managed) and ioctls go through control rather than Fd,
// which would otherwise revert the file to blocking mode.
//
// It creates a virtual device and reads from its event node, so it needs write
// access to /dev/uinput (root) and skips otherwise.
func TestCloseInterruptsReadOne(t *testing.T) {
	f, err := os.OpenFile(uinputPath, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot write %s (need root): %v", uinputPath, err)
	}
	f.Close()

	const name = "go-evdev close-interrupt test"
	id := InputID{BusType: BUS_USB, Vendor: 0x9991, Product: 0x9992, Version: 1}
	v, err := CreateVirtualDevice(name, id, Capabilities{Keys: []EvCode{KEY_A}})
	if err != nil {
		t.Fatalf("CreateVirtualDevice: %v", err)
	}
	defer v.Close()

	// Find the event node the new device exposes (udev may take a moment).
	dev := openByName(t, name)
	defer dev.Close()

	// Read with no input pending: it must block until we Close.
	readErr := make(chan error, 1)
	go func() {
		_, err := dev.ReadOne()
		readErr <- err
	}()

	// Confirm it is actually blocked (not returning immediately).
	select {
	case err := <-readErr:
		t.Fatalf("ReadOne returned before any input or Close: err=%v", err)
	case <-time.After(100 * time.Millisecond):
	}

	// Close must unblock the in-flight ReadOne promptly.
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case err := <-readErr:
		if err == nil {
			t.Fatal("ReadOne returned nil error after Close; expected a closed-file error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadOne did not return after Close — the read is not interruptible")
	}
}

// openByName scans for the device node whose name matches and opens it, retrying
// briefly while udev creates the node.
func openByName(t *testing.T, name string) *Device {
	t.Helper()
	for range 50 {
		paths, err := ListDevicePaths()
		if err == nil {
			for _, p := range paths {
				d, err := Open(p)
				if err != nil {
					continue
				}
				n, err := d.Name()
				if err == nil && n == name {
					return d
				}
				d.Close()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Couldn't open our own device's node (typically event nodes aren't readable
	// without the input group / root). That's an environment limit, not a failure.
	t.Skipf("could not open event node for virtual device %q (need read access to /dev/input/event*)", name)
	return nil
}
