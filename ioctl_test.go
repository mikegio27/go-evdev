package evdev

import "testing"

// These expected values come from expanding the kernel's _IOR/_IOW/_IOC macros
// in <linux/input.h>; they are the authoritative cross-check on our encoding.
func TestIoctlEncoding(t *testing.T) {
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"EVIOCGVERSION", eviocgversion(), 0x80044501},
		{"EVIOCGID", eviocgid(), 0x80084502},
		{"EVIOCGRAB", eviocgrab(), 0x40044590},
		{"EVIOCGNAME(256)", eviocgname(256), 0x81004506},
		{"EVIOCGPHYS(256)", eviocgphys(256), 0x81004507},
		{"EVIOCGBIT(0,8)", eviocgbit(0, 8), 0x80084520},
		{"EVIOCGBIT(EV_KEY,96)", eviocgbit(uintptr(EV_KEY), 96), 0x80604521},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
