package evdev

import "testing"

func TestKnownConstants(t *testing.T) {
	tests := []struct {
		name string
		got  uint16
		want uint16
	}{
		{"EV_SYN", uint16(EV_SYN), 0x00},
		{"EV_KEY", uint16(EV_KEY), 0x01},
		{"KEY_A", uint16(KEY_A), 30},
		{"KEY_ENTER", uint16(KEY_ENTER), 28},
		{"KEY_ESC", uint16(KEY_ESC), 1},
		{"BTN_LEFT", uint16(BTN_LEFT), 0x110},
		{"KEY_MAX", uint16(KEY_MAX), 0x2ff},
		{"BUS_USB", uint16(BUS_USB), 0x03},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestCodeName(t *testing.T) {
	if got := CodeName(EV_KEY, KEY_A); got != "KEY_A" {
		t.Errorf("CodeName(EV_KEY, KEY_A) = %q, want KEY_A", got)
	}
	if got := CodeName(EV_REL, REL_X); got != "REL_X" {
		t.Errorf("CodeName(EV_REL, REL_X) = %q, want REL_X", got)
	}
	// Unknown code falls back to a typed numeric form.
	if got := CodeName(EV_KEY, 0xfff); got != "KEY_?(0xfff)" {
		t.Errorf("CodeName fallback = %q, want KEY_?(0xfff)", got)
	}
}

func TestNameLookups(t *testing.T) {
	if c, ok := EvCodeByName("KEY_A"); !ok || c != KEY_A {
		t.Errorf("EvCodeByName(KEY_A) = %d,%v want %d,true", c, ok, KEY_A)
	}
	if c, ok := EvCodeByName("BTN_LEFT"); !ok || c != BTN_LEFT {
		t.Errorf("EvCodeByName(BTN_LEFT) = %d,%v want %d,true", c, ok, BTN_LEFT)
	}
	if _, ok := EvCodeByName("KEY_NOPE"); ok {
		t.Error("EvCodeByName(KEY_NOPE) = ok, want not ok")
	}
	if ty, ok := EvTypeByName("EV_KEY"); !ok || ty != EV_KEY {
		t.Errorf("EvTypeByName(EV_KEY) = %d,%v want %d,true", ty, ok, EV_KEY)
	}
}

func TestStringers(t *testing.T) {
	if got := EV_KEY.String(); got != "EV_KEY" {
		t.Errorf("EV_KEY.String() = %q, want EV_KEY", got)
	}
	if got := BUS_USB.String(); got != "BUS_USB" {
		t.Errorf("BUS_USB.String() = %q, want BUS_USB", got)
	}
}
