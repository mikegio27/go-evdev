# go-evdev

`go-evdev` is a Go library for monitoring input devices on a Linux system. It provides functionality to monitor single or multiple input devices, read key events, and generate key maps from Linux header files.

## Installation

To install the library, use `go get`:

```sh
go get github.com/mikegio27/go-evdev
```

## Usage

### Monitoring a Single Device

To monitor a single device, use the `MonitorSingleDevice` function:

```go
package main

import (
    "fmt"
    "github.com/mikegio27/go-evdev"
)

func main() {
    devicePath := "/dev/input/event0" // or extract from InputDevice method
    dataChan, cancel, err := evdev.MonitorSingleDevice(devicePath)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer cancel()

    for event := range dataChan {
        fmt.Println("Event:", event)
    }
}
```

### Monitoring Keyboard Devices

```go
func main() {
    devices, err := evdev.InputDevices()
    if err != nil {
        fmt.Println("Error getting input devices:", err)
        return
    }

    keyboardDevices := []evdev.InputDevice{}
    for _, device := range devices {
        if device.IsKeyboard() {
            keyboardDevices = append(keyboardDevices, device)
        }
    }

    if len(keyboardDevices) == 0 {
        fmt.Println("No keyboard devices found.")
        return
    }

    dataChanMap, cancel, err := evdev.MonitorDevices(keyboardDevices)
    if err != nil {
        fmt.Println("Error monitoring devices:", err)
        return
    }
    defer cancel()

    for devicePath, dataChan := range dataChanMap {
        go func(devicePath string, dataChan chan evdev.InputEvent) {
            for event := range dataChan {
                fmt.Printf("Device: %s, Event: %v\n", devicePath, event)
            }
        }(devicePath, dataChan)
    }

    // Keep the main function running
    select {}
}
```

### Monitoring All Devices

To monitor all input devices, use the `MonitorAllDevices` function:

```go
package main

import (
    "fmt"
    "github.com/mikegio27/go-evdev"
)

func main() {
    dataChanMap, cancel, err := evdev.MonitorDevices()
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer cancel()

    for devicePath, dataChan := range dataChanMap {
        go func(devicePath string, dataChan chan evdev.InputEvent) {
            for event := range dataChan {
                fmt.Printf("Device: %s, Event: %v\n", devicePath, event)
            }
        }(devicePath, dataChan)
    }

    // Keep the main function running
    select {}
}
```

### Generating a Key Map

To generate a key map from the Linux `input-event-codes.h` file, use the `GenerateKeyMap` function:

```go
package main

import (
    "fmt"
    "github.com/mikegio27/go-evdev"
)

func main() {
    keyMap := evdev.GenerateKeyMap()
    for code, name := range keyMap {
        fmt.Printf("Code: %d, Name: %s\n", code, name)
    }
}
```

## License

This library is licensed under the MIT License. See the LICENSE file for more details.