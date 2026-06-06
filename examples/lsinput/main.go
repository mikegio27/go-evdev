// lsinput lists every accessible evdev device with its identity and the event
// types it supports. Run with privileges to see all devices:
//
//	sudo go run ./examples/lsinput
package main

import (
	"fmt"
	"os"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	devices, err := evdev.ListDevices()
	if err != nil {
		fmt.Fprintln(os.Stderr, "list devices:", err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Println("no accessible input devices (try running with sudo)")
		return
	}

	for _, d := range devices {
		defer d.Close()

		name, _ := d.Name()
		id, _ := d.ID()
		keyboard, _ := d.IsKeyboard()
		types, _ := d.CapableTypes()

		fmt.Printf("%s\t%q\n", d.Path(), name)
		fmt.Printf("    bus=%s vendor=%04x product=%04x version=%04x keyboard=%v\n",
			id.BusType, id.Vendor, id.Product, id.Version, keyboard)

		fmt.Printf("    events:")
		for _, t := range types {
			fmt.Printf(" %s", t)
		}
		fmt.Println()
	}
}
