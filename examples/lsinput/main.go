// lsinput lists every accessible evdev device with its identity, physical
// topology, and the event codes it supports. Run with privileges to see all
// devices:
//
//	sudo go run ./examples/lsinput
//
// The phys line distinguishes the several nodes a single physical device often
// exposes (e.g. .../input0, .../input1), and the per-type code breakdown shows
// what each node can emit — useful when one node never seems to produce events.
package main

import (
	"fmt"
	"os"
	"strings"

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
		phys, _ := d.Phys()
		uniq, _ := d.Uniq()
		id, _ := d.ID()
		keyboard, _ := d.IsKeyboard()
		types, _ := d.CapableTypes()
		props, _ := d.CapableProps()

		fmt.Printf("%s\t%q\n", d.Path(), name)
		fmt.Printf("    bus=%s vendor=%04x product=%04x version=%04x keyboard=%v\n",
			id.BusType, id.Vendor, id.Product, id.Version, keyboard)
		fmt.Printf("    phys=%q uniq=%q\n", phys, uniq)

		if len(props) > 0 {
			propNames := make([]string, len(props))
			for i, p := range props {
				propNames[i] = p.String()
			}
			fmt.Printf("    props: %s\n", strings.Join(propNames, " "))
		}

		for _, t := range types {
			codes, _ := d.CapableCodes(t)
			names := make([]string, len(codes))
			for i, c := range codes {
				names[i] = evdev.CodeName(t, c)
			}
			fmt.Printf("    %s (%d): %s\n", t, len(codes), strings.Join(names, " "))
		}
	}
}
