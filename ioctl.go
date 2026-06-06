package evdev

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// ioctl request encoding, mirroring the kernel's <asm-generic/ioctl.h> _IOC
// macros. evdev request numbers (EVIOC*) are not exported by x/sys/unix, so we
// compute them here.
const (
	iocNRBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14

	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits

	iocNone  = 0
	iocWrite = 1
	iocRead  = 2

	// evdev ioctls use the 'E' type byte.
	evdevType = 'E'
)

// ioc builds an ioctl request number from direction, type, number, and size.
func ioc(dir, typ, nr, size uintptr) uintptr {
	return dir<<iocDirShift | typ<<iocTypeShift | nr<<iocNRShift | size<<iocSizeShift
}

// ior builds a "read" ioctl request (data flows kernel -> userspace).
func ior(typ, nr, size uintptr) uintptr { return ioc(iocRead, typ, nr, size) }

// iow builds a "write" ioctl request (data flows userspace -> kernel).
func iow(typ, nr, size uintptr) uintptr { return ioc(iocWrite, typ, nr, size) }

// io0 builds a no-argument ioctl request (the _IO macro: no data transfer).
func io0(typ, nr uintptr) uintptr { return ioc(iocNone, typ, nr, 0) }

// evdev request builders.

func eviocgversion() uintptr { return ior(evdevType, 0x01, unsafe.Sizeof(int32(0))) }
func eviocgid() uintptr      { return ior(evdevType, 0x02, unsafe.Sizeof(InputID{})) }

func eviocgname(length uintptr) uintptr { return ioc(iocRead, evdevType, 0x06, length) }
func eviocgphys(length uintptr) uintptr { return ioc(iocRead, evdevType, 0x07, length) }
func eviocguniq(length uintptr) uintptr { return ioc(iocRead, evdevType, 0x08, length) }

// eviocgprop builds EVIOCGPROP, the device-properties bitmask request (backs
// Device.CapableProps over INPUT_PROP_*).
func eviocgprop(length uintptr) uintptr { return ioc(iocRead, evdevType, 0x09, length) }

// eviocgbit builds the request to fetch the capability bitmask for an event
// type. ev == 0 returns the set of supported event types.
func eviocgbit(ev, length uintptr) uintptr {
	return ioc(iocRead, evdevType, 0x20+ev, length)
}

// eviocgrab builds the EVIOCGRAB request (reserved for a future Grab/Ungrab).
func eviocgrab() uintptr { return iow(evdevType, 0x90, unsafe.Sizeof(int32(0))) }

// ioctl issues an ioctl on fd. arg must point to memory of the size encoded in
// req; the caller is responsible for keeping it alive for the call.
func ioctl(fd, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// ioctlBuf issues an ioctl whose argument is a buffer the kernel fills (the
// evdev string and bitmask requests), returning the number of bytes the kernel
// reports writing. It centralizes the unsafe pointer conversion for these
// variable-length requests, for which x/sys/unix offers no typed helper. buf
// must be non-empty.
func ioctlBuf(fd, req uintptr, buf []byte) (int, error) {
	r, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return 0, errno
	}
	return int(r), nil
}
