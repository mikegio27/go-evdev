# go-evdev

`go-evdev` is a Go library for accessing the Linux [evdev](https://www.kernel.org/doc/html/latest/input/input.html)
input subsystem. It opens `/dev/input/event*` devices directly and uses ioctls
to query device identity and capabilities, then reads input events off the
device — the standard, idiomatic way to talk to evdev.

It's intended as a foundation for input monitoring and remapping tools.

**Note:** accessing `/dev/input` requires elevated privileges or membership in
the `input` group.

## Features

- Open devices and read decoded `InputEvent`s (`Open`, `ReadOne`, `Read`).
- Query identity: `Name`, `Phys`, `Uniq`, `ID` (bus/vendor/product/version), `DriverVersion`.
- Query capabilities: `CapableTypes`, `CapableCodes`, `HasCode`, `IsKeyboard`.
- Discover devices: `ListDevicePaths`, `ListDevices`, `ListKeyboards`.
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

## Runnable examples

- `examples/lsinput` — list devices with identity and supported event types.
- `examples/monitor` — stream events from one device.

```sh
sudo go run ./examples/lsinput
sudo go run ./examples/monitor /dev/input/event0
```

## Regenerating event codes

`codes.go` is generated from the kernel headers and checked into the repo, so
consumers never need them. To regenerate (maintainers only — requires the Linux
headers installed, e.g. `linux-headers-$(uname -r)`):

```sh
go generate ./...
```

## Roadmap

- Exclusive device grabbing (`EVIOCGRAB`).
- Virtual device creation and event injection via `uinput`.
- A higher-level remap helper that grabs a source device and re-emits events.

## License

MIT — see [LICENSE.md](LICENSE.md).
