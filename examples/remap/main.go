// remap is the capstone example: it uses evdev.Remapper to grab a keyboard and
// re-emit its events with Caps Lock and Escape swapped, passing everything else
// through. The whole mapping is the swap function; the Remapper handles the
// grab -> read -> transform -> inject loop and teardown.
//
// Run with privileges (input access + write to /dev/uinput):
//
//	sudo go run ./examples/remap /dev/input/eventX
//
// WARNING: this grabs the device exclusively, so while it runs that keyboard's
// keys reach ONLY this program. Point it at a keyboard you are not relying on to
// stop the program, or run it over SSH. Ctrl-C stops it; on exit the grab is
// released and the virtual device destroyed.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s /dev/input/eventX\n", os.Args[0])
		os.Exit(2)
	}

	src, err := evdev.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer src.Close()

	rm, err := evdev.NewRemapper(src, swapCapsEsc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "remapper:", err)
		os.Exit(1)
	}
	defer rm.Close()

	// On Ctrl-C, tear down the remapper and close the source so Run unblocks.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		rm.Close()
		src.Close()
	}()

	name, _ := src.Name()
	fmt.Printf("remapping %q (Caps Lock <-> Escape) — Ctrl-C to stop\n", name)

	if err := rm.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}

// swapCapsEsc swaps Caps Lock and Escape, passing every other event through.
func swapCapsEsc(ev evdev.InputEvent) []evdev.InputEvent {
	if ev.Type == evdev.EV_KEY {
		switch ev.Code {
		case evdev.KEY_CAPSLOCK:
			ev.Code = evdev.KEY_ESC
		case evdev.KEY_ESC:
			ev.Code = evdev.KEY_CAPSLOCK
		}
	}
	return []evdev.InputEvent{ev}
}
