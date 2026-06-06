package evdev

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// InputEvent mirrors the kernel's struct input_event. Its in-memory layout
// matches the bytes read from a /dev/input/event* device, so it can be decoded
// directly. The Time field uses unix.Timeval, which is architecture-correct.
type InputEvent struct {
	Time  unix.Timeval
	Type  EvType
	Code  EvCode
	Value int32
}

// sizeofInputEvent is the size of one struct input_event record on this
// architecture (24 bytes on 64-bit, 16 on 32-bit).
const sizeofInputEvent = int(unsafe.Sizeof(InputEvent{}))

// When returns the event timestamp as a time.Time.
func (e InputEvent) When() time.Time {
	return time.Unix(e.Time.Sec, e.Time.Usec*1000)
}

// CodeName returns the symbolic name of this event's code within its type's
// namespace (e.g. "KEY_A").
func (e InputEvent) CodeName() string {
	return CodeName(e.Type, e.Code)
}

// String renders the event as "TYPE CODE value", e.g. "EV_KEY KEY_A 1".
func (e InputEvent) String() string {
	if e.Type == EV_SYN {
		return fmt.Sprintf("%s %s", e.Type, CodeName(e.Type, e.Code))
	}
	return fmt.Sprintf("%s %s %d", e.Type, e.CodeName(), e.Value)
}
