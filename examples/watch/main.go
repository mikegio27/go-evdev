// watch prints evdev devices as they are plugged in and removed, so you can see
// hotplug events live (e.g. unplug and replug a mouse). Run with privileges:
//
//	sudo go run ./examples/watch
//
// Press Ctrl-C to stop.
package main

import (
	"fmt"
	"os"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	// Create the watcher before listing, so a device that appears during startup
	// is reported rather than missed.
	w, err := evdev.NewWatcher()
	if err != nil {
		fmt.Fprintln(os.Stderr, "watch:", err)
		os.Exit(1)
	}
	defer w.Close()

	if paths, err := evdev.ListDevicePaths(); err == nil {
		fmt.Println("current devices:")
		for _, p := range paths {
			fmt.Printf("  %s\n", p)
		}
	}
	fmt.Println("watching for changes — Ctrl-C to stop")

	for ev := range w.Events() {
		switch ev.Action {
		case evdev.DeviceAdded:
			fmt.Printf("+ added   %s\n", ev.Path)
		case evdev.DeviceRemoved:
			fmt.Printf("- removed %s\n", ev.Path)
		}
	}
	if err := <-w.Errors(); err != nil {
		fmt.Fprintln(os.Stderr, "watch:", err)
		os.Exit(1)
	}
}
