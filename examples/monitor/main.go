// monitor prints events from evdev devices. Run with privileges:
//
//	sudo go run ./examples/monitor                       # all devices
//	sudo go run ./examples/monitor /dev/input/eventX     # one or more nodes
//	sudo go run ./examples/monitor -grab /dev/input/eventX
//
// With no path arguments it monitors every readable device, which is the easiest
// way to discover which node carries a given input (e.g. a gaming mouse exposes
// several nodes — movement and clicks are on the EV_REL node, not its
// keyboard-emulation node). Each line is prefixed with the source device.
//
// With -grab each device is taken exclusively, so its events go only to this
// program. Be careful grabbing all devices: it captures the keyboard you would
// use to stop the program. Press Ctrl-C to stop.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	grab := flag.Bool("grab", false, "grab each device exclusively (EVIOCGRAB)")
	flag.Parse()

	devices, err := openDevices(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Fprintln(os.Stderr, "no readable devices (try sudo, or pass a path)")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, d := range devices {
		if *grab {
			if err := d.Grab(); err != nil {
				fmt.Fprintln(os.Stderr, "grab:", err)
				d.Close()
				continue
			}
			defer d.Ungrab()
		}
		name, _ := d.Name()
		fmt.Printf("monitoring %s (%q) grab=%v\n", d.Path(), name, *grab)

		wg.Add(1)
		go func(d *evdev.Device) {
			defer wg.Done()
			defer d.Close()
			monitor(d)
		}(d)
	}
	fmt.Println("— press Ctrl-C to stop —")
	wg.Wait()
}

// openDevices opens the given paths, or every readable device when paths is empty.
func openDevices(paths []string) ([]*evdev.Device, error) {
	if len(paths) == 0 {
		return evdev.ListDevices()
	}
	var devices []*evdev.Device
	for _, p := range paths {
		d, err := evdev.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", p, err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// monitor reads events from one device until it errors or disappears, printing
// each prefixed with the device's node name.
func monitor(d *evdev.Device) {
	tag := filepath.Base(d.Path())
	for {
		ev, err := d.ReadOne()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintf(os.Stderr, "%s: read: %v\n", tag, err)
			}
			return
		}
		fmt.Printf("%-8s %d.%06d  %s\n", tag, ev.Time.Sec, ev.Time.Usec, ev)
	}
}
