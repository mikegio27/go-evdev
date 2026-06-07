package evdev

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// InputID mirrors the kernel's struct input_id, returned by EVIOCGID.
type InputID struct {
	BusType BusType
	Vendor  uint16
	Product uint16
	Version uint16
}

// Device is an open evdev input device (a /dev/input/event* node).
type Device struct {
	f    *os.File
	path string
}

// capBufBytes sizes a capability bitmask buffer large enough for any event
// type's code space (KEY_* is the largest).
const capBufBytes = (int(KEY_MAX) + 8) / 8

// Open opens the evdev device at path for reading.
//
// The device is opened non-blocking so its reads go through Go's runtime poller:
// a ReadOne blocked waiting for input is then interrupted by Close (returning a
// closed-file error), which lets a long-lived reader be cancelled cleanly. For
// this to hold, ioctls must not be issued via Fd() (which reverts the file to
// blocking mode and detaches it from the poller) — Device routes them through
// control instead.
func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return &Device{f: f, path: path}, nil
}

// Close closes the underlying device file. It also unblocks a concurrent ReadOne.
func (d *Device) Close() error { return d.f.Close() }

// Path returns the device path the Device was opened with.
func (d *Device) Path() string { return d.path }

// Fd returns the underlying file descriptor.
//
// Calling Fd reverts the file to blocking mode and removes it from the runtime
// poller, so a concurrent ReadOne will no longer be interruptible by Close. It
// is retained for callers that need the raw descriptor; internal ioctls use
// control to avoid this.
func (d *Device) Fd() uintptr { return d.f.Fd() }

// control runs fn with the device's file descriptor without detaching the file
// from the runtime poller (unlike Fd), so a concurrent ReadOne stays
// interruptible by Close. The fd is valid only for the duration of fn.
func (d *Device) control(fn func(fd uintptr) error) error {
	rc, err := d.f.SyscallConn()
	if err != nil {
		return err
	}
	var fnErr error
	if cErr := rc.Control(func(fd uintptr) { fnErr = fn(fd) }); cErr != nil {
		return cErr
	}
	return fnErr
}

// ReadOne blocks until one event is available and returns it. It returns
// io.EOF when the device disappears.
func (d *Device) ReadOne() (InputEvent, error) {
	var ev InputEvent
	// InputEvent's memory layout matches the kernel's struct input_event byte
	// for byte (see event.go), so we read straight into it via an aliasing byte
	// view. This is a deliberate zero-copy fast path for the hottest call in the
	// library; the alternative (encoding/binary into a scratch buffer) would copy
	// and allocate on every event.
	buf := (*[sizeofInputEvent]byte)(unsafe.Pointer(&ev))[:]
	if _, err := io.ReadFull(d.f, buf); err != nil {
		return InputEvent{}, err
	}
	return ev, nil
}

// Read fills buf with as many events as are available in a single read,
// blocking until at least one is, and returns the count. It is more efficient
// than ReadOne for high event rates.
func (d *Device) Read(buf []InputEvent) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), len(buf)*sizeofInputEvent)
	n, err := d.f.Read(b)
	return n / sizeofInputEvent, err
}

// Name returns the device name (EVIOCGNAME), e.g. "AT Translated Set 2 keyboard".
func (d *Device) Name() (string, error) { return d.ioctlString("EVIOCGNAME", eviocgname) }

// Phys returns the physical topology path (EVIOCGPHYS).
func (d *Device) Phys() (string, error) { return d.ioctlString("EVIOCGPHYS", eviocgphys) }

// Uniq returns the unique identifier (EVIOCGUNIQ); often empty.
func (d *Device) Uniq() (string, error) { return d.ioctlString("EVIOCGUNIQ", eviocguniq) }

// ID returns the device's bus/vendor/product/version identity (EVIOCGID).
func (d *Device) ID() (InputID, error) {
	var id InputID
	if err := d.control(func(fd uintptr) error { return ioctl(fd, eviocgid(), unsafe.Pointer(&id)) }); err != nil {
		return InputID{}, fmt.Errorf("evdev: EVIOCGID %s: %w", d.path, err)
	}
	return id, nil
}

// DriverVersion returns the evdev driver version (EVIOCGVERSION).
//
// The kernel writes a 32-bit int here, so we read into an explicit int32 rather
// than using unix.IoctlGetInt (which targets a word-sized Go int). That keeps it
// correct on big-endian architectures (e.g. s390x), where a 4-byte kernel write
// into an 8-byte int would land in the wrong half — at the cost of one
// unsafe.Pointer, routed through the shared ioctl helper.
func (d *Device) DriverVersion() (int, error) {
	var v int32
	if err := d.control(func(fd uintptr) error { return ioctl(fd, eviocgversion(), unsafe.Pointer(&v)) }); err != nil {
		return 0, fmt.Errorf("evdev: EVIOCGVERSION %s: %w", d.path, err)
	}
	return int(v), nil
}

// CapableTypes returns the event types the device can emit (EVIOCGBIT(0)).
func (d *Device) CapableTypes() ([]EvType, error) {
	bits, err := d.queryBits(0, (int(EV_MAX)+8)/8)
	if err != nil {
		return nil, err
	}
	var out []EvType
	forEachSetBit(bits, func(code int) { out = append(out, EvType(code)) })
	return out, nil
}

// CapableCodes returns the codes the device supports for the given event type
// (EVIOCGBIT(t)).
func (d *Device) CapableCodes(t EvType) ([]EvCode, error) {
	bits, err := d.queryBits(uintptr(t), capBufBytes)
	if err != nil {
		return nil, err
	}
	var out []EvCode
	forEachSetBit(bits, func(code int) { out = append(out, EvCode(code)) })
	return out, nil
}

// HasCode reports whether the device supports code c for event type t.
func (d *Device) HasCode(t EvType, c EvCode) (bool, error) {
	bits, err := d.queryBits(uintptr(t), capBufBytes)
	if err != nil {
		return false, err
	}
	idx := int(c) / 8
	if idx >= len(bits) {
		return false, nil
	}
	return bits[idx]&(1<<(uint(c)%8)) != 0, nil
}

// IsKeyboard reports whether the device looks like a real keyboard: it emits
// EV_KEY events and has the alphabetic keys plus space. This distinguishes
// keyboards from mice (which also use EV_KEY, but only for BTN_* codes).
func (d *Device) IsKeyboard() (bool, error) {
	for _, c := range []EvCode{KEY_A, KEY_Z, KEY_SPACE} {
		ok, err := d.HasCode(EV_KEY, c)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// CapableProps returns the device's properties (EVIOCGPROP) — hints about how
// the device behaves, such as INPUT_PROP_POINTER for an indirect pointer (mouse)
// or INPUT_PROP_DIRECT for a touchscreen. A node may report none; that does not
// mean it is unused, only that the driver set no property bits.
func (d *Device) CapableProps() ([]InputProp, error) {
	size := (int(INPUT_PROP_MAX) + 8) / 8
	buf := make([]byte, size)
	var n int
	err := d.control(func(fd uintptr) error {
		var e error
		n, e = ioctlBuf(fd, eviocgprop(uintptr(size)), buf)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("evdev: EVIOCGPROP %s: %w", d.path, err)
	}
	if n < size {
		buf = buf[:n]
	}
	var out []InputProp
	forEachSetBit(buf, func(code int) { out = append(out, InputProp(code)) })
	return out, nil
}

// Grab requests exclusive access to the device (EVIOCGRAB). While grabbed, the
// device's events are delivered only to this open file and are withheld from all
// other readers, including the rest of the system — essential for a remapper
// that re-emits a transformed event stream. The grab is released by Ungrab or
// when the device is closed. Grabbing an already-grabbed device fails with EBUSY.
func (d *Device) Grab() error {
	if err := d.control(func(fd uintptr) error {
		return unix.IoctlSetInt(int(fd), uint(eviocgrab()), 1)
	}); err != nil {
		return fmt.Errorf("evdev: EVIOCGRAB %s: %w", d.path, err)
	}
	return nil
}

// Ungrab releases an exclusive grab previously taken with Grab. It is safe to
// call on a device that is not grabbed.
func (d *Device) Ungrab() error {
	if err := d.control(func(fd uintptr) error {
		return unix.IoctlSetInt(int(fd), uint(eviocgrab()), 0)
	}); err != nil {
		return fmt.Errorf("evdev: EVIOCGRAB(0) %s: %w", d.path, err)
	}
	return nil
}

// queryBits fetches a capability bitmask of size bytes for the given event type
// via EVIOCGBIT, returning only the bytes the kernel actually wrote.
func (d *Device) queryBits(ev uintptr, size int) ([]byte, error) {
	buf := make([]byte, size)
	var n int
	err := d.control(func(fd uintptr) error {
		var e error
		n, e = ioctlBuf(fd, eviocgbit(ev, uintptr(size)), buf)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("evdev: EVIOCGBIT(0x%x) %s: %w", ev, d.path, err)
	}
	if n < size {
		return buf[:n], nil
	}
	return buf, nil
}

// ioctlString runs a string-returning ioctl (EVIOCGNAME/PHYS/UNIQ) and trims
// the trailing NUL. name is the request's symbolic name, used for error context.
func (d *Device) ioctlString(name string, req func(uintptr) uintptr) (string, error) {
	buf := make([]byte, 256)
	var n int
	err := d.control(func(fd uintptr) error {
		var e error
		n, e = ioctlBuf(fd, req(uintptr(len(buf))), buf)
		return e
	})
	if err != nil {
		return "", fmt.Errorf("evdev: %s %s: %w", name, d.path, err)
	}
	if n <= 0 {
		return "", nil
	}
	return string(bytes.TrimRight(buf[:n], "\x00")), nil
}

// forEachSetBit calls fn with the index of each set bit in buf (LSB first).
func forEachSetBit(buf []byte, fn func(code int)) {
	for i, b := range buf {
		for bit := 0; bit < 8; bit++ {
			if b&(1<<uint(bit)) != 0 {
				fn(i*8 + bit)
			}
		}
	}
}
