package evdev

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// uinputPath is the kernel's uinput control device, used to create virtual
// input devices that inject events as if from real hardware.
const uinputPath = "/dev/uinput"

// uinputMaxNameSize matches UINPUT_MAX_NAME_SIZE in <linux/uinput.h>.
const uinputMaxNameSize = 80

// uinputSetup mirrors struct uinput_setup, the argument to UI_DEV_SETUP.
type uinputSetup struct {
	ID           InputID
	Name         [uinputMaxNameSize]byte
	FFEffectsMax uint32
}

// uinput request builders (type byte 'U'); see <linux/uinput.h>.
const uinputType = 'U'

func uiDevCreate() uintptr  { return io0(uinputType, 1) }
func uiDevDestroy() uintptr { return io0(uinputType, 2) }
func uiDevSetup() uintptr   { return iow(uinputType, 3, unsafe.Sizeof(uinputSetup{})) }

func uiSetEvbit() uintptr   { return iow(uinputType, 100, unsafe.Sizeof(int32(0))) }
func uiSetKeybit() uintptr  { return iow(uinputType, 101, unsafe.Sizeof(int32(0))) }
func uiSetRelbit() uintptr  { return iow(uinputType, 102, unsafe.Sizeof(int32(0))) }
func uiSetMscbit() uintptr  { return iow(uinputType, 104, unsafe.Sizeof(int32(0))) }
func uiSetPropbit() uintptr { return iow(uinputType, 110, unsafe.Sizeof(int32(0))) }

// Capabilities describes what a VirtualDevice can emit. Enable the event types
// and codes you intend to write: the kernel drops events whose code was not
// registered before the device was created. CapabilitiesOf copies these from a
// real device.
//
// EV_ABS axes (joysticks, touch) are not yet supported; they need per-axis
// ranges via UI_ABS_SETUP.
type Capabilities struct {
	Keys  []EvCode    // EV_KEY codes (keyboard keys and BTN_* buttons)
	Rels  []EvCode    // EV_REL codes (relative axes: REL_X, REL_WHEEL, ...)
	Mscs  []EvCode    // EV_MSC codes (e.g. MSC_SCAN)
	Props []InputProp // device properties (INPUT_PROP_*)
}

// VirtualDevice is a uinput-backed input device. Events written to it are
// injected into the system as if produced by real hardware. Close destroys it.
type VirtualDevice struct {
	f *os.File
}

// CapabilitiesOf reads a real device's capabilities so a VirtualDevice can
// mirror it — the basis for a remapper that grabs a source device and re-emits
// a transformed stream. EV_ABS axes are not copied (see Capabilities).
func CapabilitiesOf(d *Device) (Capabilities, error) {
	var caps Capabilities
	var err error
	if caps.Keys, err = d.CapableCodes(EV_KEY); err != nil {
		return Capabilities{}, err
	}
	if caps.Rels, err = d.CapableCodes(EV_REL); err != nil {
		return Capabilities{}, err
	}
	if caps.Mscs, err = d.CapableCodes(EV_MSC); err != nil {
		return Capabilities{}, err
	}
	if caps.Props, err = d.CapableProps(); err != nil {
		return Capabilities{}, err
	}
	return caps, nil
}

// CreateVirtualDevice creates and registers a uinput device with the given name,
// identity, and capabilities. The returned device is live; write events with
// WriteEvent/Write and flush each batch with Sync. The caller must Close it.
//
// Requires write access to /dev/uinput (root, or membership in a group with
// access plus a udev rule).
func CreateVirtualDevice(name string, id InputID, caps Capabilities) (*VirtualDevice, error) {
	f, err := os.OpenFile(uinputPath, os.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("evdev: open %s: %w", uinputPath, err)
	}
	v := &VirtualDevice{f: f}

	// The kernel requires every event type and code be registered before the
	// device is created.
	if err := v.enable(caps); err != nil {
		f.Close()
		return nil, err
	}

	setup := uinputSetup{ID: id}
	copyCName(setup.Name[:], name)
	if err := ioctl(f.Fd(), uiDevSetup(), unsafe.Pointer(&setup)); err != nil {
		f.Close()
		return nil, fmt.Errorf("evdev: UI_DEV_SETUP: %w", err)
	}
	if err := ioctl(f.Fd(), uiDevCreate(), nil); err != nil {
		f.Close()
		return nil, fmt.Errorf("evdev: UI_DEV_CREATE: %w", err)
	}
	return v, nil
}

// enable registers each capability bit with the kernel via the UI_SET_* ioctls,
// which take the type/code as a scalar argument.
func (v *VirtualDevice) enable(caps Capabilities) error {
	fd := int(v.f.Fd())
	set := func(req uintptr, val int) error { return unix.IoctlSetInt(fd, uint(req), val) }

	enableType := func(t EvType, setCode uintptr, codes []EvCode) error {
		if len(codes) == 0 {
			return nil
		}
		if err := set(uiSetEvbit(), int(t)); err != nil {
			return fmt.Errorf("evdev: UI_SET_EVBIT %s: %w", t, err)
		}
		for _, c := range codes {
			if err := set(setCode, int(c)); err != nil {
				return fmt.Errorf("evdev: enable %s: %w", CodeName(t, c), err)
			}
		}
		return nil
	}

	if err := enableType(EV_KEY, uiSetKeybit(), caps.Keys); err != nil {
		return err
	}
	if err := enableType(EV_REL, uiSetRelbit(), caps.Rels); err != nil {
		return err
	}
	if err := enableType(EV_MSC, uiSetMscbit(), caps.Mscs); err != nil {
		return err
	}
	for _, p := range caps.Props {
		if err := set(uiSetPropbit(), int(p)); err != nil {
			return fmt.Errorf("evdev: UI_SET_PROPBIT %s: %w", p, err)
		}
	}
	return nil
}

// WriteEvent injects a single event. Call Sync after writing a batch to deliver
// it as one atomic packet.
func (v *VirtualDevice) WriteEvent(t EvType, c EvCode, value int32) error {
	return v.Write(InputEvent{Type: t, Code: c, Value: value})
}

// Write injects a raw event. The Time field is ignored — the kernel timestamps
// emitted events itself.
func (v *VirtualDevice) Write(ev InputEvent) error {
	ev.Time = unix.Timeval{}
	b := (*[sizeofInputEvent]byte)(unsafe.Pointer(&ev))[:]
	if _, err := v.f.Write(b); err != nil {
		return fmt.Errorf("evdev: write event: %w", err)
	}
	return nil
}

// Sync emits EV_SYN/SYN_REPORT, flushing events written since the last Sync.
func (v *VirtualDevice) Sync() error {
	return v.WriteEvent(EV_SYN, SYN_REPORT, 0)
}

// Close destroys the virtual device and closes the control file. It is safe to
// call more than once.
func (v *VirtualDevice) Close() error {
	if v.f == nil {
		return nil
	}
	derr := ioctl(v.f.Fd(), uiDevDestroy(), nil)
	cerr := v.f.Close()
	v.f = nil
	if derr != nil {
		return fmt.Errorf("evdev: UI_DEV_DESTROY: %w", derr)
	}
	return cerr
}

// copyCName copies s into a fixed C char array, guaranteeing NUL termination.
// dst is assumed zero-filled, so a short copy is already terminated.
func copyCName(dst []byte, s string) {
	if n := copy(dst, s); n == len(dst) {
		dst[len(dst)-1] = 0
	}
}
