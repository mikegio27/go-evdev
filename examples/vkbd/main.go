// vkbd creates a virtual keyboard via uinput and types a short message, then
// removes it. Run with write access to /dev/uinput (root on most systems, or a
// seat user where logind grants access):
//
//	sudo go run ./examples/vkbd
//
// After it starts there is a short delay so the desktop can bind the new device;
// focus a text field during that window to see "hello" typed into it. You can
// also watch the raw events with, in another terminal:
//
//	sudo go run ./examples/monitor
package main

import (
	"fmt"
	"os"
	"time"

	evdev "github.com/mikegio27/go-evdev"
)

// message is the sequence of keys to type: h e l l o, then Enter.
var message = []evdev.EvCode{
	evdev.KEY_H, evdev.KEY_E, evdev.KEY_L, evdev.KEY_L, evdev.KEY_O, evdev.KEY_ENTER,
}

func main() {
	id := evdev.InputID{BusType: evdev.BUS_USB, Vendor: 0x1234, Product: 0x5678, Version: 1}
	v, err := evdev.CreateVirtualDevice("go-evdev virtual keyboard", id, evdev.Capabilities{Keys: message})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer v.Close()

	// The kernel exposes the device immediately, but the desktop input stack
	// needs a moment to notice and start routing its events.
	fmt.Println("virtual keyboard created — focus a text field; typing in 2s...")
	time.Sleep(2 * time.Second)

	for _, key := range message {
		if err := typeKey(v, key); err != nil {
			fmt.Fprintln(os.Stderr, "type:", err)
			os.Exit(1)
		}
		time.Sleep(40 * time.Millisecond)
	}
	fmt.Println("done")
}

// typeKey presses and releases one key, flushing each with a SYN_REPORT.
func typeKey(v *evdev.VirtualDevice, code evdev.EvCode) error {
	if err := v.WriteEvent(evdev.EV_KEY, code, 1); err != nil {
		return err
	}
	if err := v.Sync(); err != nil {
		return err
	}
	if err := v.WriteEvent(evdev.EV_KEY, code, 0); err != nil {
		return err
	}
	return v.Sync()
}
