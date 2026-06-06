package evdev

import (
	"encoding/binary"
	"os"
	"testing"
)

// TestReadOneDecode verifies that ReadOne decodes a raw struct input_event off
// the wire with correct field offsets and endianness, using a pipe instead of
// real hardware. It targets the 64-bit layout (24-byte record); on other
// layouts the byte construction below would differ, so it skips.
func TestReadOneDecode(t *testing.T) {
	if sizeofInputEvent != 24 {
		t.Skipf("test assumes 64-bit input_event layout, got size %d", sizeofInputEvent)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Build a record: time={sec:0x1122334455667788, usec:0x99aabbcc},
	// type=EV_KEY, code=KEY_A, value=1.
	rec := make([]byte, 24)
	binary.LittleEndian.PutUint64(rec[0:8], 0x1122334455667788)
	binary.LittleEndian.PutUint64(rec[8:16], 0x99aabbcc)
	binary.LittleEndian.PutUint16(rec[16:18], uint16(EV_KEY))
	binary.LittleEndian.PutUint16(rec[18:20], uint16(KEY_A))
	binary.LittleEndian.PutUint32(rec[20:24], 1)

	go func() {
		w.Write(rec)
		w.Close()
	}()

	d := &Device{f: r, path: "pipe"}
	ev, err := d.ReadOne()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Time.Sec != 0x1122334455667788 {
		t.Errorf("Sec = %#x, want 0x1122334455667788", ev.Time.Sec)
	}
	if ev.Time.Usec != 0x99aabbcc {
		t.Errorf("Usec = %#x, want 0x99aabbcc", ev.Time.Usec)
	}
	if ev.Type != EV_KEY || ev.Code != KEY_A || ev.Value != 1 {
		t.Errorf("got type=%s code=%s value=%d, want EV_KEY KEY_A 1", ev.Type, ev.CodeName(), ev.Value)
	}
	if got := ev.String(); got != "EV_KEY KEY_A 1" {
		t.Errorf("String() = %q, want %q", got, "EV_KEY KEY_A 1")
	}
}

// TestForEachSetBit checks the capability bitmask decoder.
func TestForEachSetBit(t *testing.T) {
	// bit 1 in byte 0 (code 1), bit 0 in byte 2 (code 16).
	buf := []byte{0b00000010, 0x00, 0b00000001}
	var got []int
	forEachSetBit(buf, func(code int) { got = append(got, code) })
	want := []int{1, 16}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("forEachSetBit = %v, want %v", got, want)
	}
}

// TestListDevicePaths just ensures discovery doesn't error on this host; the
// result may legitimately be empty (e.g. in a sandbox).
func TestListDevicePaths(t *testing.T) {
	if _, err := ListDevicePaths(); err != nil {
		t.Fatalf("ListDevicePaths: %v", err)
	}
}
