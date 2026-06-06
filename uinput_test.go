package evdev

import (
	"os"
	"testing"
	"unsafe"
)

// Expected values come from expanding the kernel's _IO/_IOW macros in
// <linux/uinput.h> (type byte 'U').
func TestUinputEncoding(t *testing.T) {
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"UI_DEV_CREATE", uiDevCreate(), 0x5501},
		{"UI_DEV_DESTROY", uiDevDestroy(), 0x5502},
		{"UI_DEV_SETUP", uiDevSetup(), 0x405c5503},
		{"UI_SET_EVBIT", uiSetEvbit(), 0x40045564},
		{"UI_SET_KEYBIT", uiSetKeybit(), 0x40045565},
		{"UI_SET_RELBIT", uiSetRelbit(), 0x40045566},
		{"UI_SET_MSCBIT", uiSetMscbit(), 0x40045568},
		{"UI_SET_PROPBIT", uiSetPropbit(), 0x4004556e},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// The uinput_setup layout must match the kernel struct exactly, or UI_DEV_SETUP
// misreads the identity and name.
func TestUinputSetupSize(t *testing.T) {
	if got := unsafe.Sizeof(uinputSetup{}); got != 92 {
		t.Errorf("sizeof(uinputSetup) = %d, want 92", got)
	}
}

// TestVirtualDeviceSmoke creates a real virtual keyboard, emits a keypress, and
// tears it down. It needs write access to /dev/uinput (root), so it skips
// otherwise — keeping the suite green for unprivileged/CI runs.
func TestVirtualDeviceSmoke(t *testing.T) {
	f, err := os.OpenFile(uinputPath, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot write %s (need root): %v", uinputPath, err)
	}
	f.Close()

	id := InputID{BusType: BUS_USB, Vendor: 0x1234, Product: 0x5678, Version: 1}
	v, err := CreateVirtualDevice("go-evdev test keyboard", id, Capabilities{Keys: []EvCode{KEY_A}})
	if err != nil {
		t.Fatalf("CreateVirtualDevice: %v", err)
	}
	defer v.Close()

	for _, val := range []int32{1, 0} { // press, release
		if err := v.WriteEvent(EV_KEY, KEY_A, val); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
		if err := v.Sync(); err != nil {
			t.Fatalf("Sync: %v", err)
		}
	}
}
