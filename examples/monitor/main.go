// monitor prints events from a single evdev device. Run with privileges:
//
//	sudo go run ./examples/monitor /dev/input/eventX
//
// Press Ctrl-C to stop.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s /dev/input/eventX\n", os.Args[0])
		os.Exit(2)
	}

	d, err := evdev.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer d.Close()

	name, _ := d.Name()
	fmt.Printf("monitoring %s (%q) — press Ctrl-C to stop\n", d.Path(), name)

	for {
		ev, err := d.ReadOne()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Fprintln(os.Stderr, "read:", err)
			os.Exit(1)
		}
		fmt.Printf("%d.%06d  %s\n", ev.Time.Sec, ev.Time.Usec, ev)
	}
}
