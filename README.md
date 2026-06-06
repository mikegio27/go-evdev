# go-evdev

`go-evdev` is a Go library for accessing the Linux [evdev](https://www.kernel.org/doc/html/latest/input/input.html)
input subsystem. It opens `/dev/input/event*` devices directly and uses ioctls
to query device identity and capabilities, then reads input events off the
device — the standard, idiomatic way to talk to evdev.

It's intended as a foundation for input monitoring and remapping tools: it can
read and grab real devices, inject events through `uinput`, and watch for devices
being hotplugged — everything a remapper needs.

**Note:** accessing `/dev/input` requires elevated privileges or membership in
the `input` group. Creating virtual devices additionally needs write access to
`/dev/uinput` (root, or a seat user where logind grants it via `uaccess`).

## Features

- Open devices and read decoded `InputEvent`s (`Open`, `ReadOne`, `Read`).
- Query identity: `Name`, `Phys`, `Uniq`, `ID` (bus/vendor/product/version), `DriverVersion`.
- Query capabilities: `CapableTypes`, `CapableCodes`, `HasCode`, `CapableProps`, `IsKeyboard`.
- Discover devices: `ListDevicePaths`, `ListDevices`, `ListKeyboards`.
- Grab a device exclusively: `Grab`, `Ungrab` (`EVIOCGRAB`).
- Create virtual devices and inject events via `uinput`: `CreateVirtualDevice`,
  `WriteEvent`, `Sync`, plus `CapabilitiesOf` to mirror a real device.
- Remap a device with one function: `Remapper` wraps the grab → transform →
  inject loop; a `MapFunc` returns the events to emit (rebind, suppress, macro).
- Watch for devices being plugged in and removed: `NewWatcher`, `DeviceEvent`.
- Generated event-code constants (`EV_*`, `KEY_*`, `BTN_*`, `REL_*`, `ABS_*`, …)
  with name lookups (`CodeName`, `EvCodeByName`, `EvTypeByName`) — **no kernel
  headers needed** at build or run time.

## Installation

```sh
go get github.com/mikegio27/go-evdev
```

## Usage

### Monitor a single device

```go
package main

import (
	"fmt"

	evdev "github.com/mikegio27/go-evdev"
)

func main() {
	d, err := evdev.Open("/dev/input/event0")
	if err != nil {
		panic(err)
	}
	defer d.Close()

	for {
		ev, err := d.ReadOne()
		if err != nil {
			panic(err)
		}
		fmt.Println(ev) // e.g. "EV_KEY KEY_A 1"
	}
}
```

### List devices and inspect capabilities

```go
devices, _ := evdev.ListDevices()
for _, d := range devices {
	defer d.Close()
	name, _ := d.Name()
	id, _ := d.ID()
	kbd, _ := d.IsKeyboard()
	fmt.Printf("%s: %q bus=%s keyboard=%v\n", d.Path(), name, id.BusType, kbd)
}
```

### Monitor all keyboards concurrently

```go
keyboards, _ := evdev.ListKeyboards()
for _, d := range keyboards {
	go func(d *evdev.Device) {
		defer d.Close()
		for {
			ev, err := d.ReadOne()
			if err != nil {
				return
			}
			if ev.Type == evdev.EV_KEY {
				fmt.Printf("%s: %s %d\n", d.Path(), ev.CodeName(), ev.Value)
			}
		}
	}(d)
}
select {} // block forever
```

### Remap a device

`Remapper` grabs the source exclusively, mirrors its capabilities onto a virtual
device, and runs the read → transform → inject loop. You just write the mapping:
return one event (rebind), none (suppress), or many (a macro/combo).

```go
src, _ := evdev.Open("/dev/input/event0")
defer src.Close()

swap := func(ev evdev.InputEvent) []evdev.InputEvent {
	if ev.Type == evdev.EV_KEY && ev.Code == evdev.KEY_CAPSLOCK {
		ev.Code = evdev.KEY_ESC                  // Caps Lock -> Escape
	}
	return []evdev.InputEvent{ev}                // SYN frames pass through automatically
}

rm, _ := evdev.NewRemapper(src, swap)            // use WithExtraKeys(...) to emit keys
defer rm.Close()                                 // the source lacks (e.g. mouse -> Ctrl+C)
rm.Run()                                          // blocks until the source ends
```

### Watch for hotplugged devices

```go
w, _ := evdev.NewWatcher()                       // create before listing, to not miss races
defer w.Close()

for ev := range w.Events() {
	switch ev.Action {
	case evdev.DeviceAdded:
		fmt.Println("plugged in:", ev.Path)      // e.g. a swapped mouse
	case evdev.DeviceRemoved:
		fmt.Println("removed:", ev.Path)
	}
}
```

## Runnable examples

- `examples/lsinput` — list devices with identity and supported event types.
- `examples/monitor` — stream events from one or more devices (no args = all
  readable devices; pass `-grab` to take them exclusively).
- `examples/vkbd` — create a virtual keyboard via uinput and type a message.
- `examples/watch` — print devices as they are plugged in and removed.
- `examples/remap` — grab a keyboard and re-emit it with Caps Lock ↔ Escape
  swapped (the capstone read → grab → transform → inject loop).

```sh
sudo go run ./examples/lsinput
sudo go run ./examples/monitor                     # every device, labeled by node
sudo go run ./examples/monitor /dev/input/event0
sudo go run ./examples/monitor -grab /dev/input/event0
sudo go run ./examples/vkbd                        # types into the focused field
sudo go run ./examples/watch                       # unplug/replug a device to see it
sudo go run ./examples/remap /dev/input/event0     # grabs the device — see its warning
```

A single physical device often exposes several `event*` nodes (e.g. a gaming
mouse: movement and clicks on its `EV_REL` node, media keys on a separate
keyboard-emulation node). Running `monitor` with no arguments shows which node
carries what.

## Regenerating event codes

`codes.go` is generated from the kernel headers and checked into the repo, so
consumers never need them. To regenerate (maintainers only — requires the Linux
headers installed, e.g. `linux-headers-$(uname -r)`):

```sh
go generate ./...
```

## Roadmap

The read/grab/inject/watch/remap building blocks are in place (see `Remapper`
and `examples/remap`). Possible future addition:

- `EV_ABS` output from virtual devices (touchpads, joysticks, tablets), via
  `UI_ABS_SETUP`. Mice and keyboards (`EV_KEY`/`EV_REL`) are already covered.

## License

MIT — see [LICENSE.md](LICENSE.md).
